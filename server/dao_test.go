package server

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
)

func init() {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	s, _ := db.DB()
	_, _ = s.Exec("DELETE FROM app_class_dos")
	_, _ = s.Exec("DELETE FROM app_dos")
	_, _ = s.Exec("DELETE FROM app_pod_metrics_dos")
	_, _ = s.Exec("DELETE FROM class_section_metrics_dos")
}

func TestNewDao(t *testing.T) {
	_, err := NewDao()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDao_SaveAllAppPodMetrics(t *testing.T) {
	arr := make([]*AppPodMetrics, 10)
	for i := 0; i < len(arr); i++ {
		arr[i] = &AppPodMetrics{
			AppName:   fmt.Sprintf("test-%d", i),
			Namespace: "test",
			Timestamp: 10000,
			Cpu:       float32(i) * 10,
			Mem:       float32(i) * 10,
		}
	}

	dao, _ := NewDao()

	err := dao.SaveAllAppPodMetrics(arr)
	if err != nil {
		t.Fatal(err)
	}

	impl := dao.(*daoImpl)

	for _, metrics := range arr {
		dest := &AppPodMetricsDO{}
		impl.db.Where(&AppPodMetricsDO{AppId: impl.appIdMap[impl.keyFunc(metrics.AppName, metrics.Namespace)], Timestamp: metrics.Timestamp}).First(dest)
		assert.Equal(t, uint64(10000), dest.Timestamp)
	}
}

func TestDaoImpl_SaveAppClass(t *testing.T) {
	dao, _ := NewDao()

	arr := make([]*AppClass, 10)
	for i := 0; i < len(arr); i++ {
		arr[i] = &AppClass{
			AppName:   fmt.Sprintf("test-%d", i),
			Namespace: "test",
			ClassId:   uint(i),
		}
	}

	for _, class := range arr {
		err := dao.SaveAppClass(class)
		if err != nil {
			t.Error(err)
		}
	}

	impl := dao.(*daoImpl)
	for _, class := range arr {
		dest := &AppClassDO{}
		impl.db.Where(&AppClassDO{AppId: impl.appIdMap[impl.keyFunc(class.AppName, class.Namespace)]}).First(dest)
		assert.Equal(t, class.ClassId, dest.ClassId)
	}
}

func TestDaoImpl_SaveClassMetrics(t *testing.T) {
	c := &ClassMetrics{
		ClassId: 10,
		Data:    make([]*internal.ProcessedSectionData, internal.NumSections),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &internal.ProcessedSectionData{}
		c.Data[i].CpuAvg = float32(i)
		c.Data[i].CpuMax = float32(i)
		c.Data[i].CpuMin = float32(i)
		c.Data[i].CpuP50 = float32(i)
		c.Data[i].CpuP90 = float32(i)
		c.Data[i].CpuP99 = float32(i)
		c.Data[i].MemAvg = float32(i)
		c.Data[i].MemMax = float32(i)
		c.Data[i].MemMin = float32(i)
		c.Data[i].MemP50 = float32(i)
		c.Data[i].MemP90 = float32(i)
		c.Data[i].MemP99 = float32(i)
	}

	dao, _ := NewDao()
	err := dao.SaveClassMetrics(c)
	assert.NoError(t, err)

	dest := []*ClassSectionMetricsDO{}
	db := dao.(*daoImpl).db
	db.Where(&ClassSectionMetricsDO{ID: 10}).Find(&dest)
	assert.NotEqual(t, 0, len(dest))

	for _, do := range dest {
		assert.Equal(t, float32(do.SectionNum), do.CpuAvg)
	}
}

func TestDaoImpl_RemoveAppPodMetricsBefore(t *testing.T) {
	dao, _ := NewDao()
	size := 10000
	arr := make([]*AppPodMetrics, size)
	for i := 0; i < size; i++ {
		arr[i] = &AppPodMetrics{
			AppName:   "test-1",
			Namespace: "test",
			Timestamp: uint64(i),
			Cpu:       float32(i),
			Mem:       float32(i),
		}
	}

	err := dao.SaveAllAppPodMetrics(arr)
	assert.NoError(t, err)
	err = dao.RemoveAppPodMetricsBefore(5000)
	assert.NoError(t, err)

	db := dao.(*daoImpl).db
	queryArr := []*AppPodMetricsDO{}
	err = db.Find(&queryArr).Error
	assert.NoError(t, err)
	assert.Equal(t, 5000, len(queryArr))
}
