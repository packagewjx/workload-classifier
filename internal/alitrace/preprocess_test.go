package alitrace

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestA(t *testing.T) {
	meta, err := LoadContainerMeta("../test/csv/container_meta.csv")
	if err != nil {
		t.Error(err)
	}

	cnt := 0
	err = SplitContainerUsage("../test/csv/container_usage_6000000.csv", meta, &cnt)
	if err != nil {
		t.Error(err)
	}
}

func TestPreProcessUsage(t *testing.T) {
	meta, err := LoadContainerMeta("../test/csv/container_meta.csv")
	if err != nil {
		t.Error(err)
	}

	cnt := int64(0)
	err = PreProcessUsages([]string{"../test/csv/container_usage_6000000.csv"}, meta, &cnt, false, 4)
	if err != nil {
		t.Error(err)
	}
}

func TestMergeFiles(t *testing.T) {
	err := mergeFile([]string{"../test/csv/c_15794.csv", "../test/csv/container_meta.csv", "../test/csv/container_usage_6000000.csv"}, "mergeTest.csv")
	if err != nil {
		t.Error(err)
	}
}

func TestImputeData(t *testing.T) {
	record := make([]string, NumSectionFields*NumSections)

	for si := 0; si < NumSectionFields; si++ {
		for i := 0; i < NumSections; i++ {
			record[i*NumSectionFields+si] = fmt.Sprintf("%d", i)
		}

		for i := 0; i < 4; i++ {
			record[i*NumSectionFields+si] = "NaN"
		}

		record[10*NumSectionFields+si] = "NaN"
		record[11*NumSectionFields+si] = "NaN"

		record[50*NumSectionFields+si] = "NaN"

		record[95*NumSectionFields+si] = "NaN"

		imputeData(record)

		assert.Equal(t, "0.80", record[0*NumSectionFields+si])
		assert.Equal(t, "1.60", record[1*NumSectionFields+si])
		assert.Equal(t, "2.40", record[2*NumSectionFields+si])
		assert.Equal(t, "3.20", record[3*NumSectionFields+si])
		assert.Equal(t, "10.00", record[10*NumSectionFields+si])
		assert.Equal(t, "11.00", record[11*NumSectionFields+si])
		assert.Equal(t, "50.00", record[50*NumSectionFields+si])
		assert.Equal(t, "47.00", record[95*NumSectionFields+si])

		for i := 0; i < NumSections; i++ {
			assert.NotEqual(t, "NaN", record[i*NumSectionFields+si])
		}
	}
}
