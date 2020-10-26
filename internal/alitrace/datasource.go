package alitrace

import (
	"bufio"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
	"strconv"
	"strings"
)

// 创建一个能读取alibaba trace data v2018的datasource
func NewAlitraceDatasource(reader io.Reader) datasource.MetricDataSource {
	return &alitraceDataSource{reader: bufio.NewReader(reader)}
}

type alitraceDataSource struct {
	reader *bufio.Reader
}

func (a *alitraceDataSource) Load() (*datasource.ContainerMetric, error) {
	line, err := a.reader.ReadString(core.LineBreak)
	if line != "" {
		record := strings.Split(line, core.Splitter)
		if len(record) != 11 {
			return nil, fmt.Errorf("输入格式有误。可能不是有效的container_usage.csv文件")
		}
		cid := record[0]
		timestamp, err := strconv.ParseUint(record[2], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("解析时间戳错误，错误值为%s", record[2]))
		}
		cpuUtil, err := strconv.ParseFloat(record[3], 32)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("解析CPU利用率错误，错误值为%s", record[3]))
		}
		memUtil, err := strconv.ParseFloat(record[4], 32)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("解析Mem利用率错误，错误值为%s", record[4]))
		}

		return &datasource.ContainerMetric{
			ContainerId: cid,
			Cpu:         float32(cpuUtil),
			Mem:         float32(memUtil),
			Timestamp:   timestamp,
		}, nil
	}
	return nil, err
}
