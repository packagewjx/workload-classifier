package server

import (
	"container/ring"
	. "github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"io"
)

const onetimeReadSize = 500

func NewDatabaseDatasource(db *gorm.DB) MetricDataSource {
	return &dbDatasource{
		lastId: 0,
		db:     db,
		buffer: ring.New(onetimeReadSize),
	}
}

type dbDatasource struct {
	lastId uint
	db     *gorm.DB
	buffer *ring.Ring
}

func (d *dbDatasource) Load() (*ContainerMetric, error) {
	val := d.buffer.Value
	if val == nil {
		err := d.doLoad()
		if err != nil {
			return nil, err
		}
		val = d.buffer.Value
	}

	d.buffer.Value = nil
	d.buffer = d.buffer.Next()
	return val.(*ContainerMetric), nil
}

func (d *dbDatasource) doLoad() error {
	result := []*AppPodMetricsDO{}
	err := d.db.Limit(onetimeReadSize).Order("id ASC").Where("id > ?", d.lastId).Find(&result).Error
	if err != nil {
		return errors.Wrap(err, "读取数据库AppPodMetrics时出错")
	}
	if len(result) == 0 {
		// 所有数据读取完毕
		return io.EOF
	}

	idMap, err := queryContainerId(d.db, result)
	if err != nil {
		return err
	}

	d.lastId = result[len(result)-1].ID
	buf := d.buffer

	for _, do := range result {
		buf.Value = &ContainerMetric{
			ContainerId: idMap[do.AppId],
			Cpu:         do.Cpu,
			Mem:         do.Mem,
			Timestamp:   do.Timestamp,
		}
		buf = buf.Next()
	}

	return nil
}

func queryContainerId(db *gorm.DB, arr []*AppPodMetricsDO) (map[uint]string, error) {
	idSet := map[uint]struct{}{}
	for _, do := range arr {
		idSet[do.AppId] = struct{}{}
	}
	idList := make([]uint, 0, len(idSet))
	for id := range idSet {
		idList = append(idList, id)
	}
	appDos := make([]*AppDo, 0, len(idList))
	err := db.Where("id IN ?", idList).Find(&appDos).Error
	if err != nil {
		return nil, errors.Wrap(err, "查询AppName出错")
	}

	result := make(map[uint]string)
	for _, do := range appDos {
		result[do.ID] = do.AppName.ContainerId()
	}
	return result, nil
}
