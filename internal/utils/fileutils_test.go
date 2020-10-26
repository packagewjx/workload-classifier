package utils

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
)

func TestWriteContainerWorkloadHeader(t *testing.T) {
	builder := &strings.Builder{}
	err := WriteContainerWorkloadHeader(builder)
	assert.NoError(t, err)
	split := strings.Split(builder.String(), core.Splitter)
	assert.Equal(t, core.NumSectionFields*core.NumSections+1, len(split))
}

func TestWriteContainerWorkloadData(t *testing.T) {
	builder := &strings.Builder{}
	data := []*core.ContainerWorkloadData{
		{
			ContainerId: "test-1",
			Data:        make([]*core.SectionData, core.NumSections),
		},
		{
			ContainerId: "test-2",
			Data:        make([]*core.SectionData, core.NumSections),
		},
	}
	for i := 0; i < core.NumSections; i++ {
		data[0].Data[i] = &core.SectionData{}
		data[1].Data[i] = &core.SectionData{}
		val := reflect.ValueOf(data[1].Data[i])
		for j := 0; j < core.NumSectionFields; j++ {
			val.Elem().Field(j).SetFloat(float64(i + 1))
		}
	}
	err := WriteContainerWorkloadData(builder, data)
	assert.NoError(t, err)
	lines := strings.Split(builder.String(), "\n")
	assert.Equal(t, 3, len(lines))

	// 验证第一行数据
	record := strings.Split(lines[0], core.Splitter)
	assert.Equal(t, core.NumSections*core.NumSectionFields+1, len(record))
	assert.Equal(t, "test-1", record[0])
	assert.Equal(t, "0.00", record[1])

	// 验证第二行数据
	record = strings.Split(lines[1], core.Splitter)
	assert.Equal(t, core.NumSections*core.NumSectionFields+1, len(record))
	assert.Equal(t, "test-2", record[0])
	assert.Equal(t, "1.00", record[1])
}
