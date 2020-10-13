package alitrace

import (
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"strconv"
	"strings"
	"testing"
)

//
//c_11333,m_2041,326620,1,36,,,,0.01,0.01,4
func TestContainerWorkloadReader_Read(t *testing.T) {
	builder := strings.Builder{}
	containerId := "test_1"
	machineId := "m_1"
	record := []string{containerId, machineId, "0", "0", "0", "", "", "", "0", "0", "0"}
	for i := 0; i < internal.NumSections*7; i++ {
		secondStart := i * internal.SectionLength
		for j := 0; j < 10; j++ {
			record[2] = strconv.FormatInt(int64(secondStart+j), 10)
			record[3] = strconv.FormatInt(int64(j), 10)
			record[4] = strconv.FormatInt(int64(j), 10)
			builder.WriteString(strings.Join(record, internal.Splitter) + "\n")
		}
	}
	reader := strings.NewReader(builder.String())

	meta := map[string][]*ContainerMeta{
		containerId: {
			&ContainerMeta{
				ContainerId: containerId,
				MachineId:   machineId,
				Timestamp:   0,
				AppDu:       "app_1",
				Status:      "started",
				CpuRequest:  10,
				CpuLimit:    10,
				MemSize:     1,
			},
		},
	}

	workloadReader := NewContainerWorkloadReader(reader, meta)
	workloads, err := workloadReader.Read()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(workloads))
	data := workloads[0]
	for _, v := range data.Data {
		assert.Equal(t, float32(0), v.CpuMin)
		assert.Equal(t, float32(0), v.MemMin)
		assert.Equal(t, float32(9), v.CpuMax)
		assert.Equal(t, float32(9), v.MemMax)
		assert.Equal(t, float32(4.5), v.CpuAvg)
		assert.Equal(t, float32(4.5), v.MemAvg)
		assert.Equal(t, float32(5), v.CpuP50)
		assert.Equal(t, float32(5), v.MemP50)
	}
}
