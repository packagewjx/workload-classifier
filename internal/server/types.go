package server

import (
	"github.com/packagewjx/workload-classifier/internal"
	"strings"
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
	Data    []*internal.SectionData
}

type AppName struct {
	Name      string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
	Namespace string `gorm:"uniqueIndex:app;type:VARCHAR(256)"`
}

func (name AppName) ContainerId() string {
	return name.Namespace + NamespaceSplit + name.Name
}

const NamespaceSplit = "::"

func AppNameFromContainerId(containerId string) AppName {
	split := strings.Split(containerId, NamespaceSplit)
	return AppName{
		Name:      split[1],
		Namespace: split[0],
	}
}
