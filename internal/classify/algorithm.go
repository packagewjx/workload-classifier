package classify

import (
	"github.com/packagewjx/kmeanspp"
	"log"
)

// 聚类算法接口
type Algorithm interface {
	Run(data [][]float32, numClass int, context interface{}) (centers [][]float32, class []int)
}

type AlgorithmType string

const (
	KMeans = AlgorithmType("kmeans")
)

func GetAlgorithm(algorithmType AlgorithmType) Algorithm {
	switch algorithmType {
	case KMeans:
		return &kMeansRunner{}
	default:
		return nil
	}
}

type KMeansContext struct {
	Round int
}

const (
	KMeansDefaultRound = 30
)

type kMeansRunner struct {
}

func (k *kMeansRunner) Run(data [][]float32, numClass int, context interface{}) (centers [][]float32, class []int) {
	round := KMeansDefaultRound

	if context != nil {
		ctx, ok := context.(*KMeansContext)
		if !ok {
			log.Printf("输入的context不是KMeansContext类型。将使用默认参数")
		} else {
			round = ctx.Round
		}
	}

	return kmeanspp.KMeansPP(numClass, round, data)
}
