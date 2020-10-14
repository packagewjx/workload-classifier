package preprocess

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestImputeMissingValues(t *testing.T) {
	record := make([]string, internal.NumSectionFields*internal.NumSections)

	for si := 0; si < internal.NumSectionFields; si++ {
		for i := 0; i < internal.NumSections; i++ {
			record[i*internal.NumSectionFields+si] = fmt.Sprintf("%d", i)
		}

		for i := 0; i < 4; i++ {
			record[i*internal.NumSectionFields+si] = "NaN"
		}

		record[10*internal.NumSectionFields+si] = "NaN"
		record[11*internal.NumSectionFields+si] = "NaN"

		record[50*internal.NumSectionFields+si] = "NaN"

		record[95*internal.NumSectionFields+si] = "NaN"
	}

	reader := strings.NewReader(strings.Join(record, internal.Splitter))
	builder := &strings.Builder{}
	err := ImputeMissingValues(reader, builder)
	assert.NoError(t, err)
	result := strings.Split(builder.String(), internal.Splitter)
	for _, s := range result {
		assert.NotEqual(t, "NaN", s)
	}

	assert.Equal(t, "0.80", result[0*internal.NumSectionFields+1])
	assert.Equal(t, "1.60", result[1*internal.NumSectionFields+1])
	assert.Equal(t, "2.40", result[2*internal.NumSectionFields+1])
	assert.Equal(t, "3.20", result[3*internal.NumSectionFields+1])
	assert.Equal(t, "10.00", result[10*internal.NumSectionFields+1])
	assert.Equal(t, "11.00", result[11*internal.NumSectionFields+1])
	assert.Equal(t, "50.00", result[50*internal.NumSectionFields+1])
	assert.Equal(t, "47.00", result[95*internal.NumSectionFields+1])
}
