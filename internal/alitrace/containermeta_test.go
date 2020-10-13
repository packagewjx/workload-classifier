package alitrace

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestLoadContainerMeta(t *testing.T) {
	testData := "c_1,m_2556,0,app_5052,started,400,400,1.56\n" +
		"c_1,m_2556,287942,app_5052,started,400,400,1.56\n" +
		"c_1,m_2556,338909,app_5052,started,400,400,1.56\n"
	reader := strings.NewReader(testData)

	meta, err := LoadContainerMeta(reader)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(meta))
	d, ok := meta["c_1"]
	assert.True(t, ok)
	assert.Equal(t, 3, len(d))
	assert.Equal(t, int64(287942), d[1].Timestamp)
	assert.Equal(t, 400, d[1].CpuRequest)
	assert.Equal(t, 400, d[1].CpuLimit)
	assert.Equal(t, float32(1.56), d[1].MemSize)

	// 读取错误的数据
	testData = "c_1,m_2556,,app_5052,started,,,"
	reader = strings.NewReader(testData)
	_, err = LoadContainerMeta(reader)
	assert.NoError(t, err)
}
