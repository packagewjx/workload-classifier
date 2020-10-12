package alitrace

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestA(t *testing.T) {
	meta, err := LoadContainerMeta("../../test/csv/container_meta.csv")
	if err != nil {
		t.Error(err)
	}

	cnt := 0
	err = SplitContainerUsage("../../test/csv/container_usage_6000000.csv", meta, &cnt)
	if err != nil {
		t.Error(err)
	}
}

func TestPreProcessUsage(t *testing.T) {
	meta, err := LoadContainerMeta("../../test/csv/container_meta.csv")
	if err != nil {
		t.Error(err)
	}

	cnt := int64(0)
	err = PreProcessUsages([]string{"../../test/csv/container_usage_6000000.csv"}, meta, &cnt, false, 4)
	if err != nil {
		t.Error(err)
	}
}

func TestMergeFiles(t *testing.T) {
	err := mergeFile([]string{"../../test/csv/container_meta.csv", "../../test/csv/container_usage_6000000.csv"}, "mergeTest.csv")
	if err != nil {
		t.Error(err)
	}
}

func TestImputeData(t *testing.T) {
	record := make([]string, internal.NumSectionFields*internal.NumSections)

	for si := 0; si < internal.NumSectionFields; si++ {
		for i := 0; i < internal.NumSections; i++ {
			record[i*internal.NumSectionFields+si] = fmt.Sprintf("%d", i)
		}

		for i := 0; i < 4; i++ {
			record[i*internal.NumSectionFields+si] = "NaN"
		}

		record[10*internal.NumSectionFields+si] = "NaN"
		record[11*internal.NumSectionFields+si] = "NaN"

		record[50*internal.NumSectionFields+si] = "NaN"

		record[95*internal.NumSectionFields+si] = "NaN"

		imputeData(record)

		assert.Equal(t, "0.80", record[0*internal.NumSectionFields+si])
		assert.Equal(t, "1.60", record[1*internal.NumSectionFields+si])
		assert.Equal(t, "2.40", record[2*internal.NumSectionFields+si])
		assert.Equal(t, "3.20", record[3*internal.NumSectionFields+si])
		assert.Equal(t, "10.00", record[10*internal.NumSectionFields+si])
		assert.Equal(t, "11.00", record[11*internal.NumSectionFields+si])
		assert.Equal(t, "50.00", record[50*internal.NumSectionFields+si])
		assert.Equal(t, "47.00", record[95*internal.NumSectionFields+si])

		for i := 0; i < internal.NumSections; i++ {
			assert.NotEqual(t, "NaN", record[i*internal.NumSectionFields+si])
		}
	}
}

func TestNormalize(t *testing.T) {
	arr := make([]*internal.ProcessedSectionData, internal.NumSections)
	for i := 0; i < len(arr); i++ {
		arr[i] = &internal.ProcessedSectionData{
			CpuAvg: float32(i),
			CpuMax: float32(i),
			CpuMin: float32(i),
			CpuP50: float32(i),
			CpuP90: float32(i),
			CpuP99: float32(i),
			MemAvg: float32(i),
			MemMax: float32(i),
			MemMin: float32(i),
			MemP50: float32(i),
			MemP90: float32(i),
			MemP99: float32(i),
		}
	}

	normalizeSectionData(arr)

	for i := 0; i < len(arr)-1; i++ {
		assert.Condition(t, func() (success bool) {
			return arr[i].CpuAvg < 1
		})
	}
	assert.Equal(t, float32(1), arr[len(arr)-1].CpuAvg)
}
