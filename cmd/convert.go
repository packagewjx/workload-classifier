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
	"github.com/packagewjx/workload-classifier/internal/datasource"
	"github.com/packagewjx/workload-classifier/internal/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"time"
)

const OutputHeaderFlag = "outputHeader"

var outputHeader bool

type readCounter struct {
	readCount *uint64
	reader    io.ReadCloser
}

func (r readCounter) Close() error {
	return r.reader.Close()
}

func (r readCounter) Read(p []byte) (n int, err error) {
	read, err := r.reader.Read(p)
	*r.readCount += uint64(read)
	return read, err
}

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert outputFile inputFile...",
	Short: "将container_usage格式的文件转换为特征，并输出到文件中",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("命令错误")
		}
		_, err := os.Stat(args[0])
		if !os.IsNotExist(err) {
			return fmt.Errorf("输出文件已存在")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		readCount := uint64(0)
		readCloser, totalByte, err := openFilesWithReadCounter(args[1:], &readCount)
		if len(readCloser) == 0 {
			return errors.Wrap(err, "没有文件处理，退出")
		}

		fout, err := os.OpenFile(args[0], os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0666)
		defer func() {
			_ = fout.Close()
		}()

		done := make(chan struct{})
		defer func() {
			done <- struct{}{}
		}()
		go func() {
			for {
				select {
				case <-done:
					fmt.Printf("\r预处理完成\n")
					return
				case <-time.After(time.Second):
					fmt.Printf("\r预处理进度：%8.2f%% (%d/%d)", float32(readCount)/float32(totalByte)*100, readCount, totalByte)
				}
			}
		}()

		// 打开文件并设置readCounter
		return merge(readCloser, fout)
	},
}

func merge(readCloser []io.ReadCloser, fout io.Writer) error {
	if outputHeader {
		err := utils.WriteContainerWorkloadHeader(fout)
		if err != nil {
			return errors.Wrap(err, "写入表头出错")
		}
	}

	for _, rc := range readCloser {
		dataSource := alitrace.NewAlitraceDatasource(rc)
		rawData, err := datasource.NewDataSourceRawDataReader(dataSource).Read()
		if err != nil {
			return errors.Wrap(err, "读取输入文件出错")
		}
		workloadData := datasource.ConvertAllRawData(rawData)
		_ = rc.Close()

		err = utils.WriteContainerWorkloadData(fout, workloadData)
		if err != nil {
			return errors.Wrap(err, "写入出错")
		}
	}

	return nil
}

func openFilesWithReadCounter(filePatterns []string, countBuf *uint64) (outputs []io.ReadCloser, totalByte int64, err error) {
	outputs = make([]io.ReadCloser, 0, len(filePatterns))
	// 读取输入并预处理
	for _, filePattern := range filePatterns {
		glob, _ := filepath.Glob(filePattern)
		for _, fileName := range glob {
			out, err := os.Open(fileName)
			if err != nil {
				return nil, 0, errors.Wrap(err, fmt.Sprintf("打开文件%s失败", fileName))
			}
			outputs = append(outputs, &readCounter{
				readCount: countBuf,
				reader:    out,
			})
			stat, _ := os.Stat(fileName)
			totalByte += stat.Size()
		}
	}
	return outputs, totalByte, err
}

func init() {
	preprocessCmd.AddCommand(convertCmd)

	convertCmd.Flags().BoolVarP(&outputHeader, OutputHeaderFlag, "t", false,
		"若设置，则输出表头到csv文件。默认不输出")
}
