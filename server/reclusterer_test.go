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
	f, _ := os.Open("../test/csv/centers.csv")
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
