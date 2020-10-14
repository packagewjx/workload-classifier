package datasource

import "github.com/packagewjx/workload-classifier/internal"

type RawDataReader interface {
	Read() ([]*internal.ContainerRawData, error)
}

type MetricDataSource interface {
	// 读取一条容器监控数据。若读取完毕，则error设置为io.EOF。error为其他时表示读取出错
	Load() (*ContainerMetric, error)
}

type ContainerMetric struct {
	ContainerId string
	Cpu         float32
	Mem         float32
	Timestamp   uint64
}
