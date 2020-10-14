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
	"github.com/packagewjx/workload-classifier/internal/preprocess"
	"github.com/pkg/errors"
	"os"

	"github.com/spf13/cobra"
)

// normalizeCmd represents the normalize command
var normalizeCmd = &cobra.Command{
	Use:   "normalize infile outfile",
	Short: "将转换后的数据标准化",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			return fmt.Errorf("参数错误")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		in, err := os.Open(args[0])
		if err != nil {
			return errors.Wrap(err, "打开输入文件错误")
		}
		out, err := os.Create(args[1])
		if err != nil {
			return errors.Wrap(err, "创建输出文件错误")
		}
		defer func() {
			_ = out.Close()
		}()

		return preprocess.NormalizeSection(in, out)
	},
}

func init() {
	preprocessCmd.AddCommand(normalizeCmd)
}
