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
package cmd

import (
	"fmt"
	"github.com/packagewjx/workload-classifier/internal/classify"
	"github.com/spf13/cobra"
	"log"
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
	Run: func(cmd *cobra.Command, args []string) {
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
		data, err := loader.Load(args[0], removeColumn)
		if err != nil {
			log.Fatalln("读取错误", err)
			return
		}
		log.Println("读取数据完成")

		log.Println("运行K-Means算法中")
		numClass, err := strconv.ParseInt(args[2], 10, 64)
		centers, _ := alg.Run(data, int(numClass), context)
		log.Println("运行K-Means算法完成")

		err = classify.OutputResult(centers, args[1], outputPrecision)
		if err != nil {
			log.Fatalln("输出文件错误", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(clusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

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
