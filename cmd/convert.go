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
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MergeFilePathFlag = "mergeFile"
const OutputHeaderFlag = "outputHeader"
const DefaultReadRoutine = 4

var mergeFilePath string
var outputHeader bool

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use: "convert containerMetaFile inputFile...",
	Short: "将container_usage格式的文件转换为特征。默认输出文件将会是输入文件的文件名后加入" + alitrace.PreProcessOutputSuffix +
		"。若输入包含这些预处理输出文件，会跳过这些文件。",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("文件输入错误")
		}
		_, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return fmt.Errorf("元数据文件不存在")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("读取元数据文件中")
		metaFile = args[0]
		meta, err := alitrace.LoadContainerMeta(metaFile)
		if err != nil {
			fmt.Println("读取元数据文件错误", err)
			return
		}
		log.Println("读取元数据文件完毕")

		// 读取输入并预处理
		args = args[1:]
		var fileNames []string
		totalByte := int64(0)

		for _, filePattern := range args {
			glob, _ := filepath.Glob(filePattern)
			for _, fileName := range glob {
				if strings.Contains(fileName, alitrace.PreProcessOutputSuffix) {
					fmt.Printf("略过文件%s", fileName)
					continue
				}
				fileNames = append(fileNames, fileName)
				stat, _ := os.Stat(fileName)
				totalByte += stat.Size()
			}
		}

		if len(fileNames) == 0 {
			fmt.Println("没有文件处理，退出")
			return
		}

		cnt := int64(0)
		done := make(chan struct{})

		go func() {
			for {
				select {
				case <-done:
					fmt.Printf("\r预处理完成\n")
					return
				case <-time.After(time.Second):
					fmt.Printf("\r预处理进度：%8.2f%% (%d/%d)", float32(cnt)/float32(totalByte)*100, cnt, totalByte)
				}
			}
		}()

		// 如果合并文件，则需要检查输出路径是否已经存在文件，若存在，则不允许继续。
		// 这么做的原因是合并文件使用追加模式，可能导致与原本的数据混合。
		if mergeFilePath != "" {
			err := alitrace.PreProcessUsagesAndMerge(fileNames, mergeFilePath, meta, outputHeader, &cnt, DefaultReadRoutine)
			if err != nil {
				fmt.Printf("预处理错误：%v\n", err)
			}
		} else {
			err := alitrace.PreProcessUsages(fileNames, meta, &cnt, outputHeader, DefaultReadRoutine)
			if err != nil {
				fmt.Printf("预处理错误：%v\n", err)
			}
		}

		done <- struct{}{}
	},
}

func init() {
	preprocessCmd.AddCommand(convertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// convertCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	convertCmd.Flags().StringVarP(&mergeFilePath, MergeFilePathFlag, "m", "",
		"若设置这个为文件路径，则会将所有预处理的文件合并到一起，输出到这个文件中")
	convertCmd.Flags().BoolVarP(&outputHeader, OutputHeaderFlag, "t", false,
		"若设置，则输出表头到csv文件。默认不输出")
}
