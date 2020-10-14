package datasource

import "github.com/packagewjx/workload-classifier/internal"

type RawDataReader interface {
	Read() []*internal.ContainerRawData
}
