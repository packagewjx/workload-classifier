package utils

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sort"
	"strconv"
	"testing"
)

func TestPartition(t *testing.T) {
	arr := []float32{3, 6, 1, 76, 2, 16, 549}
	idx := Partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 5)
	assert.Equal(t, arr[5], float32(76))
	idx = Partition(arr, 0, idx)
	assert.Condition(t, func() (success bool) {
		return idx < 5
	})

	arr = []float32{1, 2, 3}
	idx = Partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 1)
	assert.Equal(t, arr[1], float32(2))

	arr = []float32{4, 2, 1, 3}
	idx = Partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 0)
	assert.Equal(t, arr[0], float32(1))

	arr = []float32{}
	idx = Partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 0)
}

func TestGetSortedPosition(t *testing.T) {
	arr := []float32{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	n := GetSortedPositionValue(arr, 4)
	assert.Equal(t, float32(4), n)

	arr = make([]float32, 10000)
	for i := 0; i < len(arr); i++ {
		arr[i] = rand.Float32() * 10000
	}
	p0 := GetSortedPositionValue(arr, 0)
	p1000 := GetSortedPositionValue(arr, 1000)
	p2000 := GetSortedPositionValue(arr, 2000)
	p5000 := GetSortedPositionValue(arr, 5000)
	p9999 := GetSortedPositionValue(arr, 9999)
	sort.Slice(arr, func(i, j int) bool {
		return arr[i] < arr[j]
	})
	assert.Equal(t, arr[0], p0)
	assert.Equal(t, arr[1000], p1000)
	assert.Equal(t, arr[2000], p2000)
	assert.Equal(t, arr[5000], p5000)
	assert.Equal(t, arr[9999], p9999)
}

func TestWorkloadToContainerData(t *testing.T) {
	record := make([]string, internal.NumSections*internal.NumSectionFields+1)
	for i := 1; i < len(record); i++ {
		record[i] = strconv.FormatFloat(float64(i), 'f', 2, 32)
	}
	record[0] = "test"
	cData, err := RecordToContainerWorkloadData(record)
	assert.NoError(t, err)
	for _, data := range cData.Data {
		assert.NotEqual(t, float32(0), data.CpuAvg)
	}
	assert.Equal(t, float32(1), cData.Data[0].CpuAvg)
	assert.Equal(t, float32(internal.NumSectionFields*internal.NumSections), cData.Data[len(cData.Data)-1].MemP99)
	assert.Equal(t, "test", cData.ContainerId)

	/*
		测试空数据
	*/
	record[internal.NumSections] = ""
	_, err = RecordsToSectionArray(record)
	assert.Error(t, err)
}

func TestWorkloadDataToStringRecord(t *testing.T) {
	data := &internal.ContainerWorkloadData{
		ContainerId: "test",
		Data:        make([]*internal.SectionData, internal.NumSections),
	}

	for i := 0; i < len(data.Data); i++ {
		data.Data[i] = &internal.SectionData{
			CpuAvg: float32(i * internal.NumSectionFields),
			CpuMax: float32(i*internal.NumSectionFields + 1),
			CpuMin: float32(i*internal.NumSectionFields + 2),
			CpuP50: float32(i*internal.NumSectionFields + 3),
			CpuP90: float32(i*internal.NumSectionFields + 4),
			CpuP99: float32(i*internal.NumSectionFields + 5),
			MemAvg: float32(i*internal.NumSectionFields + 6),
			MemMax: float32(i*internal.NumSectionFields + 7),
			MemMin: float32(i*internal.NumSectionFields + 8),
			MemP50: float32(i*internal.NumSectionFields + 9),
			MemP90: float32(i*internal.NumSectionFields + 10),
			MemP99: float32(i*internal.NumSectionFields + 11),
		}
	}

	record := WorkloadDataToStringRecord(data)
	assert.Equal(t, "test", record[0])
	for i := 1; i < len(record); i++ {
		expect := fmt.Sprintf("%.2f", float32(i-1))
		assert.Equal(t, expect, record[i])
	}

}

func TestContainerWorkloadToFloatArray(t *testing.T) {
	data := make([]*internal.ContainerWorkloadData, 1)
	data[0] = &internal.ContainerWorkloadData{
		ContainerId: "test",
		Data: []*internal.SectionData{
			{
				CpuAvg: 1,
				CpuMax: 2,
				CpuMin: 3,
				CpuP50: 4,
				CpuP90: 5,
				CpuP99: 6,
				MemAvg: 7,
				MemMax: 8,
				MemMin: 9,
				MemP50: 10,
				MemP90: 11,
				MemP99: 12,
			},
		},
	}

	array := ContainerWorkloadToFloatArray(data)["test"]
	assert.Equal(t, 12, len(array))
	assert.Equal(t, float32(1), array[0])
	assert.Equal(t, float32(2), array[1])
	assert.Equal(t, float32(3), array[2])
	assert.Equal(t, float32(4), array[3])
	assert.Equal(t, float32(5), array[4])
	assert.Equal(t, float32(6), array[5])
	assert.Equal(t, float32(7), array[6])
	assert.Equal(t, float32(8), array[7])
	assert.Equal(t, float32(9), array[8])
	assert.Equal(t, float32(10), array[9])
	assert.Equal(t, float32(11), array[10])
	assert.Equal(t, float32(12), array[11])
}
