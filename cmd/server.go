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
	"github.com/packagewjx/workload-classifier/internal/server"
	"github.com/spf13/cobra"
	"time"
)

const (
	FlagPort            = "port"
	FlagScrapeInterval  = "interval"
	FlagMetricsDuration = "duration"
	FlagReClusterTime   = "re-cluster-time"
	FlagNumRound        = "round"
	FlagNumClass        = "class"
	FlagCenterFile      = "center-file"
	FlagMysqlHost       = "mysql-host"
)

var (
	port            uint16
	scrapeInterval  time.Duration
	metricsDuration time.Duration
	reClusterTime   time.Duration
	numRound        uint
	numClass        uint
	centerFile      string
	mysqlHost       string
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
			MetricDuration:       metricsDuration,
			Port:                 port,
			ScrapeInterval:       scrapeInterval,
			ReClusterTime:        reClusterTime,
			NumClass:             numClass,
			NumRound:             numRound,
			InitialCenterCsvFile: centerFile,
			MysqlHost:            mysqlHost,
		})
		if err != nil {
			return err
		}

		return server.Start()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().Uint16VarP(&port, FlagPort, "p", server.DefaultPort,
		"服务端口号")
	serverCmd.Flags().DurationVarP(&scrapeInterval, FlagScrapeInterval, "i", server.DefaultScrapeInterval,
		"获取监控数据的间隔，至少为15s")
	serverCmd.Flags().DurationVarP(&metricsDuration, FlagMetricsDuration, "d", server.DefaultMetricDuration,
		"保存数据的时间，至少为1天")
	serverCmd.Flags().DurationVarP(&reClusterTime, FlagReClusterTime, "t", server.DefaultReClusterTime,
		"每天定时跑聚类算法的时间，值应该小于24小时")
	serverCmd.Flags().UintVarP(&numRound, FlagNumRound, "r", server.DefaultNumRound,
		"聚类迭代次数")
	serverCmd.Flags().UintVarP(&numClass, FlagNumClass, "c", server.DefaultNumClass,
		"聚类类别数量")
	serverCmd.Flags().StringVarP(&centerFile, FlagCenterFile, "f", "",
		"初始中心文件。若不为空，则启动时将会读取此文件并作为各个类别的数据，此过程将删除旧有数据。若为空，则使用原类数据")
	serverCmd.Flags().StringVar(&mysqlHost, FlagMysqlHost, "",
		"Mysql服务器主机端口，格式为：host:port。若为空，则读取环境变量MYSQL_SERVICE_HOST与MYSQL_SERVICE_PORT取得")
}
