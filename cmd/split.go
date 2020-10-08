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
	"github.com/packagewjx/workload-classifier/internal/alitrace"
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
	Short: "根据AppDU，将container_usage.csv分割成小文件，然后结束运行",
	Run: func(cmd *cobra.Command, args []string) {
		meta, err := alitrace.LoadContainerMeta(metaFile)
		if err != nil {
			fmt.Printf("读取元数据错误：%v\n", err)
			os.Exit(1)
		}

		cnt := 0
		err = alitrace.SplitContainerUsage(usageFile, meta, &cnt)
		if err != nil {
			fmt.Printf("分割出错：%v\n", err)
			os.Exit(1)
		}
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
