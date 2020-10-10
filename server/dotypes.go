package server

import (
	"github.com/packagewjx/workload-classifier/internal"
	"gorm.io/gorm"
	"time"
)

type AppDo struct {
	gorm.Model
	AppName
}

type AppPodMetricsDO struct {
	gorm.Model
	AppId     uint   `gorm:"uniqueIndex:unique_record"`
	Timestamp uint64 `gorm:"uniqueIndex:unique_record,priority:9"`
	Cpu       float32
	Mem       float32
}

type AppClassDO struct {
	gorm.Model
	AppId   uint `gorm:"uniqueIndex"`
	ClassId uint
}

type ClassSectionMetricsDO struct {
	ID         uint `gorm:"primarykey"`
	SectionNum uint `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	internal.ProcessedSectionData
}
