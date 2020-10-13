package classify

import (
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestOutputResult(t *testing.T) {
	builder := &strings.Builder{}
	data := [][]float32{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}
	err := OutputResult(data, builder, 2)
	assert.NoError(t, err)
	assert.Equal(t, "1.00,2.00,3.00\n4.00,5.00,6.00\n7.00,8.00,9.00\n", builder.String())
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
