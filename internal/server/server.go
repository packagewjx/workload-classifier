package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
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
	MysqlHost            string
}

func (s ServerConfig) String() string {
	marshal, _ := json.Marshal(s)
	return string(marshal)
}

type Server interface {
	Start() error
}

func NewServer(config *ServerConfig) (Server, error) {
	if err := config.Complete(); err != nil {
		return nil, err
	}

	dao, err := NewDao(config.MysqlHost)
	if err != nil {
		return nil, err
	}

	return &serverImpl{
		config:           config,
		dao:              dao,
		logger:           log.New(os.Stdout, "workload server: ", log.LstdFlags|log.Lshortfile|log.Lmsgprefix),
		executeReCluster: make(chan struct{}),
	}, nil
}

type serverImpl struct {
	config           *ServerConfig
	dao              Dao
	logger           *log.Logger
	executeReCluster chan struct{}
}

func (config *ServerConfig) Complete() error {
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

	if config.MysqlHost == "" {
		config.MysqlHost = fmt.Sprintf("%s:%s",
			os.Getenv("MYSQL_SERVICE_HOST"), os.Getenv("MYSQL_SERVICE_PORT"))
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

	server := s.buildServer()
	errCh := make(chan error)
	go s.serve(server, errCh)

	// 注册信号接收器
	termSigChan := make(chan os.Signal)
	signal.Notify(termSigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-termSigChan:
		err := server.Shutdown(rootCtx)
		if err != nil {
			return errors.Wrap(err, "关闭HTTP服务器失败")
		}
		cancel()
	}

	// 等待HTTP服务器结束
	err := <-errCh
	if err != nil {
		return errors.Wrap(err, "HTTP关闭出现错误")
	}

	return nil
}

func (s *serverImpl) buildServer() *http.Server {
	mux := http.NewServeMux()
	const NamePattern = "[\\d\\w]|(?:[\\d\\w][\\d\\w-.]{0,251}[\\d\\w])"
	mux.HandleFunc("/namespaces/", func(writer http.ResponseWriter, request *http.Request) {
		pattern := regexp.MustCompile(fmt.Sprintf("/namespaces/(%s)/appclass/(%s)", NamePattern, NamePattern))
		if !pattern.MatchString(request.URL.Path) {
			http.NotFound(writer, request)
			return
		}
		subMatch := pattern.FindStringSubmatch(request.URL.Path)
		namespace := subMatch[1]
		name := subMatch[2]

		class, err := s.QueryAppClass(AppName{
			Name:      name,
			Namespace: namespace,
		})
		if err == ErrAppNotFound {
			http.NotFound(writer, request)
			return
		} else if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		marshal, err := json.Marshal(class)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json")
		_, err = writer.Write(marshal)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/recluster", func(writer http.ResponseWriter, request *http.Request) {
		s.ReCluster()
		_, _ = writer.Write([]byte("OK"))
	})

	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}
	return srv
}

func (s *serverImpl) serve(server *http.Server, errCh chan<- error) {
	s.logger.Printf("API服务器启动")

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		errCh <- err
	}

	s.logger.Printf("API服务器结束")
	errCh <- nil
}
