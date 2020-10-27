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
	"github.com/packagewjx/workload-classifier/internal/alitrace"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

const (
	MetaFileFlag  = "metaFile"
	UsageFileFlag = "usageFile"
)

var (
	metaFile  string
	usageFile string
)

// splitCmd represents the split command
var splitCmd = &cobra.Command{
	Use:   "split",
	Short: "根据AppDU将container_usage.csv分割成小文件",
	RunE: func(cmd *cobra.Command, args []string) error {
		fin, err := os.Open(metaFile)
		if err != nil {
			return errors.Wrap(err, "打开元数据文件错误")
		}
		meta, err := alitrace.LoadContainerMeta(fin)
		if err != nil {
			return errors.Wrap(err, "读取元数据错误")
		}
		_ = fin.Close()

		fin, err = os.Open(usageFile)
		if err != nil {
			return errors.Wrap(err, "打开输入文件错误")
		}
		finCounter := &utils.ReadCounter{
			Count:  0,
			Reader: fin,
		}
		err = alitrace.SplitContainerUsage(finCounter, meta)
		if err != nil {
			return errors.Wrap(err, "分割出错")
		}
		_ = fin.Close()

		return nil
	},
}

func init() {
	preprocessCmd.AddCommand(splitCmd)

	splitCmd.Flags().StringVarP(&metaFile, MetaFileFlag, "m", "",
		"container_meta.csv的文件路径")
	splitCmd.Flags().StringVarP(&usageFile, UsageFileFlag, "u", "",
		"container_usage.csv的文件路径")
	_ = splitCmd.MarkFlagRequired(MetaFileFlag)
	_ = splitCmd.MarkFlagRequired(UsageFileFlag)
}
