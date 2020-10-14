package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const PodMetricsListUrl = "http://localhost:8001/apis/metrics.k8s.io/v1beta1/pods"
const PodListUrl = "http://localhost:8001/api/v1/pods"

const (
	KindReplicaSet  = "ReplicaSet"
	KindDaemonSet   = "DaemonSet"
	KindStatefulSet = "StatefulSet"
)

const (
	DefaultPort           = 2000
	DefaultScrapeInterval = time.Minute
	DefaultMetricDuration = 7 * 24 * time.Hour
	DefaultReClusterTime  = 1 * time.Hour
	DefaultNumRound       = 30
	DefaultNumClass       = 20
)

const minDuration = 24 * time.Hour

type ServerConfig struct {
	MetricDuration       time.Duration // 给每个应用保留的数据的时间长度
	Port                 uint16        // 本服务器监听端口
	ScrapeInterval       time.Duration // 从metrics server获取数据的周期。至少为15s。
	ReClusterTime        time.Duration // 再聚类的时间
	NumClass             uint          // 类别数量
	NumRound             uint          // 聚类迭代轮次
	InitialCenterCsvFile string        // 初始各类中心的数据文件。若不是空，则会清空数据库的数据并读取。若为空，则使用数据库数据，此时如果数据库没有类别数据，则会产生错误。
}

func (s ServerConfig) String() string {
	return fmt.Sprintf("监听端口：%d。数据保留时长：%v。再聚类时间：%v。类别数量：%v。聚类迭代论次：%v。获取数据周期：%v。初始中心文件：'%v'。",
		s.Port, s.MetricDuration, s.ReClusterTime, s.NumClass, s.NumRound, s.ScrapeInterval, s.InitialCenterCsvFile)
}

type Server interface {
	Start() error
}

func NewServer(ctx *ServerConfig) (Server, error) {
	if err := checkConfig(ctx); err != nil {
		return nil, err
	}

	dao, err := NewDao()
	if err != nil {
		return nil, err
	}

	return &serverImpl{
		config: ctx,
		dao:    dao,
		logger: log.New(os.Stdout, "workload server: ", log.LstdFlags|log.Lshortfile|log.Lmsgprefix),
	}, nil
}

type serverImpl struct {
	config *ServerConfig
	dao    Dao
	logger *log.Logger
}

func checkConfig(config *ServerConfig) error {
	if config.Port < 1024 {
		return fmt.Errorf("端口号应该在1024到65535之间，现在为%d", config.Port)
	}

	if config.MetricDuration < minDuration {
		return fmt.Errorf("MetricDuration应该至少为%f小时，现在为%f小时", minDuration.Hours(), config.MetricDuration.Hours())
	}

	if config.ScrapeInterval < time.Second*15 {
		return fmt.Errorf("时间不能短于15s，现在是%fs", config.ScrapeInterval.Seconds())
	}

	// 限制重计算时间在24小时内，为一天内的时间
	config.ReClusterTime %= 24 * time.Hour

	if config.NumRound == 0 {
		return fmt.Errorf("聚类轮次不能为0")
	}
	if config.NumClass == 0 {
		return fmt.Errorf("聚类类别数目不能为0")
	}

	return nil
}

func (s *serverImpl) Start() error {
	rootCtx, cancel := context.WithCancel(context.Background())
	scraperContext, _ := context.WithCancel(rootCtx)
	reClustererContext, _ := context.WithCancel(rootCtx)

	s.logger.Printf("服务器启动。配置：%v\n", s.config)

	go s.scrapper(scraperContext)

	go s.reClusterer(reClustererContext)

	// 注册信号接收器
	termSigChan := make(chan os.Signal)
	signal.Notify(termSigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-termSigChan:
		cancel()
	}

	return nil
}