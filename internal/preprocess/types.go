package preprocess

import (
	"github.com/packagewjx/workload-classifier/internal"
)

type Preprocessor interface {
	Preprocess(workload *internal.ContainerWorkloadData)
}

type defaultPreprocess struct {
	chain []Preprocessor
}

func (d *defaultPreprocess) Preprocess(workload *internal.ContainerWorkloadData) {
	for _, processor := range d.chain {
		processor.Preprocess(workload)
	}
}

func Default() Preprocessor {
	return &defaultPreprocess{chain: []Preprocessor{Impute(), Normalize()}}
}
