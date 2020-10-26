package core

import "reflect"

type SectionData struct {
	CpuAvg float32 `json:"cpuAvg"`
	CpuMax float32 `json:"cpuMax"`
	CpuMin float32 `json:"cpuMin"`
	CpuP50 float32 `json:"cpuP50"`
	CpuP90 float32 `json:"cpuP90"`
	CpuP99 float32 `json:"cpuP99"`
	MemAvg float32 `json:"memAvg"`
	MemMax float32 `json:"memMax"`
	MemMin float32 `json:"memMin"`
	MemP50 float32 `json:"memP50"`
	MemP90 float32 `json:"memP90"`
	MemP99 float32 `json:"memP99"`
}

const SectionLength = 15 * 60

const DayLength = 24 * 60 * 60

const NumSections = DayLength / SectionLength

var NumSectionFields = reflect.TypeOf(SectionData{}).NumField()

type ContainerWorkloadData struct {
	ContainerId string
	Data        []*SectionData
}

const LineBreak = '\n'

const Splitter = ","

type RawSectionData struct {
	Cpu    []float32
	Mem    []float32
	CpuSum float32 // Cpu的总和。用于计算平均值
	MemSum float32 // Mem的总和。用于计算平均值
}

type ContainerRawData struct {
	ContainerId string
	Data        []*RawSectionData
}
