package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/classify"
	"github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/packagewjx/workload-classifier/internal/preprocess"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"github.com/pkg/errors"
	"io"
	"os"
	"reflect"
	"time"
)

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
				panic(fmt.Sprintf("写入类数据失败，类数据为%v，错误为：%v", center, err))
			}
		}
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
		case <-s.executeReCluster:
			err := s.reCluster()
			if err != nil {
				panic(errors.Wrap(err, "再聚类出错"))
			}
		}
	}

}

func readInitialCenter(csvInput io.Reader) ([]*server.ClassMetrics, error) {
	result := make([]*server.ClassMetrics, 0)
	records, err := csv.NewReader(csvInput).ReadAll()
	if err != nil {
		return nil, errors.Wrap(err, "读取CSV数据出错")
	} else if len(records) == 0 {
		return nil, fmt.Errorf("没有读取到任何数据")
	}

	preprocessor := preprocess.Default()
	for i, record := range records {
		if len(record) != core.NumSections*core.NumSectionFields {
			return nil, fmt.Errorf("第%d行数据有问题", i)
		}

		c := &server.ClassMetrics{
			ClassId: uint(i + 1),
			Data:    nil,
		}

		array, err := utils.RecordsToSectionArray(record)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("第%d行数据有问题", i))
		}
		// 对array的数据进行预处理
		temp := &core.ContainerWorkloadData{
			ContainerId: "",
			Data:        array,
		}
		preprocessor.Preprocess(temp)

		c.Data = array

		result = append(result, c)
	}

	return result, nil
}

func (s *serverImpl) reCluster() error {
	s.logger.Println("再聚类开始")

	type dataFeature struct {
		cpuMax float32
		memMax float32
	}

	// 获取并转换数据
	s.logger.Println("正在获取所有应用监控数据")
	dataSource := NewDatabaseDatasource(s.dao.DB())
	rawData, err := datasource.NewDataSourceRawDataReader(dataSource).Read()
	if err != nil {
		return errors.Wrap(err, "读取数据库监控出错")
	}
	workloadData := datasource.ConvertAllRawData(rawData)

	// 由于预处理后真实数据将会丢失，此处保留数据特征
	features := make([]dataFeature, len(workloadData))
	for i, datum := range workloadData {
		cpuMax := float32(0)
		memMax := float32(0)
		for _, data := range datum.Data {
			if data.CpuMax > cpuMax {
				cpuMax = data.CpuMax
			}
			if data.MemMax > memMax {
				memMax = data.MemMax
			}
		}
		features[i].cpuMax = cpuMax
		features[i].memMax = memMax
	}

	preprocessor := preprocess.Default()
	for _, datum := range workloadData {
		preprocessor.Preprocess(datum)
	}
	dataArray := utils.ContainerWorkloadToFloatArray(workloadData)

	// 获取算法实现
	alg := classify.GetAlgorithm(classify.KMeans)
	ctx := &classify.KMeansContext{
		Round: int(s.config.NumRound),
	}

	// 获取类别中心，并加入到dataArray中作为数据的一部分，避免中心变化太大
	s.logger.Println("正在获取聚类中心数据，并加入到数据集中")
	classMetrics, err := s.dao.QueryAllClassMetrics()
	if err != nil {
		return errors.Wrap(err, "查询类别中心时出错")
	}
	for _, metric := range classMetrics {
		dataArray = append(dataArray, utils.SectionDataToFloatArray(metric.Data))
	}

	// 聚类执行
	s.logger.Println("开始执行聚类")
	centers, class := alg.Run(dataArray, int(s.config.NumClass), ctx)
	s.logger.Println("聚类执行完成")

	s.logger.Println("正在保存中心数据")
	for i, center := range centers {
		c := floatArrayToClassMetrics(i+1, center)
		err := s.dao.SaveClassMetrics(c)
		if err != nil {
			return errors.Wrap(err, "保存ClassMetrics时出现错误")
		}
	}

	s.logger.Println("保存新的应用与类别绑定关系")
	for i := 0; i < len(workloadData); i++ {
		a := &server.AppClass{
			AppName: server.AppNameFromContainerId(workloadData[i].ContainerId),
			ClassId: uint(class[i]),
			CpuMax:  features[i].cpuMax,
			MemMax:  features[i].memMax,
		}
		err := s.dao.SaveAppClass(a)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("保存名称空间%s，名称为%s的应用的ClassID出现问题",
				a.AppName.Namespace, a.AppName.Name))
		}
	}

	s.logger.Println("再聚类结束")
	return nil
}

func floatArrayToClassMetrics(id int, data []float32) *server.ClassMetrics {
	result := &server.ClassMetrics{
		ClassId: uint(id),
		Data:    make([]*core.SectionData, core.NumSections),
	}

	for i := 0; i < len(result.Data); i++ {
		result.Data[i] = &core.SectionData{}
		val := reflect.ValueOf(result.Data[i]).Elem()
		for j := 0; j < core.NumSectionFields; j++ {
			val.Field(j).SetFloat(float64(data[i*core.NumSectionFields+j]))
		}
	}

	return result
}
