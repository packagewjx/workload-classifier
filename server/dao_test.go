package server

import (
	"fmt"
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

	dao, err := NewDao()
	if err != nil {
		t.Fatal(err)
	}
	err = dao.SaveAllAppPodMetrics(arr)
	if err != nil {
		t.Fatal(err)
	}

	impl := dao.(*daoImpl)
	for i := 0; i < len(arr); i++ {
		dest := &AppPodMetricsDO{}
		impl.db.Where("app_id = ? AND timestamp = ?", impl.appIdMap[impl.keyFunc(arr[i].AppName, arr[i].Namespace)], arr[i].Timestamp).First(dest)
		assert.Equal(t, uint64(10000), dest.Timestamp)
	}

}
