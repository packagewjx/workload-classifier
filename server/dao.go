package server

import (
	"crypto/md5"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const databaseURL = "root:wujunxian@tcp(127.0.0.1:3306)/metrics?charset=utf8mb4&parseTime=True&loc=Local"

func init() {
	_, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		panic(err)
	}
}

type UpdateDao interface {
	SaveClassMetrics(c *ClassMetrics) error
	SaveAppClass(a *AppClass) error
	SaveAllAppPodMetrics(arr []*AppPodMetrics) error

	RemoveAppPodMetricsBefore(timestamp int) error
}

type QueryDao interface {
	QueryClassMetricsByClassId(classId int) (*ClassMetrics, error)
	QueryAppClassIdByName(appName, namespace string) (int, error)
	QueryAllAppPodMetrics() (map[string][]*AppPodMetrics, error)
}

type Dao interface {
	UpdateDao
	QueryDao
}

type daoImpl struct {
	db       *gorm.DB
	appIdMap map[string]uint
	keyFunc  func(appName, namespace string) string
}

func NewDao() (Dao, error) {
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "连接数据库错误")
	}

	// 转换为单一字符串的函数
	keyFunc := func(appName, namespace string) string {
		sum := md5.Sum([]byte(appName + namespace))
		return string(sum[:])
	}

	// 创建表格等
	err = db.AutoMigrate(&AppPodMetricsDO{}, &AppClassDO{}, &ClassSectionMetricsDO{}, &AppDo{})
	if err != nil {
		return nil, errors.Wrap(err, "创建表格时出现异常")
	}

	// 读取AppID
	appIdMap := make(map[string]uint)
	appRecords := make([]*AppDo, 0)
	err = db.Find(&appRecords).Error
	if err != nil {
		return nil, errors.Wrap(err, "读取应用记录时出错")
	}
	for _, record := range appRecords {
		appIdMap[keyFunc(record.AppName, record.Namespace)] = record.ID
	}

	return &daoImpl{
		db:       db,
		appIdMap: make(map[string]uint),
		keyFunc:  keyFunc,
	}, nil
}

func (d *daoImpl) SaveClassMetrics(c *ClassMetrics) error {
	panic("implement me")
}

func (d *daoImpl) queryAppId(name, namespace string) (uint, error) {
	key := d.keyFunc(name, namespace)
	id, ok := d.appIdMap[key]
	if ok {
		return id, nil
	}

	app := &AppDo{}
	err := d.db.FirstOrCreate(app, &AppDo{
		Model:     gorm.Model{},
		AppName:   name,
		Namespace: namespace,
	}).Error
	if err != nil {
		return 0, errors.Wrap(err, "创建App记录出错")
	}

	d.appIdMap[key] = app.ID

	return app.ID, nil
}

func (d *daoImpl) SaveAppClass(a *AppClass) error {
	panic("implement me")
}

func (d *daoImpl) SaveAllAppPodMetrics(arr []*AppPodMetrics) error {
	doarr := make([]*AppPodMetricsDO, len(arr))
	for i, metrics := range arr {
		id, err := d.queryAppId(metrics.AppName, metrics.Namespace)
		if err != nil {
			return err
		}
		doarr[i] = &AppPodMetricsDO{
			Model:     gorm.Model{},
			AppId:     id,
			Timestamp: arr[i].Timestamp,
			Cpu:       arr[i].Cpu,
			Mem:       arr[i].Mem,
		}
	}

	return d.db.Save(doarr).Error
}

func (d *daoImpl) RemoveAppPodMetricsBefore(timestamp int) error {
	panic("implement me")
}

func (d *daoImpl) QueryClassMetricsByClassId(classId int) (*ClassMetrics, error) {
	panic("implement me")
}

func (d *daoImpl) QueryAppClassIdByName(appName, namespace string) (int, error) {
	panic("implement me")
}

func (d *daoImpl) QueryAllAppPodMetrics() (map[string][]*AppPodMetrics, error) {
	panic("implement me")
}
