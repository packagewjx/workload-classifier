package classify

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestLoadCsv(t *testing.T) {
	loader := &csvLoader{}
	file, _ := os.Open("../../test/csv/container_meta.csv")

	data, err := loader.Load(file, []int{0, 1, 3, 4})
	if err != nil {
		t.Error(err)
	}
	assert.NotEqual(t, 0, len(data))
	assert.Equal(t, 4, len(data[0]))
}
