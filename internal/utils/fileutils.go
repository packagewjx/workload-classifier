package utils

import (
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
	"strings"
)

func WriteContainerWorkloadHeader(out io.Writer) error {
	header := make([]string, 1, 1+core.NumSections+core.NumSectionFields)
	header[0] = "container_id"
	for i := 0; i < core.NumSections; i++ {
		header = append(header,
			fmt.Sprintf("cpu_avg_%d", i),
			fmt.Sprintf("cpu_max_%d", i),
			fmt.Sprintf("cpu_min_%d", i),
			fmt.Sprintf("cpu_p50_%d", i),
			fmt.Sprintf("cpu_p90_%d", i),
			fmt.Sprintf("cpu_p99_%d", i),
			fmt.Sprintf("mem_avg_%d", i),
			fmt.Sprintf("mem_max_%d", i),
			fmt.Sprintf("mem_min_%d", i),
			fmt.Sprintf("mem_p50_%d", i),
			fmt.Sprintf("mem_p90_%d", i),
			fmt.Sprintf("mem_p99_%d", i))
	}
	_, err := out.Write([]byte(strings.Join(header, core.Splitter)))
	return err
}

func WriteContainerWorkloadData(out io.Writer, data []*core.ContainerWorkloadData) error {
	writer := csv.NewWriter(out)
	defer writer.Flush()

	for i, cData := range data {
		record := WorkloadDataToStringRecord(cData)
		err := writer.Write(record)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("写入第%d条数据出错", i))
		}
	}

	return nil
}
