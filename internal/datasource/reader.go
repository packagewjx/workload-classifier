package datasource

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/pkg/errors"
	"io"
)

func NewDataSourceRawDataReader(source MetricDataSource) RawDataReader {
	return &dataSourceRawDataReader{datasource: source}
}

type dataSourceRawDataReader struct {
	datasource MetricDataSource
}

func (d *dataSourceRawDataReader) Read() ([]*core.ContainerRawData, error) {
	m := make(map[string]*core.ContainerRawData)
	var r *ContainerMetric
	var err error
	for r, err = d.datasource.Load(); err == nil; r, err = d.datasource.Load() {
		cd, ok := m[r.ContainerId]
		if !ok {
			cd = &core.ContainerRawData{
				ContainerId: r.ContainerId,
				Data:        make([]*core.RawSectionData, core.NumSections),
			}
			for i := 0; i < len(cd.Data); i++ {
				cd.Data[i] = &core.RawSectionData{
					Cpu:    make([]float32, 0),
					Mem:    make([]float32, 0),
					CpuSum: 0,
					MemSum: 0,
				}
			}
			m[r.ContainerId] = cd
		}
		section := cd.Data[r.Timestamp%core.DayLength/core.SectionLength]
		section.Cpu = append(section.Cpu, r.Cpu)
		section.CpuSum += r.Cpu
		section.Mem = append(section.Mem, r.Mem)
		section.MemSum += r.Mem
	}

	if err != io.EOF {
		return nil, errors.Wrap(err, "读取ContainerMetric出现问题")
	}

	result := make([]*core.ContainerRawData, 0, len(m))
	for _, data := range m {
		result = append(result, data)
	}

	return result, nil
}
