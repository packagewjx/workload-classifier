package alitrace

import (
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sort"
	"testing"
)

func TestPartition(t *testing.T) {
	arr := []float32{3, 6, 1, 76, 2, 16, 549}
	idx := partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 5)
	assert.Equal(t, arr[5], float32(76))
	idx = partition(arr, 0, idx)
	assert.Condition(t, func() (success bool) {
		return idx < 5
	})

	arr = []float32{1, 2, 3}
	idx = partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 1)
	assert.Equal(t, arr[1], float32(2))

	arr = []float32{4, 2, 1, 3}
	idx = partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 0)
	assert.Equal(t, arr[0], float32(1))

	arr = []float32{}
	idx = partition(arr, 0, len(arr)-1)
	assert.Equal(t, idx, 0)
}

func TestGetSortedPosition(t *testing.T) {
	arr := []float32{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	n := getSortedPositionValue(arr, 4)
	assert.Equal(t, float32(4), n)

	arr = make([]float32, 10000)
	for i := 0; i < len(arr); i++ {
		arr[i] = rand.Float32() * 10000
	}
	p0 := getSortedPositionValue(arr, 0)
	p1000 := getSortedPositionValue(arr, 1000)
	p2000 := getSortedPositionValue(arr, 2000)
	p5000 := getSortedPositionValue(arr, 5000)
	p9999 := getSortedPositionValue(arr, 9999)
	sort.Slice(arr, func(i, j int) bool {
		return arr[i] < arr[j]
	})
	assert.Equal(t, arr[0], p0)
	assert.Equal(t, arr[1000], p1000)
	assert.Equal(t, arr[2000], p2000)
	assert.Equal(t, arr[5000], p5000)
	assert.Equal(t, arr[9999], p9999)
}
