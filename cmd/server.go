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
	"github.com/packagewjx/workload-classifier/server"
	"github.com/spf13/cobra"
	"time"
)

const (
	FlagPort            = "port"
	FlagScrapeInterval  = "interval"
	FlagMetricsDuration = "duration"
	FlagReClusterTime   = "re-cluster-time"
)

var (
	port            int
	scrapeInterval  time.Duration
	metricsDuration time.Duration
	reClusterTime   time.Duration
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "负载分类服务器",
	Long: "本服务器将从Kubernetes的metrics server中每隔一段时间（通过interval指定）获取一次监控数据，并保存记录。\n" +
		"监控数据仅保留最近一段时间（通过duration设置）。服务器每天定时（通过re-cluster-time设置）聚类，更新分类数据，\n" +
		"以确保数据反映近期的真实情况。用户可以通过本服务器提供的接口获取应用属于哪个类别的数据。\n",
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := server.NewServer(&server.ServerConfig{
			MetricDuration: metricsDuration,
			Port:           port,
			ScrapeInterval: scrapeInterval,
			ReClusterTime:  reClusterTime,
		})
		if err != nil {
			return err
		}

		return server.Start()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serverCmd.Flags().IntVarP(&port, FlagPort, "p", server.DefaultPort,
		fmt.Sprintf("服务端口号，默认为%d", server.DefaultPort))
	serverCmd.Flags().DurationVarP(&scrapeInterval, FlagScrapeInterval, "i", server.DefaultScrapeInterval,
		fmt.Sprintf("获取监控数据的间隔，至少为15s。默认为%f秒", server.DefaultScrapeInterval.Seconds()))
	serverCmd.Flags().DurationVarP(&metricsDuration, FlagMetricsDuration, "d", server.DefaultMetricDuration,
		fmt.Sprintf("保存数据的时间，至少为1天。默认为%f小时", server.DefaultMetricDuration.Hours()))
	serverCmd.Flags().DurationVarP(&reClusterTime, FlagReClusterTime, "t", server.DefaultReClusterTime,
		fmt.Sprintf("每天定时跑聚类算法的时间，值应该小于24小时。默认为%f时", server.DefaultReClusterTime.Hours()))
}
