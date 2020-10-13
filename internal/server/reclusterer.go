package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/pkg/errors"
	"io"
	"os"
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
				panic(fmt.Sprintf("写入类数据失败，类数据为%v", center))
			}
		}

		s.logger.Println("正在重新聚类")

	}

	// 将next设置为下一天的启动时间
	now := time.Now()
	now.Add(24 * time.Hour)
	now.Date()
	next := time.Date(now.Year(), now.Month(), now.Day(), int(s.config.ReClusterTime.Hours()), 0, 0, 0, now.Location())
	waitTime := next.Sub(now)

	select {
	case <-ctx.Done():
		s.logger.Println("再聚类线程退出")
		return
	case <-time.After(waitTime):

		waitTime = time.Hour * 24
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
	return nil
}
