package internal

import "reflect"

type ProcessedSectionData struct {
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

var NumSectionFields = reflect.TypeOf(ProcessedSectionData{}).NumField()
