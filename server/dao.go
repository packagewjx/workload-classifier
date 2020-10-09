package server

import (
	"crypto/md5"
	"fmt"
	"github.com/packagewjx/workload-classifier/internal"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
)

const databaseURL = "root:wujunxian@tcp(127.0.0.1:3306)/metrics?charset=utf8mb4&parseTime=True&loc=Local"
const unknownNamespace = "Unknown"

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
	QueryAppClassIdByApp(appName, namespace string) (uint, error)
	QueryAllAppPodMetrics() (map[string]map[string][]*AppPodMetrics, error)
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
		logger:   log.New(os.Stdout, "Dao: ", log.LstdFlags|log.Lshortfile),
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
	doarr := []*ClassSectionMetricsDO{}
	err := d.db.Order("section_num asc").Find(&doarr, &ClassSectionMetricsDO{
		ID: classId,
	}).Error
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("查询ClassSectionMetricsDO出错，classID为%d", classId))
	}

	// 检查数据是否正常
	for i := 0; i < len(doarr); i++ {
		if doarr[i].SectionNum != uint(i) {
			return nil, fmt.Errorf("ClassID为%d的各个数据错误，第%d条数据非第%d个Section，中间可能出现缺漏", classId, i, i)
		}
	}
	if len(doarr) != internal.NumSections {
		return nil, fmt.Errorf("ClassID为%d的数据不是%d个", classId, internal.NumSections)
	}

	result := &ClassMetrics{
		ClassId: classId,
		Data:    make([]*internal.ProcessedSectionData, len(doarr)),
	}

	for i := 0; i < len(doarr); i++ {
		result.Data[i] = &internal.ProcessedSectionData{}
		*result.Data[i] = doarr[i].ProcessedSectionData
	}

	return result, nil
}

func (d *daoImpl) QueryAppClassIdByApp(appName, namespace string) (uint, error) {
	appId, err := d.queryAppId(appName, namespace)
	if err != nil {
		return 0, err
	}

	record := &AppClassDO{}
	err = d.db.First(record, &AppClassDO{
		AppId: appId,
	}).Error
	if err != nil {
		return 0, errors.Wrap(err, "查询AppClass时出错")
	}

	return record.ClassId, nil
}

func (d *daoImpl) QueryAllAppPodMetrics() (map[string]map[string][]*AppPodMetrics, error) {
	doarr := []*AppPodMetricsDO{}
	err := d.db.Order("timestamp ASC").Find(&doarr).Error
	if err != nil {
		return nil, errors.Wrap(err, "查询所有AppPodMetrics记录出错")
	}

	type AppId struct {
		AppName   string
		Namespace string
	}
	appIdMap := make(map[uint]*AppId)

	result := make(map[string]map[string][]*AppPodMetrics)
	for _, do := range doarr {
		appId, ok := appIdMap[do.AppId]

		if !ok {
			dest := &AppDo{}
			err := d.db.First(dest, &AppDo{Model: gorm.Model{ID: do.AppId}}).Error
			if err == gorm.ErrRecordNotFound {
				log.Printf("不存在AppID为%d的记录，将会把名称设置为%d，命名空间为Unkonwn", do.AppId, do.AppId)
				dest.AppName = fmt.Sprintf("%d", do.AppId)
				dest.Namespace = unknownNamespace
			} else if err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("查询AppId时出错，ID为%d", do.AppId))
			}

			appId = &AppId{
				AppName:   dest.AppName,
				Namespace: dest.Namespace,
			}
			appIdMap[do.AppId] = appId
		}

		namespaceMap, ok := result[appId.Namespace]
		if !ok {
			namespaceMap = make(map[string][]*AppPodMetrics)
			result[appId.Namespace] = namespaceMap
		}

		metricsArr := namespaceMap[appId.AppName]
		metricsArr = append(metricsArr, &AppPodMetrics{
			AppName:   appId.AppName,
			Namespace: appId.Namespace,
			Timestamp: do.Timestamp,
			Cpu:       do.Cpu,
			Mem:       do.Mem,
		})
		namespaceMap[appId.AppName] = metricsArr
	}

	return result, nil
}

// 根据AppName和namespace查询AppID，若不存在，则创建一条记录。
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
