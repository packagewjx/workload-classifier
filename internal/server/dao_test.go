package server

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/core"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

var testHost = "127.0.0.1:3306"

func init() {
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("root:wujunxian@tcp(%s)/metrics?charset=utf8mb4&parseTime=True&loc=Local", testHost)), &gorm.Config{})
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
	db, _ := gorm.Open(mysql.Open(fmt.Sprintf("root:wujunxian@tcp(%s)/metrics?charset=utf8mb4&parseTime=True&loc=Local", testHost)), &gorm.Config{})
	db.Create(&AppDo{
		Model: gorm.Model{ID: 1000},
		AppName: server.AppName{
			Name:      "haha",
			Namespace: "test",
		},
	})

	_, err := NewDao(testHost)
	assert.NoError(t, err)
}

func TestDao_SaveAllAppPodMetrics(t *testing.T) {
	arr := make([]*server.AppPodMetrics, 10)
	for i := 0; i < len(arr); i++ {
		arr[i] = &server.AppPodMetrics{
			AppName: server.AppName{
				Name:      fmt.Sprintf("test-%d", i),
				Namespace: "test",
			},
			Timestamp: 10000,
			Cpu:       float32(i) * 10,
			Mem:       float32(i) * 10,
		}
	}

	dao, _ := NewDao(testHost)

	err := dao.SaveAllAppPodMetrics(arr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "保存AppPodMetrics失败")
	}

	impl := dao.(*daoImpl)

	for _, metrics := range arr {
		dest := &AppPodMetricsDO{}
		impl.db.Where(&AppPodMetricsDO{AppId: impl.appIdMap[impl.keyFunc(&metrics.AppName)], Timestamp: metrics.Timestamp}).First(dest)
		assert.Equal(t, uint64(10000), dest.Timestamp)
	}

	/**
	测试更新
	*/
	for _, metrics := range arr {
		metrics.Cpu = 100
		metrics.Mem = 100
	}
	err = dao.SaveAllAppPodMetrics(arr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "更新AppPodMetrics失败")
	}

	for _, metrics := range arr {
		dest := &AppPodMetricsDO{}
		impl.db.Where(&AppPodMetricsDO{AppId: impl.appIdMap[impl.keyFunc(&metrics.AppName)], Timestamp: metrics.Timestamp}).First(dest)
		assert.Equal(t, uint64(10000), dest.Timestamp)
		assert.Equal(t, float32(100), dest.Cpu)
	}
}

func TestDaoImpl_SaveAppClass(t *testing.T) {
	dao, _ := NewDao(testHost)

	/*
		测试新增
	*/
	arr := make([]*server.AppClass, 10)
	for i := 0; i < len(arr); i++ {
		arr[i] = &server.AppClass{
			AppName: server.AppName{
				Name:      fmt.Sprintf("test-%d", i),
				Namespace: "test",
			},
			ClassId: uint(i),
			CpuMax:  float32(i),
			MemMax:  float32(i),
		}
		err := dao.SaveAppClass(arr[i])
		assert.NoError(t, err)
	}

	impl := dao.(*daoImpl)
	for _, class := range arr {
		dest := &AppClassDO{}
		impl.db.Where(&AppClassDO{AppId: impl.appIdMap[impl.keyFunc(&class.AppName)]}).First(dest)
		assert.Equal(t, class.ClassId, dest.ClassId)
		assert.Equal(t, class.CpuMax, dest.CpuMax)
		assert.Equal(t, class.MemMax, dest.MemMax)
	}

	/*
		测试更新
	*/

	for _, class := range arr {
		class.ClassId += 10
		class.MemMax += 10
		class.CpuMax += 10
		err := dao.SaveAppClass(class)
		assert.NoError(t, err)

		dest := &AppClassDO{}
		impl.db.Where(&AppClassDO{AppId: impl.appIdMap[impl.keyFunc(&class.AppName)]}).First(dest)
		assert.Equal(t, class.ClassId, dest.ClassId)
		assert.Equal(t, class.CpuMax, dest.CpuMax)
		assert.Equal(t, class.MemMax, dest.MemMax)
	}

	/*
		测试不存在的AppID
	*/
	appClass := &server.AppClass{
		AppName: server.AppName{
			Name:      "absolutelyNotExistApp",
			Namespace: "absolutelyNotExistNamespace",
		},
		ClassId: 10,
	}
	err := dao.SaveAppClass(appClass)
	assert.NoError(t, err)
}

func TestDaoImpl_SaveClassMetrics(t *testing.T) {
	c := &server.ClassMetrics{
		ClassId: 10,
		Data:    make([]*core.SectionData, core.NumSections),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &core.SectionData{}
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

	dao, _ := NewDao(testHost)
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
	dao, _ := NewDao(testHost)
	size := 10000
	arr := make([]*server.AppPodMetrics, size)
	for i := 0; i < size; i++ {
		arr[i] = &server.AppPodMetrics{
			AppName: server.AppName{
				Name:      "test-1",
				Namespace: "test",
			},
			Timestamp: uint64(i),
			Cpu:       float32(i),
			Mem:       float32(i),
		}
	}

	err := dao.SaveAllAppPodMetrics(arr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "保存AppPodMetrics出错")
	}
	timeStart := uint64(5000)
	err = dao.RemoveAppPodMetricsBefore(timeStart)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "移除AppPodMetrics出错")
	}

	db := dao.(*daoImpl).db
	queryArr := []*AppPodMetricsDO{}
	err = db.Find(&queryArr).Error
	assert.NoError(t, err)
	for _, do := range queryArr {
		assert.Condition(t, func() (success bool) {
			return do.Timestamp >= timeStart
		})
	}
}

