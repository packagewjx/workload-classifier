package server

import (
	"github.com/packagewjx/workload-classifier/internal"
)

type AppPodMetrics struct {
	AppName
	Timestamp uint64  `gorm:"uniqueIndex:record"`
	Cpu       float32 `gorm:"not null;precision:2"`
	Mem       float32 `gorm:"not null:precision:2"`
}

type AppClass struct {
	AppName
	ClassId uint
}

type ClassMetrics struct {
	ClassId uint
	Data    []*internal.ProcessedSectionData
}

type AppName struct {
	Name      string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
	Namespace string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
}
