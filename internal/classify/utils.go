package classify

import (
	"encoding/csv"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"io"
	"reflect"
	"strconv"
)

func OutputResult(data [][]float32, output io.Writer, precision int) error {
	writer := csv.NewWriter(output)
	for _, datum := range data {
		record := make([]string, len(datum))
		for i, f := range datum {
			record[i] = strconv.FormatFloat(float64(f), 'f', precision, 32)
		}
		err := writer.Write(record)
		if err != nil {
			return errors.Wrap(err, "写入数据错误")
		}
	}

	writer.Flush()
	return nil
}

func ContainerWorkloadToFloatArray(workloads []*internal.ContainerWorkloadData) map[string][]float32 {
	result := make(map[string][]float32)
	typ := reflect.TypeOf(internal.SectionData{})
	for _, workload := range workloads {
		arr := make([]float32, internal.NumSectionFields*len(workload.Data))
		for j, datum := range workload.Data {
			val := reflect.ValueOf(datum)
			for k := 0; k < typ.NumField(); k++ {
				arr[j*internal.NumSectionFields+k] = float32(val.Elem().Field(k).Float())
			}
		}
		result[workload.ContainerId] = arr
	}
	return result
}
