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
	db, _ := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	db.Create(&AppDo{
		Model:     gorm.Model{ID: 1000},
		AppName:   "haha",
		Namespace: "test",
	})

	_, err := NewDao()
	assert.NoError(t, err)
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
		assert.FailNow(t, "保存AppPodMetrics失败")
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

	/*
		测试新增
	*/
	arr := make([]*AppClass, 10)
	for i := 0; i < len(arr); i++ {
		arr[i] = &AppClass{
			AppName:   fmt.Sprintf("test-%d", i),
			Namespace: "test",
			ClassId:   uint(i),
		}
		err := dao.SaveAppClass(arr[i])
		assert.NoError(t, err)
	}

	impl := dao.(*daoImpl)
	for _, class := range arr {
		dest := &AppClassDO{}
		impl.db.Where(&AppClassDO{AppId: impl.appIdMap[impl.keyFunc(class.AppName, class.Namespace)]}).First(dest)
		assert.Equal(t, class.ClassId, dest.ClassId)
	}

	/*
		测试更新
	*/

	for i, class := range arr {
		class.ClassId = uint(10 + i)
		err := dao.SaveAppClass(class)
		assert.NoError(t, err)

		dest := &AppClassDO{}
		impl.db.Where(&AppClassDO{AppId: impl.appIdMap[impl.keyFunc(class.AppName, class.Namespace)]}).First(dest)
		assert.Equal(t, class.ClassId, dest.ClassId)
	}

	/*
		测试不存在的AppID
	*/
	appClass := &AppClass{
		AppName:   "absolutelyNotExistApp",
		Namespace: "absolutelyNotExistNamespace",
		ClassId:   10,
	}
	err := dao.SaveAppClass(appClass)
	assert.NoError(t, err)
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
	dao, _ := NewDao()
	classId := uint(10)
	c := &ClassMetrics{
		ClassId: classId,
		Data:    make([]*internal.ProcessedSectionData, internal.NumSections),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &internal.ProcessedSectionData{
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
		ID:                   classId,
		SectionNum:           10,
		ProcessedSectionData: internal.ProcessedSectionData{},
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
	c = &ClassMetrics{
		ClassId: classId,
		Data:    make([]*internal.ProcessedSectionData, 10),
	}
	for i := 0; i < len(c.Data); i++ {
		c.Data[i] = &internal.ProcessedSectionData{}
	}
	err = dao.SaveClassMetrics(c)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "插入不足数据出错")
	}
	_, err = dao.QueryClassMetricsByClassId(classId)
	assert.Error(t, err)
}

func TestDaoImpl_QueryAppClassIdByApp(t *testing.T) {
	dao, _ := NewDao()

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
		AppName:   appName,
		Namespace: namespace,
	})

	id, err := dao.QueryAppClassIdByApp(appName, namespace)
	assert.NoError(t, err)
	assert.Equal(t, classId, id)

	/*
		查询不存在的
	*/
	_, err = dao.QueryAppClassIdByApp("absolutelyNotExistApp", "definitelyNotExistNamespace")
	assert.Error(t, err)
}

func TestDaoImpl_QueryAllAppPodMetrics(t *testing.T) {
	dao, _ := NewDao()

	arr := make([]*AppPodMetrics, 0, 100)
	namespaceMap := map[string]struct {
		numApp int
	}{
		"namespace-1": {
			numApp: 1,
		},
		"namespace-2": {
			numApp: 2,
		},
		"namespace-3": {
			numApp: 3,
		},
		"namespace-4": {
			numApp: 4,
		},
	}
	appSize := 5

	for namespace, stu := range namespaceMap {
		for j := 0; j < stu.numApp; j++ {
			for k := 0; k < appSize; k++ {
				arr = append(arr, &AppPodMetrics{
					AppName:   fmt.Sprintf("test-%d", j),
					Namespace: namespace,
					Timestamp: uint64(k),
					Cpu:       float32(k),
					Mem:       float32(k),
				})
			}
		}
	}

	err := dao.SaveAllAppPodMetrics(arr)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "保存AppPodMetrics出错")
	}

	metricsMap, err := dao.QueryAllAppPodMetrics()
	assert.NoError(t, err)
	for namespace, namespaceAppMetricsMap := range metricsMap {
		if _, ok := namespaceMap[namespace]; !ok {
			// 跳过不属于本测试的数据
			continue
		}
		assert.Equal(t, namespaceMap[namespace].numApp, len(namespaceAppMetricsMap))
		for _, metrics := range namespaceAppMetricsMap {
			assert.Equal(t, appSize, len(metrics))
			for i, metric := range metrics {
				assert.Equal(t, uint64(i), metric.Timestamp)
			}
		}
	}

	db := dao.(*daoImpl).db
	err = db.Delete(&AppPodMetricsDO{}, "1 = 1").Error
	if !assert.NoError(t, err) {
		assert.FailNow(t, "删除所有AppPodMetrics记录失败")
	}

	noExistAppId := uint(2333)
	db.Create(&AppPodMetricsDO{
		Model:     gorm.Model{},
		AppId:     noExistAppId,
		Timestamp: 0,
		Cpu:       0,
		Mem:       0,
	})

	metricsMap, err = dao.QueryAllAppPodMetrics()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(metricsMap))
	m, ok := metricsMap[unknownNamespace]
	assert.True(t, ok)
	assert.Equal(t, 1, len(m))
	a, ok := m[fmt.Sprintf("%d", noExistAppId)]
	assert.True(t, ok)
	assert.Equal(t, 1, len(a))
}
