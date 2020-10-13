package server

import (
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestReadInitialCenters(t *testing.T) {
	f, _ := os.Open("../../test/csv/centers.csv")
	centers, err := readInitialCenter(f)
	assert.NoError(t, err)
	assert.Equal(t, 20, len(centers))
	for _, center := range centers {
		assert.Equal(t, internal.NumSections, len(center.Data))
		for _, datum := range center.Data {
			v := reflect.ValueOf(*datum)
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i).Float()
				assert.NotEqual(t, 0, f)
				assert.False(t, math.IsNaN(f))
			}
		}
	}

	// 读取错误的csv数据
	f, _ = os.Open("/dev/null")
	_, err = readInitialCenter(f)
	assert.Error(t, err)

	reader := strings.NewReader(",")
	_, err = readInitialCenter(reader)
	assert.Error(t, err)

	// 读取一行中间有错误数据的数据
	falseString := make([]string, internal.NumSectionFields*internal.NumSections)
	for i := 0; i < len(falseString); i++ {
		falseString[i] = strconv.FormatInt(int64(i), 10)
	}
	falseString[internal.NumSections] = ""
	reader = strings.NewReader(strings.Join(falseString, ","))
	_, err = readInitialCenter(reader)
	assert.Error(t, err)
}

func TestPodMetricToRawData(t *testing.T) {
	m := map[string]map[string][]*AppPodMetrics{
		"test": {
			"test": make([]*AppPodMetrics, 0),
		},
	}

	arr := m["test"]["test"]
	for i := 0; i < internal.NumSections; i++ {
		sectionTimestamp := i * internal.SectionLength

		for j := 0; j < 10; j++ {
			arr = append(arr, &AppPodMetrics{
				AppName: AppName{
					Name:      "test",
					Namespace: "test",
				},
				Timestamp: uint64(sectionTimestamp + j),
				Cpu:       float32(j),
				Mem:       float32(j),
			})
		}
	}

	m["test"]["test"] = arr
	rawDataMap := podMetricsToRawData(m)
	assert.Equal(t, 1, len(rawDataMap))

	rawDataArray, ok := rawDataMap["test::test"]
	assert.True(t, ok)
	for _, data := range rawDataArray {
		assert.Equal(t, float32(45), data.MemSum)
		assert.Equal(t, float32(45), data.CpuSum)
		assert.Equal(t, 10, len(data.Cpu))
		assert.Equal(t, 10, len(data.Cpu))
	}
}

func TestFloatArrayToClassMetrics(t *testing.T) {
	arr := make([]float32, internal.NumSections*internal.NumSectionFields)
	for i := 0; i < len(arr); i++ {
		arr[i] = float32(i)
	}
	metrics := floatArrayToClassMetrics(1, arr)
	assert.Equal(t, uint(1), metrics.ClassId)
	assert.Equal(t, internal.NumSections, len(metrics.Data))
	assert.Equal(t, float32(1), metrics.Data[0].CpuMax)
}
