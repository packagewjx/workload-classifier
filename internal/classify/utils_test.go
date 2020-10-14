package classify

import (
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
