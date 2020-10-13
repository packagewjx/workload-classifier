package alitrace

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	record := make([]string, 1+internal.NumSections*internal.NumSectionFields)
	record[0] = "test"
	for i := 0; i < internal.NumSections; i++ {
		for j := 0; j < internal.NumSectionFields; j++ {
			record[1+i*internal.NumSectionFields+j] = fmt.Sprintf("%d", i)
		}
	}
	reader := strings.NewReader(strings.Join(record, internal.Splitter))
	builder := &strings.Builder{}

	err := NormalizeSection(reader, builder)
	assert.NoError(t, err)

	result := strings.Split(builder.String(), internal.Splitter)
	for i := 0; i < internal.NumSections-1; i++ {
		for j := 0; j < internal.NumSectionFields; j++ {
			f, err := strconv.ParseFloat(result[1+i*internal.NumSectionFields+j], 32)
			assert.NoError(t, err)
			assert.Condition(t, func() (success bool) {
				return !math.IsNaN(f) && f < 1
			})
		}
	}

	for i := (internal.NumSections-1)*internal.NumSectionFields + 1; i < len(result); i++ {
		assert.Equal(t, "1.00", strings.TrimSpace(result[i]))
	}

	/*
		数据不够长
	*/
	reader = strings.NewReader("1,2,3,4,5")
	err = NormalizeSection(reader, os.Stdout)
	assert.Error(t, err)

	/*
		存在错误数据
	*/
	record[1] = "a"
	err = NormalizeSection(strings.NewReader(strings.Join(record, internal.Splitter)), os.Stdout)
	assert.Error(t, err)

	/*
		读取不能读的数据
	*/
	err = NormalizeSection(os.Stdout, os.Stdout)
	assert.Error(t, err)

}
