package utils

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"math"
	"reflect"
	"strconv"
)

func GetSortedPositionValue(arr []float32, pos int) float32 {
	if pos < 0 || pos >= len(arr) {
		return float32(math.NaN())
	}

	l := 0
	r := len(arr) - 1
	for idx := Partition(arr, l, r); idx != pos && l+1 < r; idx = Partition(arr, l, r) {
		if idx < pos {
			l = idx + 1
		} else if idx > pos {
			r = idx - 1
		}
	}

	return arr[pos]
}

func Partition(arr []float32, l, r int) int {
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

func RecordToContainerWorkloadData(record []string) (*core.ContainerWorkloadData, error) {
	name := ""

	if len(record) < core.NumSectionFields*core.NumSections {
		return nil, fmt.Errorf("数据有误")
	}

	if len(record) > core.NumSections*core.NumSectionFields {
		name = record[0]
	}

	array, err := RecordsToSectionArray(record[len(record)-core.NumSectionFields*core.NumSections:])
	if err != nil {
		return nil, err
	}

	return &core.ContainerWorkloadData{
		ContainerId: name,
		Data:        array,
	}, nil
}

func RecordsToSectionArray(record []string) ([]*core.SectionData, error) {
	arr := make([]*core.SectionData, core.NumSections)
	for s := 0; s < core.NumSections; s++ {
		data := &core.SectionData{}
		val := reflect.ValueOf(data).Elem()
		for fi := 0; fi < core.NumSectionFields; fi++ {
			f, err := strconv.ParseFloat(record[s*core.NumSectionFields+fi], 32)
			if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("第%d个数据有问题，数据为：%s",
					s*core.NumSectionFields+fi, record[s*core.NumSectionFields+fi]))
			}
			val.Field(fi).SetFloat(f)
		}

		arr[s] = data
	}

	return arr, nil
}

func WorkloadDataToStringRecord(data *core.ContainerWorkloadData) []string {
	record := make([]string, 1+len(data.Data)*core.NumSectionFields)
	record[0] = data.ContainerId
	for i, datum := range data.Data {
		val := reflect.ValueOf(datum).Elem()
		for j := 0; j < core.NumSectionFields; j++ {
			record[1+i*core.NumSectionFields+j] = strconv.FormatFloat(val.Field(j).Float(), 'f', 2, 32)
		}
	}

	return record
}

func ContainerWorkloadToFloatArray(workloads []*core.ContainerWorkloadData) [][]float32 {
	result := make([][]float32, len(workloads))
	for i, workload := range workloads {
		result[i] = SectionDataToFloatArray(workload.Data)
	}
	return result
}

func SectionDataToFloatArray(sectionData []*core.SectionData) []float32 {
	arr := make([]float32, core.NumSectionFields*len(sectionData))
	for j, datum := range sectionData {
		val := reflect.ValueOf(datum)
		for k := 0; k < core.NumSectionFields; k++ {
			arr[j*core.NumSectionFields+k] = float32(val.Elem().Field(k).Float())
		}
	}
	return arr
}