func TestDaoImpl_QueryClassMetricsByClassId(t *testing.T) {
	dao, _ := NewDao(testHost)
	classId := uint(10)
	c := &server.ClassMetrics{
		ClassId: classId,
		Data:    make([]*core.SectionData, core.NumSections),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &core.SectionData{
			CpuAvg: float32(i),
			CpuMax: float32(i),
			CpuMin: float32(i),
			CpuP50: float32(i),
			CpuP90: float32(i),
			CpuP99: float32(i),
			MemAvg: float32(i),
			MemMax: float32(i),
			MemMin: float32(i),
			MemP50: float32(i),
			MemP90: float32(i),
			MemP99: float32(i),
		}
	}

	err := dao.SaveClassMetrics(c)
	assert.NoError(t, err)

	c, err = dao.QueryClassMetricsByClassId(classId)
	assert.NoError(t, err)
	assert.Equal(t, classId, c.ClassId)
	for i, datum := range c.Data {
		assert.Equal(t, float32(i), datum.CpuAvg)
	}

	/*
		查询不存在的记录
	*/
	_, err = dao.QueryClassMetricsByClassId(1000)
	assert.Error(t, err)

	/*
		查询带缺漏的数据
	*/
	db := dao.(*daoImpl).db
	classId = 10000
	err = db.Create(&ClassSectionMetricsDO{
		ID:          classId,
		SectionNum:  10,
		SectionData: core.SectionData{},
	}).Error
	if !assert.NoError(t, err) {
		assert.FailNow(t, "插入缺漏数据出错")
	}
	_, err = dao.QueryClassMetricsByClassId(classId)
	assert.Error(t, err)

	/*
		查询不缺漏，但是不够数量的数据
	*/
	classId = 10001
	c = &server.ClassMetrics{
		ClassId: classId,
		Data:    make([]*core.SectionData, 10),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &core.SectionData{}
	}
	err = dao.SaveClassMetrics(c)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "插入不足数据出错")
	}
	_, err = dao.QueryClassMetricsByClassId(classId)
	assert.Error(t, err)
}

func TestDaoImpl_QueryAppClassIdByApp(t *testing.T) {
	dao, _ := NewDao(testHost)

	db := dao.(*daoImpl).db
	classId := uint(10)
	appName := "queryAppClass"
	namespace := "query"
	db.Create(&AppClassDO{
		Model:   gorm.Model{},
		AppId:   10,
		ClassId: classId,
	})
	db.Create(&AppDo{
		Model: gorm.Model{
			ID: 10,
		},
		AppName: server.AppName{
			Name:      appName,
			Namespace: namespace,
		},
	})

	appClass, err := dao.QueryAppClassByApp(&server.AppName{
		Name:      appName,
		Namespace: namespace,
	})
	assert.NoError(t, err)
	assert.Equal(t, classId, appClass.ClassId)

	/*
		查询不存在的
	*/
	_, err = dao.QueryAppClassByApp(&server.AppName{
		Name:      "absolutelyNotExistApp_TestDaoImpl_QueryAppClassIdByApp",
		Namespace: "absolutelyNotExistNamespace_TestDaoImpl_QueryAppClassIdByApp",
	})
	assert.Error(t, err)
}

func TestDaoImpl_RemoveAllClassMetrics(t *testing.T) {
	dao, _ := NewDao(testHost)
	db := dao.(*daoImpl).db
	for i := 0; i < 10; i++ {
		err := db.Create(&ClassSectionMetricsDO{}).Error
		if !assert.NoError(t, err) {
			assert.FailNow(t, "创建初始数据出错")
		}
	}

	arr := []*ClassSectionMetricsDO{}
	db.Find(&arr)
	assert.NotEqual(t, 0, len(arr))

	err := dao.RemoveAllClassMetrics()
	assert.NoError(t, err)

	arr = []*ClassSectionMetricsDO{}
	db.Find(&arr)
	assert.Equal(t, 0, len(arr))
}

func TestDaoImpl_QueryAllClassMetrics(t *testing.T) {
	dao, _ := NewDao(testHost)
	err := dao.DB().Delete(&ClassSectionMetricsDO{}, "1 = 1").Error
	if err != nil {
		assert.FailNow(t, "删除数据失败")
	}

	testData := make([]*server.ClassMetrics, DefaultNumClass)
	for i := 0; i < len(testData); i++ {
		testData[i] = &server.ClassMetrics{
			ClassId: uint(i + 1), // ID不能为0
			Data:    make([]*core.SectionData, core.NumSections),
		}

		for j := 0; j < len(testData[i].Data); j++ {
			data := &core.SectionData{}
			val := reflect.ValueOf(data).Elem()
			for k := 0; k < val.NumField(); k++ {
				val.Field(k).SetFloat(float64(j))
			}
			testData[i].Data[j] = data
		}

		err := dao.SaveClassMetrics(testData[i])
		if err != nil {
			assert.FailNow(t, "保存数据失败")
		}
	}

	metrics, err := dao.QueryAllClassMetrics()
	assert.NoError(t, err)
	assert.Equal(t, len(testData), len(metrics))
	for _, metric := range metrics {
		assert.Condition(t, func() (success bool) {
			return metric.ClassId <= DefaultNumClass
		})
		for i, datum := range metric.Data {
			val := reflect.ValueOf(datum).Elem()
			for j := 0; j < val.NumField(); j++ {
				assert.Equal(t, float64(i), val.Field(j).Float())
			}
		}
	}
}
