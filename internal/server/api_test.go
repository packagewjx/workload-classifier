package server

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"os"
	"testing"
)

func TestServerImpl_QueryAppCharacteristics(t *testing.T) {
	// 数据准备
	dao, _ := NewDao("127.0.0.1:3306")
	s := &serverImpl{
		config: &ServerConfig{
			MetricDuration:       0,
			Port:                 0,
			ScrapeInterval:       0,
			ReClusterTime:        0,
			NumClass:             DefaultNumClass,
			NumRound:             DefaultNumRound,
			InitialCenterCsvFile: "",
			MysqlHost:            "",
		},
		dao:              dao,
		logger:           log.New(os.Stdout, "", 0),
		executeReCluster: nil,
	}

	centerFile, _ := os.Open("../../test/csv/centers.csv")
	center, _ := readInitialCenter(centerFile)
	for _, metrics := range center {
		err := s.dao.SaveClassMetrics(metrics)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "准备数据出错，无法保存ClassMetrics")
		}
	}

	const sectionSize = 15
	arr := []*server.AppPodMetrics{}
	for i := 0; i < core.NumSections; i++ {
		for j := 0; j < sectionSize; j++ {
			t := uint64(i*core.SectionLength + 60*j)
			// 一个基本没有负载的应用数据
			arr = append(arr, &server.AppPodMetrics{
				AppName: server.AppName{
					Name:      "low",
					Namespace: "test",
				},
				Timestamp: t,
				Cpu:       rand.Float32(),
				Mem:       rand.Float32(),
			})

			// 一个基本高负载的应用数据
			arr = append(arr, &server.AppPodMetrics{
				AppName: server.AppName{
					Name:      "high",
					Namespace: "test",
				},
				Timestamp: t,
				Cpu:       95 + 5*rand.Float32(),
				Mem:       95 + 5*rand.Float32(),
			})

			// 一个线性增长的应用数据
			arr = append(arr, &server.AppPodMetrics{
				AppName: server.AppName{
					Name:      "linear",
					Namespace: "test",
				},
				Timestamp: t,
				Cpu:       100 * (float32(t) / core.DayLength),
				Mem:       100 * (float32(t) / core.DayLength),
			})
		}
	}

	err := s.dao.SaveAllAppPodMetrics(arr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "造数据错误")
	}

	err = s.reCluster()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "聚类错误")
	}

	low, err := s.QueryAppCharacteristics(server.AppName{
		Name:      "low",
		Namespace: "test",
	})
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(low.SectionData))

	high, err := s.QueryAppCharacteristics(server.AppName{
		Name:      "high",
		Namespace: "test",
	})
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(high.SectionData))

	linear, err := s.QueryAppCharacteristics(server.AppName{
		Name:      "linear",
		Namespace: "test",
	})
	assert.NoError(t, err)
	assert.NotEqual(t, 0, len(linear.SectionData))

	assert.Condition(t, func() (success bool) {
		return low.SectionData[0].CpuAvg != high.SectionData[0].CpuAvg &&
			high.SectionData[0].CpuAvg != linear.SectionData[0].CpuAvg &&
			low.SectionData[0].CpuAvg != linear.SectionData[0].CpuAvg
	})

}
