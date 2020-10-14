package alitrace

import (
	. "github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

func TestAlitraceDataSource_Load(t *testing.T) {
	testData := "c_10133,m_470,166760,5,67,,,,0.03,0.03,6\n" +
		"c_10133,m_470,167160,6,67,,,,0.03,0.03,7\n" +
		"c_10133,m_470,167210,6,67,,,,0.03,0.03,7\n" +
		"c_10133,m_470,167220,5,67,,,,0.03,0.03,8\n" +
		"c_10133,m_470,167620,5,66,,,,0.03,0.03,7\n" +
		"c_10133,m_470,167700,5,66,,,,0.03,0.03,7\n" +
		"c_10133,m_470,167900,7,67,,,,0.03,0.03,6\n" +
		"c_10133,m_470,169230,7,66,,,,0.03,0.03,7\n" +
		"c_10133,m_470,169650,7,67,,,,0.03,0.03,6\n" +
		"c_10133,m_470,169950,7,66,,,,0.03,0.03,7"
	reader := strings.NewReader(testData)
	datasource := NewAlitraceDatasource(reader)
	data := make([]*ContainerMetric, 0)
	var r *ContainerMetric
	var err error
	for r, err = datasource.Load(); err == nil; r, err = datasource.Load() {
		assert.NoError(t, err)
		data = append(data, r)
	}
	assert.Equal(t, 10, len(data))
	assert.Equal(t, "c_10133", data[0].ContainerId)
	assert.Equal(t, uint64(166760), data[0].Timestamp)
	assert.Equal(t, float32(5), data[0].Cpu)
	assert.Equal(t, float32(67), data[0].Mem)
	assert.Equal(t, io.EOF, err)
}
