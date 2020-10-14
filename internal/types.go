package internal

import "reflect"

type SectionData struct {
	CpuAvg float32
	CpuMax float32
	CpuMin float32
	CpuP50 float32
	CpuP90 float32
	CpuP99 float32
	MemAvg float32
	MemMax float32
	MemMin float32
	MemP50 float32
	MemP90 float32
	MemP99 float32
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
