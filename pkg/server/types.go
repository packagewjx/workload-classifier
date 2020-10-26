package server

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
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
	CpuMax  float32 // 本应用CPU最大值。由于类数据是标准化后的数据，无法得知实际使用了多少CPU。CPU最大值代表类数据为1的时候的实际使用量
	MemMax  float32 // 本应用内存最大值
}

type ClassMetrics struct {
	ClassId uint                `json:"classId"`
	Data    []*core.SectionData `json:"data"`
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

var ErrAppNotFound = fmt.Errorf("不存在本应用")

var ErrAppNotClassified = fmt.Errorf("尚未对App分类")

type AppCharacteristics struct {
	AppName `json:",inline"`

	SectionData []*core.SectionData `json:"sectionData"`
}

type API interface {
	QueryAppCharacteristics(appName AppName) (*AppCharacteristics, error)

	ReCluster()
}
