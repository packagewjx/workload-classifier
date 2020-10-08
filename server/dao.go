package server

import (
	"crypto/md5"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
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

	RemoveAppPodMetricsBefore(timestamp uint64) error
}

type QueryDao interface {
	QueryClassMetricsByClassId(classId uint) (*ClassMetrics, error)
	QueryAppClassIdByName(appName, namespace string) (uint, error)
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
	logger   *log.Logger
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
		logger:   log.New(os.Stdout, "Dao: ", log.LstdFlags),
	}, nil
}

func (d *daoImpl) SaveClassMetrics(c *ClassMetrics) error {
	doarr := make([]*ClassSectionMetricsDO, len(c.Data))
	for i, datum := range c.Data {
		doarr[i] = &ClassSectionMetricsDO{
			ID:                   c.ClassId,
			SectionNum:           uint(i),
			ProcessedSectionData: *datum,
		}
	}

	d.logger.Printf("正在插入ClassID为%d的ClassMetrics", c.ClassId)

	return d.db.Save(doarr).Error
}

func (d *daoImpl) SaveAppClass(a *AppClass) error {
	appId, err := d.queryAppId(a.AppName, a.Namespace)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("查询名称为%s，命名空间为%s的AppID时出错", a.AppName, a.Namespace))
	}

	dest := &AppClassDO{}
	err = d.db.First(dest, &AppClassDO{
		AppId: appId,
	}).Error

	if err == gorm.ErrRecordNotFound {
		d.logger.Printf("不存在AppID为%d，ClassID为%d的记录，正在数据库中创建", appId, a.ClassId)
		err = d.db.Create(&AppClassDO{
			AppId:   appId,
			ClassId: a.ClassId,
		}).Error
	} else {
		dest.ClassId = a.ClassId
		err = d.db.Updates(dest).Error
	}

	if err != nil {
		return errors.Wrap(err, "保存AppClassDO出错，AppID为%d，ClassID为%d")
	}

	return nil
}

func (d *daoImpl) SaveAllAppPodMetrics(arr []*AppPodMetrics) error {
	const MaxOneRun = 5000

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

	log.Printf("保存%d条AppPodMetrics到数据库", len(arr))

	for i := 0; i < len(doarr); i += MaxOneRun {
		end := i + MaxOneRun
		if end > len(doarr) {
			end = len(doarr)
		}
		err := d.db.Create(doarr[i:end]).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *daoImpl) RemoveAppPodMetricsBefore(timestamp uint64) error {
	return d.db.Model(&AppPodMetricsDO{}).Where("timestamp < ?", timestamp).Delete(&AppPodMetricsDO{}).Error
}

func (d *daoImpl) QueryClassMetricsByClassId(classId uint) (*ClassMetrics, error) {
	panic("implement me")
}

func (d *daoImpl) QueryAppClassIdByName(appName, namespace string) (uint, error) {
	panic("implement me")
}

func (d *daoImpl) QueryAllAppPodMetrics() (map[string][]*AppPodMetrics, error) {
	panic("implement me")
}

func (d *daoImpl) queryAppId(name, namespace string) (uint, error) {
	key := d.keyFunc(name, namespace)
	id, ok := d.appIdMap[key]
	if ok {
		return id, nil
	}

	d.logger.Printf("没有找到名称为%s，命名空间为%s的ID记录，将从数据库中获取", name, namespace)

	app := &AppDo{}
	err := d.db.FirstOrCreate(app, &AppDo{
		Model:     gorm.Model{},
		AppName:   name,
		Namespace: namespace,
	}).Error
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("从数据库中查询或创建App记录出错。名称为%s，命名空间为%s", name, namespace))
	}

	d.appIdMap[key] = app.ID

	return app.ID, nil
}
