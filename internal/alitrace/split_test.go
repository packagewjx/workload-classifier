package alitrace

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

//c_11333,m_2041,326620,1,36,,,,0.01,0.01,4
func TestSplitContainerUsage(t *testing.T) {
	testData := "c_1,m_1,0,0,0,,,,0,0,0\n" +
		"c_2,m_2,0,0,0,,,,0,0,0\n" +
		"c_3,m_3,0,0,0,,,,0,0,0\n"

	reader := strings.NewReader(testData)
	err := SplitContainerUsage(reader, map[string][]*ContainerMeta{
		"c_1": {
			{
				ContainerId: "c_1",
				MachineId:   "m_1",
				AppDu:       "app_1",
				Status:      "started",
			},
		},
		"c_2": {
			{
				ContainerId: "c_2",
				MachineId:   "m_2",
				AppDu:       "app_1",
				Status:      "started",
			},
		},
		"c_3": {
			{
				ContainerId: "c_3",
				MachineId:   "m_3",
				AppDu:       "app_2",
				Status:      "started",
			},
		},
	})
	assert.NoError(t, err)
}
