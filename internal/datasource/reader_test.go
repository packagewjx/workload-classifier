package datasource

import (
	. "github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

type testDataSource int

func (t *testDataSource) Load() (*ContainerMetric, error) {
	if *t < DayLength {
		temp := *t
		*t++
		return &ContainerMetric{
			ContainerId: "test",
			Cpu:         float32(temp%SectionLength) / float32(SectionLength),
			Mem:         float32(temp%SectionLength) / float32(SectionLength),
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
	assert.Equal(t, NumSections, len(data.Data))
	assert.Equal(t, SectionLength, len(data.Data[0].Cpu))
	assert.Equal(t, SectionLength, len(data.Data[0].Mem))
}
