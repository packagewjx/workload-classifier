package datasource

import (
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"math"
	"sync"
)

func ConvertRawData(rawData *core.ContainerRawData) *core.ContainerWorkloadData {
	rawSectionData := rawData.Data
	cData := &core.ContainerWorkloadData{
		ContainerId: rawData.ContainerId,
		Data:        make([]*core.SectionData, len(rawSectionData)),
	}

	for si, section := range rawSectionData {
		stat := &core.SectionData{}
		if len(section.Cpu) != 0 {
			stat.CpuAvg = section.CpuSum / float32(len(section.Cpu))
			stat.CpuMax = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)-1)
			stat.CpuMin = utils.GetSortedPositionValue(section.Cpu, 0)
			stat.CpuP50 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)/2)
			stat.CpuP90 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)*90/100)
			stat.CpuP99 = utils.GetSortedPositionValue(section.Cpu, len(section.Cpu)*99/100)
		} else {
			stat.CpuAvg = float32(math.NaN())
			stat.CpuMax = float32(math.NaN())
			stat.CpuMin = float32(math.NaN())
			stat.CpuP50 = float32(math.NaN())
			stat.CpuP90 = float32(math.NaN())
			stat.CpuP99 = float32(math.NaN())
		}
		if len(section.Mem) != 0 {
			stat.MemAvg = section.MemSum / float32(len(section.Mem))
			stat.MemMax = utils.GetSortedPositionValue(section.Mem, len(section.Mem)-1)
			stat.MemMin = utils.GetSortedPositionValue(section.Mem, 0)
			stat.MemP50 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)/2)
			stat.MemP90 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)*90/100)
			stat.MemP99 = utils.GetSortedPositionValue(section.Mem, len(section.Mem)*99/100)
		} else {
			stat.MemAvg = float32(math.NaN())
			stat.MemMax = float32(math.NaN())
			stat.MemMin = float32(math.NaN())
			stat.MemP50 = float32(math.NaN())
			stat.MemP90 = float32(math.NaN())
			stat.MemP99 = float32(math.NaN())
		}
		cData.Data[si] = stat
	}

	return cData
}

func ConvertAllRawData(rawData []*core.ContainerRawData) []*core.ContainerWorkloadData {
	wg := sync.WaitGroup{}
	result := make([]*core.ContainerWorkloadData, len(rawData))
	for i, datum := range rawData {
		wg.Add(1)
		go func(idx int, d *core.ContainerRawData) {
			defer wg.Done()
			result[idx] = ConvertRawData(d)
		}(i, datum)
	}
	wg.Wait()
	return result
}
