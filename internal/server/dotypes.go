package server

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"gorm.io/gorm"
	"time"
)

type AppDo struct {
	gorm.Model
	server.AppName
}

type AppPodMetricsDO struct {
	gorm.Model
	AppId     uint   `gorm:"uniqueIndex:unique_record"`
	Timestamp uint64 `gorm:"uniqueIndex:unique_record"`
	Cpu       float32
	Mem       float32
}

type AppClassDO struct {
	gorm.Model
	AppId   uint `gorm:"uniqueIndex"`
	ClassId uint
	CpuMax  float32
	MemMax  float32
}

type ClassSectionMetricsDO struct {
	ID         uint `gorm:"primarykey"`
	SectionNum uint `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	core.SectionData
}
