/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package workload_classifier

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/classify"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"os"
	"regexp"
	"strconv"
)

const (
	AlgorithmKMeans = "kmeans"
)

// Global Flags
const (
	AlgorithmFlag       = "algorithm"
	DataFormatFlag      = "dataFormat"
	RemoveColumnFlag    = "removeColumn"
	OutputPrecisionFlag = "outputPrecision"
)

// Global Defaults
const (
	DefaultOutputPrecision = 2
)

// Flags for K-Means
const (
	KMeansRoundFlag = "kMeansRound"
)

var algorithm string
var format string
var removeColumn []int
var outputPrecision int
var kMeansRound int

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster dataFile outputFile numClass",
	Short: "读取数据文件聚类计算，并输出结果到新文件中",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if format == "" {
			return fmt.Errorf("必须指定数据文件格式")
		} else if len(args) != 3 {
			return fmt.Errorf("参数错误")
		} else if args[0] == args[1] {
			return fmt.Errorf("dataFile与outputFile不能一致")
		}

		if match, _ := regexp.MatchString("^\\d+$", args[2]); !match {
			return fmt.Errorf("类数量参数不是数字")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var algType classify.AlgorithmType
		var context interface{}
		switch algorithm {
		default:
			algType = classify.KMeans
			context = &classify.KMeansContext{Round: kMeansRound}
		}
		alg := classify.GetAlgorithm(algType)

		var dataType classify.DataFormat
		switch format {
		default:
			dataType = classify.CSV
		}
		loader := classify.NewDataLoader(dataType)

		log.Println("读取数据中")
		inFile, err := os.Open(args[0])
		if err != nil {
			return errors.Wrap(err, "打开输入文件错误")
		}
		data, err := loader.Load(inFile, removeColumn)
		if err != nil {
			return errors.Wrap(err, "读取错误")
		}
		log.Println("读取数据完成")

		log.Println("运行K-Means算法中")
		numClass, err := strconv.ParseInt(args[2], 10, 64)
		centers, _ := alg.Run(data, int(numClass), context)
		log.Println("运行K-Means算法完成")

		fout, err := os.Create(args[1])
		if err != nil {
			return errors.Wrap(err, "创建输出文件错误")
		}
		err = classify.OutputResult(centers, fout, outputPrecision)
		if err != nil {
			return errors.Wrap(err, "输出文件错误")
		}

		_ = fout.Close()
		_ = inFile.Close()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)

	clusterCmd.Flags().StringVarP(&algorithm, AlgorithmFlag, "a", AlgorithmKMeans,
		"指定使用的算法。默认为kmeans，可选值：kmeans")
	clusterCmd.Flags().StringVarP(&format, DataFormatFlag, "f", "",
		"数据文件格式")
	clusterCmd.Flags().IntSliceVarP(&removeColumn, RemoveColumnFlag, "r", []int{},
		"需要移除的列号，从0开始计算。使用此字段忽略掉不是数字的列")
	clusterCmd.Flags().IntVarP(&outputPrecision, OutputPrecisionFlag, "p", DefaultOutputPrecision,
		"输出文件数据精度，默认为2")

	// Flags for K-Means Algorithm
	clusterCmd.Flags().IntVar(&kMeansRound, KMeansRoundFlag, classify.KMeansDefaultRound,
		"K-Means算法执行的轮次")
}
