package preprocess

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestImputeMissingValues(t *testing.T) {
	record := make([]string, core.NumSectionFields*core.NumSections)

	for si := 0; si < core.NumSectionFields; si++ {
		for i := 0; i < core.NumSections; i++ {
			record[i*core.NumSectionFields+si] = fmt.Sprintf("%d", i)
		}

		for i := 0; i < 4; i++ {
			record[i*core.NumSectionFields+si] = "NaN"
		}

		record[10*core.NumSectionFields+si] = "NaN"
		record[11*core.NumSectionFields+si] = "NaN"

		record[50*core.NumSectionFields+si] = "NaN"

		record[95*core.NumSectionFields+si] = "NaN"
	}

	reader := strings.NewReader(strings.Join(record, core.Splitter))
	builder := &strings.Builder{}
	err := ImputeMissingValues(reader, builder)
	assert.NoError(t, err)
	result := strings.Split(builder.String(), core.Splitter)
	for _, s := range result {
		assert.NotEqual(t, "NaN", s)
	}

	assert.Equal(t, "0.80", result[0*core.NumSectionFields+1])
	assert.Equal(t, "1.60", result[1*core.NumSectionFields+1])
	assert.Equal(t, "2.40", result[2*core.NumSectionFields+1])
	assert.Equal(t, "3.20", result[3*core.NumSectionFields+1])
	assert.Equal(t, "10.00", result[10*core.NumSectionFields+1])
	assert.Equal(t, "11.00", result[11*core.NumSectionFields+1])
	assert.Equal(t, "50.00", result[50*core.NumSectionFields+1])
	assert.Equal(t, "47.00", result[95*core.NumSectionFields+1])
}
