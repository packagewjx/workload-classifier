package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/classify"
	"github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/packagewjx/workload-classifier/internal/preprocess"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/pkg/errors"
	"io"
	"os"
	"reflect"
	"strings"
	"time"
)

const NamespaceSplit = "::"

func (s *serverImpl) reClusterer(ctx context.Context) {
	s.logger.Println("再聚类线程启动")

	if s.config.InitialCenterCsvFile != "" {
		s.logger.Println("正在读取中心数据")
		f, err := os.Open(s.config.InitialCenterCsvFile)
		if err != nil {
			panic(fmt.Sprintf("打开文件%s失败", s.config.InitialCenterCsvFile))
		}
		center, err := readInitialCenter(f)
		if err != nil {
			panic(fmt.Sprintf("读取文件%s失败", s.config.InitialCenterCsvFile))
		}

		s.logger.Printf("正在删除数据库的中心数据\n")
		err = s.dao.RemoveAllClassMetrics()
		if err != nil {
			panic(fmt.Sprintf("删除数据库聚类数据失败：%v", err))
		}

		s.logger.Println("正在更新数据库的中心数据")
		for _, center := range center {
			err := s.dao.SaveClassMetrics(center)
			if err != nil {
				panic(fmt.Sprintf("写入类数据失败，类数据为%v", center))
			}
		}

		s.logger.Println("正在重新聚类")

	}

	// 将next设置为下一天的启动时间
	now := time.Now()
	now.Add(24 * time.Hour)
	now.Date()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, int(s.config.ReClusterTime.Hours()), 0, 0, 0, now.Location())
	waitTime := next.Sub(now)

	for {
		next = time.Now().Add(waitTime)
		s.logger.Printf("聚类将于%s执行\n", next.Format("2006-01-02T15:04:05-0700"))
		select {
		case <-ctx.Done():
			s.logger.Println("再聚类线程退出")
			return
		case <-time.After(waitTime):
			waitTime = time.Hour * 24
			err := s.reCluster()
			if err != nil {
				panic(errors.Wrap(err, "再聚类出错"))
			}
		}
	}

}

func readInitialCenter(csvInput io.Reader) ([]*ClassMetrics, error) {
	result := make([]*ClassMetrics, 0)
	records, err := csv.NewReader(csvInput).ReadAll()
	if err != nil {
		return nil, errors.Wrap(err, "读取CSV数据出错")
	} else if len(records) == 0 {
		return nil, fmt.Errorf("没有读取到任何数据")
	}

	for i, record := range records {
		if len(record) != internal.NumSections*internal.NumSectionFields {
			return nil, fmt.Errorf("第%d行数据有问题", i)
		}

		c := &ClassMetrics{
			ClassId: 0,
			Data:    make([]*internal.SectionData, internal.NumSections),
		}

		array, err := utils.RecordsToSectionArray(record)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("第%d行数据有问题", i))
		}
		c.Data = array

		result = append(result, c)
	}

	return result, nil
}

func (s *serverImpl) reCluster() error {
	metrics, err := s.dao.QueryAllAppPodMetrics()
	if err != nil {
		return errors.Wrap(err, "再聚类时，查询监控数据出错")
	}

	workloadFloatMap := preprocessData(metrics)

	alg := classify.GetAlgorithm(classify.KMeans)
	ctx := classify.KMeansContext{
		Round: int(s.config.NumRound),
	}

	// 保存每个位置的ID
	dataArray := make([][]float32, 0, len(workloadFloatMap))
	idArray := make([]string, 0, len(workloadFloatMap))
	for containerId, arr := range workloadFloatMap {
		idArray = append(idArray, containerId)
		dataArray = append(dataArray, arr)
	}

	centers, class := alg.Run(dataArray, int(s.config.NumClass), ctx)

	for i, center := range centers {
		c := floatArrayToClassMetrics(i, center)
		err := s.dao.SaveClassMetrics(c)
		if err != nil {
			return errors.Wrap(err, "保存ClassMetrics时出现错误")
		}
	}

	for idx, id := range idArray {
		split := strings.Split(id, NamespaceSplit)
		a := &AppClass{
			AppName: AppName{
				Name:      split[1],
				Namespace: split[0],
			},
			ClassId: uint(class[idx]),
		}
		err := s.dao.SaveAppClass(a)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("保存名称空间%s，名称为%s的应用的ClassID出现问题",
				a.AppName.Namespace, a.AppName.Name))
		}
	}

	return nil
}

func podMetricsToRawData(podMetricsMap map[string]map[string][]*AppPodMetrics) []*internal.ContainerRawData {
	m := make([]*internal.ContainerRawData, 0)

	for namespace, namespaceMap := range podMetricsMap {
		for appName, metrics := range namespaceMap {
			containerId := namespace + NamespaceSplit + appName
			arr := make([]*internal.RawSectionData, internal.NumSections)
			for i := 0; i < len(arr); i++ {
				arr[i] = &internal.RawSectionData{
					Cpu:    make([]float32, 0),
					Mem:    make([]float32, 0),
					CpuSum: 0,
					MemSum: 0,
				}
			}
			for _, metric := range metrics {
				section := arr[metric.Timestamp%internal.DayLength/internal.SectionLength]
				section.Cpu = append(section.Cpu, metric.Mem)
				section.Mem = append(section.Mem, metric.Mem)
				section.CpuSum += metric.Cpu
				section.MemSum += metric.Mem
			}
			m = append(m, &internal.ContainerRawData{
				ContainerId: containerId,
				Data:        arr,
			})
		}
	}

	return m
}

func preprocessData(podMetricsMap map[string]map[string][]*AppPodMetrics) map[string][]float32 {
	rawData := podMetricsToRawData(podMetricsMap)
	workloadData := datasource.ConvertAllRawData(rawData)
	preprocessor := preprocess.Default()

	for _, datum := range workloadData {
		preprocessor.Preprocess(datum)
	}

	return classify.ContainerWorkloadToFloatArray(workloadData)
}

func floatArrayToClassMetrics(id int, data []float32) *ClassMetrics {
	result := &ClassMetrics{
		ClassId: uint(id),
		Data:    make([]*internal.SectionData, internal.NumSections),
	}

	for i := 0; i < len(result.Data); i++ {
		result.Data[i] = &internal.SectionData{}
		val := reflect.ValueOf(result.Data[i]).Elem()
		for j := 0; j < internal.NumSectionFields; j++ {
			val.Field(j).SetFloat(float64(data[i*internal.NumSectionFields+j]))
		}
	}

	return result
}
