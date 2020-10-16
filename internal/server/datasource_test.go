package server

import (
	. "github.com/packagewjx/workload-classifier/internal"
	. "github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestDbDatasource_Load(t *testing.T) {
	dao, _ := NewDao(testHost)
	_ = dao.RemoveAppPodMetricsBefore(math.MaxUint64)
	const sectionSize = 10
	testData := make([]*AppPodMetrics, 0, NumSections*sectionSize)
	for i := 0; i < NumSections; i++ {
		for j := 0; j < sectionSize; j++ {
			testData = append(testData, &AppPodMetrics{
				AppName: AppName{
					Name:      "test",
					Namespace: "test",
				},
				Timestamp: uint64(SectionLength*i + j),
				Cpu:       float32((j + 1) * 10),
				Mem:       float32((j + 1) * 10),
			})
		}
	}
	_ = dao.SaveAllAppPodMetrics(testData)

	datasource := NewDatabaseDatasource(dao.DB())
	var r *ContainerMetric
	var err error
	for r, err = datasource.Load(); err == nil; r, err = datasource.Load() {
		assert.NoError(t, err)
		assert.Equal(t, testData[0].AppName.ContainerId(), r.ContainerId)
		assert.NotEqual(t, float32(0), r.Mem)
		assert.NotEqual(t, float32(0), r.Cpu)
	}

	// 使用Reader来测试是否有问题
	ds := NewDatabaseDatasource(dao.DB())
	reader := NewDataSourceRawDataReader(ds)
	data, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(data))
	rawData := data[0]
	assert.Equal(t, testData[0].AppName.ContainerId(), rawData.ContainerId)
	assert.Equal(t, NumSections, len(rawData.Data))
	for _, datum := range rawData.Data {
		assert.Equal(t, sectionSize, len(datum.Cpu))
		assert.Equal(t, sectionSize, len(datum.Mem))
		assert.Equal(t, float32(550), datum.CpuSum)
		assert.Equal(t, float32(550), datum.MemSum)
	}
}
