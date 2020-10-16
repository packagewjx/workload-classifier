package server

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/packagewjx/workload-classifier/internal/alitrace"
	"github.com/packagewjx/workload-classifier/internal/preprocess"
	"github.com/stretchr/testify/assert"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestReadInitialCenters(t *testing.T) {
	f, _ := os.Open("../../test/csv/centers.csv")
	centers, err := readInitialCenter(f)
	assert.NoError(t, err)
	assert.Equal(t, 20, len(centers))
	for _, center := range centers {
		assert.Equal(t, internal.NumSections, len(center.Data))
		for _, datum := range center.Data {
			v := reflect.ValueOf(*datum)
			for i := 0; i < v.NumField(); i++ {
				f := v.Field(i).Float()
				assert.NotEqual(t, 0, f)
				assert.False(t, math.IsNaN(f))
			}
		}
	}

	// 读取错误的csv数据
	f, _ = os.Open("/dev/null")
	_, err = readInitialCenter(f)
	assert.Error(t, err)

	reader := strings.NewReader(",")
	_, err = readInitialCenter(reader)
	assert.Error(t, err)

	// 读取一行中间有错误数据的数据
	falseString := make([]string, internal.NumSectionFields*internal.NumSections)
	for i := 0; i < len(falseString); i++ {
		falseString[i] = strconv.FormatInt(int64(i), 10)
	}
	falseString[internal.NumSections] = ""
	reader = strings.NewReader(strings.Join(falseString, ","))
	_, err = readInitialCenter(reader)
	assert.Error(t, err)
}

func TestFloatArrayToClassMetrics(t *testing.T) {
	arr := make([]float32, internal.NumSections*internal.NumSectionFields)
	for i := 0; i < len(arr); i++ {
		arr[i] = float32(i)
	}
	metrics := floatArrayToClassMetrics(1, arr)
	assert.Equal(t, uint(1), metrics.ClassId)
	assert.Equal(t, internal.NumSections, len(metrics.Data))
	assert.Equal(t, float32(1), metrics.Data[0].CpuMax)
}

func TestReCluster(t *testing.T) {
	// 准备测试数据
	dao, err := NewDao(testHost)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "DAO创建失败")
		return
	}
	s := &serverImpl{
		config: &ServerConfig{
			MetricDuration:       0,
			Port:                 0,
			ScrapeInterval:       0,
			ReClusterTime:        0,
			NumClass:             DefaultNumClass,
			NumRound:             DefaultNumRound,
			InitialCenterCsvFile: "",
		},
		dao:    dao,
		logger: log.New(os.Stdout, "TestServer", log.LstdFlags),
	}

	// 删除无关数据
	dao.DB().Delete(&ClassSectionMetricsDO{}, "1 = 1")
	dao.DB().Delete(&AppClassDO{}, "1 = 1")
	dao.DB().Delete(&AppPodMetricsDO{}, "1 = 1")

	// 导入类别数据
	centerFin, _ := os.Open("../../test/csv/centers.csv")
	center, err := readInitialCenter(centerFin)
	preprocessor := preprocess.Default()
	for i, metrics := range center {
		metrics.ClassId = uint(i + 1)
		temp := &internal.ContainerWorkloadData{
			ContainerId: fmt.Sprintf("%d", metrics.ClassId),
			Data:        metrics.Data,
		}
		preprocessor.Preprocess(temp)
		err := dao.SaveClassMetrics(metrics)
		if !assert.NoError(t, err) {
			assert.FailNow(t, "保存类别数据失败")
		}
	}

	// 导入监控数据
	metricsFin, _ := os.Open("../../test/csv/30containers.csv")
	datasource := alitrace.NewAlitraceDatasource(metricsFin)
	podMetrics := make([]*AppPodMetrics, 0, 30000)
	appNameSet := map[AppName]struct{}{}
	for r, err := datasource.Load(); err == nil; r, err = datasource.Load() {
		pm := &AppPodMetrics{
			AppName: AppName{
				Name:      r.ContainerId,
				Namespace: "test",
			},
			Timestamp: r.Timestamp,
			Cpu:       r.Cpu,
			Mem:       r.Mem,
		}
		podMetrics = append(podMetrics, pm)
		appNameSet[pm.AppName] = struct{}{}
	}
	_ = metricsFin.Close()
	err = dao.SaveAllAppPodMetrics(podMetrics)
	if !assert.NoError(t, err) {
		assert.FailNow(t, "保存容器监控数据失败")
	}
	podMetrics = nil

	// 测试开始
	err = s.reCluster()
	assert.NoError(t, err)

	// 检验聚类结果
	classCount := map[uint]int{}
	cpuMaxAllZero := true
	memMaxAllZero := true
	for appName := range appNameSet {
		class, err := dao.QueryAppClassByApp(&appName)
		assert.NoError(t, err)
		classCount[class.ClassId] = classCount[class.ClassId] + 1
		if class.CpuMax != 0 {
			cpuMaxAllZero = false
		}
		if class.MemMax != 0 {
			memMaxAllZero = false
		}
	}
	assert.False(t, cpuMaxAllZero)
	assert.False(t, memMaxAllZero)
	for _, cnt := range classCount {
		assert.Condition(t, func() (success bool) {
			return cnt != len(appNameSet)
		})
	}

	// 检查类别是否更新
	newClassMetrics, err := dao.QueryAllClassMetrics()
	if !assert.NoError(t, err) {
		assert.FailNow(t, "获取类别数据失败")
	}
	assert.Equal(t, int(s.config.NumClass), len(newClassMetrics))
	for _, m := range center {
		for _, n := range newClassMetrics {
			assert.Equal(t, len(m.Data), len(n.Data))
			equal := true
			for i := 0; i < len(m.Data); i++ {
				if m.Data[i].CpuAvg != n.Data[i].CpuAvg {
					equal = false
					break
				}
			}
			if equal {
				assert.Fail(t, "类别数据没有更新")
			}
		}
	}

}
