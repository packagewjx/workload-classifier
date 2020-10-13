package utils

import (
	"encoding/csv"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"io"
	"strings"
)

func WriteContainerWorkloadHeader(out io.Writer) error {
	header := make([]string, 1, 1+internal.NumSections+internal.NumSectionFields)
	header[0] = "container_id"
	for i := 0; i < internal.NumSections; i++ {
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
	_, err := out.Write([]byte(strings.Join(header, internal.Splitter)))
	return err
}

func WriteContainerWorkloadData(out io.Writer, data []*internal.ContainerWorkloadData) error {
	writer := csv.NewWriter(out)
	defer writer.Flush()

	for i, cData := range data {
		record := make([]string, 1, 1+internal.NumSections*internal.NumSectionFields)
		record[0] = cData.ContainerId
		for i := 0; i < len(cData.Data); i++ {
			stat := cData.Data[i]
			record = append(record, fmt.Sprintf("%.2f", stat.CpuAvg))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuMax))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuMin))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP50))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP90))
			record = append(record, fmt.Sprintf("%.2f", stat.CpuP99))
			record = append(record, fmt.Sprintf("%.2f", stat.MemAvg))
			record = append(record, fmt.Sprintf("%.2f", stat.MemMax))
			record = append(record, fmt.Sprintf("%.2f", stat.MemMin))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP50))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP90))
			record = append(record, fmt.Sprintf("%.2f", stat.MemP99))
		}
		err := writer.Write(record)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("写入第%d条数据出错", i))
		}
	}

	return nil
}
