package datasource

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type testDataSource int

func (t *testDataSource) Load() (*ContainerMetric, error) {
	if *t < core.DayLength {
		temp := *t
		*t++
		return &ContainerMetric{
			ContainerId: "test",
			Cpu:         float32(temp%core.SectionLength) / float32(core.SectionLength),
			Mem:         float32(temp%core.SectionLength) / float32(core.SectionLength),
			Timestamp:   uint64(temp),
		}, nil
	}
	return nil, io.EOF
}

func TestDataSourceRawDataReader_Read(t *testing.T) {
	temp := 0
	datasource := (*testDataSource)(&temp)

	read, err := NewDataSourceRawDataReader(datasource).Read()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(read))
	data := read[0]
	assert.Equal(t, "test", data.ContainerId)
	assert.Equal(t, core.NumSections, len(data.Data))
	assert.Equal(t, core.SectionLength, len(data.Data[0].Cpu))
	assert.Equal(t, core.SectionLength, len(data.Data[0].Mem))
}
