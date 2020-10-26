package preprocess

import (
	"github.com/packagewjx/workload-classifier/pkg/core"
)

type Preprocessor interface {
	Preprocess(workload *core.ContainerWorkloadData)
}

type defaultPreprocess struct {
	chain []Preprocessor
}

func (d *defaultPreprocess) Preprocess(workload *core.ContainerWorkloadData) {
	for _, processor := range d.chain {
		processor.Preprocess(workload)
	}
}

func Default() Preprocessor {
	return &defaultPreprocess{chain: []Preprocessor{Impute(), Normalize()}}
}
