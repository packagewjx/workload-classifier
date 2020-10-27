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
	"github.com/packagewjx/workload-classifier/internal/preprocess"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

// imputeCmd represents the impute command
var imputeCmd = &cobra.Command{
	Use:   "impute inputFile outputFile",
	Short: "填充NaN数据",
	Long: "使用线性函数方式填充NaN数据。两端如果有NaN，则计算一次函数时默认开始或结束的数据为0,另一个端点则为第一个或最后一个有效值。" +
		"若中间为NaN，则使用两端有效数据计算一次函数并填充NaN。",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if 2 != len(args) {
			return errors.New("参数错误")
		} else if args[0] == args[1] {
			return errors.New("输入输出不能一致")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		fin, err := os.Open(args[0])
		if err != nil {
			return errors.Wrap(err, "打开输入文件错误")
		}
		fout, err := os.Create(args[1])
		if err != nil {
			return errors.Wrap(err, "创建输出文件错误")
		}

		return preprocess.ImputeMissingValues(fin, fout)
	},
}

func init() {
	preprocessCmd.AddCommand(imputeCmd)
}
