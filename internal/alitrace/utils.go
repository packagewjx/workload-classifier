package alitrace

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"math"
	"strconv"
)

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

func recordsToSectionArray(record []string) ([]*internal.ProcessedSectionData, error) {
	arr := make([]*internal.ProcessedSectionData, internal.NumSections)
	for s := 0; s < internal.NumSections; s++ {
		floatArr := make([]float32, internal.NumSectionFields)
		for fi := 0; fi < len(floatArr); fi++ {
			f, err := strconv.ParseFloat(record[s*internal.NumSectionFields+fi], 32)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("第%d个数据有问题，数据为：%s",
					s*internal.NumSectionFields+fi, record[s*internal.NumSectionFields+fi]))
			}
			floatArr[fi] = float32(f)
		}

		arr[s] = &internal.ProcessedSectionData{
			CpuAvg: floatArr[0],
			CpuMax: floatArr[1],
			CpuMin: floatArr[2],
			CpuP50: floatArr[3],
			CpuP90: floatArr[4],
			CpuP99: floatArr[5],
			MemAvg: floatArr[6],
			MemMax: floatArr[7],
			MemMin: floatArr[8],
			MemP50: floatArr[9],
			MemP90: floatArr[10],
			MemP99: floatArr[11],
		}
	}

	return arr, nil
}
