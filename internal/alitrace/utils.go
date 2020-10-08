package alitrace

import "math"

func getSortedPositionValue(arr []float32, pos int) float32 {
	if pos < 0 || pos >= len(arr) {
		return float32(math.NaN())
	}

	l := 0
	r := len(arr) - 1
	for idx := partition(arr, l, r); idx != pos && l+1 < r; idx = partition(arr, l, r) {
		if idx < pos {
			l = idx + 1
		} else if idx > pos {
			r = idx - 1
		}
	}

	return arr[pos]
}

func partition(arr []float32, l, r int) int {
	slice := arr[l : r+1]

	if len(slice) == 0 {
		return 0
	}
	m := len(slice) / 2
	temp := slice[0]
	slice[0] = slice[m]
	slice[m] = temp
	pivot := slice[0]

	i := 0
	j := len(slice) - 1

	for i < j {
		for i < j && slice[j] > pivot {
			j--
		}
		slice[i] = slice[j]

		for i < j && slice[i] <= pivot {
			i++
		}
		slice[j] = slice[i]
	}
	slice[i] = pivot

	return l + i
}
