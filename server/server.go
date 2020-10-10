package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"log"
	"net/http"
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
)

const minDuration = 24 * time.Hour

type ServerConfig struct {
	MetricDuration time.Duration // 给每个应用保留的数据的时间长度
	Port           int           // 本服务器监听端口
	ScrapeInterval time.Duration // 从metrics server获取数据的周期。至少为15s。
	ReClusterTime  time.Duration // 再聚类的时间
}

func (s ServerConfig) String() string {
	return fmt.Sprintf("监听端口：%d。数据保留时长：%v。再聚类时间：%v。获取数据周期：%v",
		s.Port, s.MetricDuration, s.ReClusterTime, s.ScrapeInterval)
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

func checkConfig(ctx *ServerConfig) error {
	if ctx.Port > 65535 || ctx.Port < 1024 {
		return fmt.Errorf("端口号应该在1024到65535之间，现在为%d", ctx.Port)
	}

	if ctx.MetricDuration < minDuration {
		return fmt.Errorf("MetricDuration应该至少为%f小时，现在为%f小时", minDuration.Hours(), ctx.MetricDuration.Hours())
	}

	if ctx.ScrapeInterval < time.Second*15 {
		return fmt.Errorf("时间不能短于15s，现在是%fs", ctx.ScrapeInterval.Seconds())
	}

	// 限制重计算时间在24小时内，为一天内的时间
	ctx.ReClusterTime %= 24 * time.Hour

	return nil
}

func (s *serverImpl) Start() error {
	rootCtx, cancel := context.WithCancel(context.Background())
	scraperContext, _ := context.WithCancel(rootCtx)

	s.logger.Printf("服务器启动。配置：%v\n", s.config)

	go s.scrapper(scraperContext)

	// 注册信号接收器
	termSigChan := make(chan os.Signal)
	signal.Notify(termSigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-termSigChan:
		cancel()
	}

	return nil
}

// 用于从metrics server获取数据并保存到数据库的goroutine主函数
func (s *serverImpl) scrapper(ctx context.Context) {
	s.logger.Println("监控数据获取线程启动")
	for {
		select {
		case <-time.After(s.config.ScrapeInterval):
			podMetrics, err := s.scrapePodMetrics()
			if err != nil {
				panic(err)
			}
			err = s.dao.SaveAllAppPodMetrics(podMetrics)
			if err != nil {
				panic(err)
			}
		case <-ctx.Done():
			s.logger.Println("监控数据获取线程结束")
			return
		}
	}
}

func (s *serverImpl) scrapePodMetrics() ([]*AppPodMetrics, error) {
	listFunc := func(url string, dest interface{}) error {
		response, err := http.Get(url)
		if err != nil {
			return err
		}
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, dest)
		if err != nil {
			return err
		}
		return nil
	}
	keyFunc := func(name, namespace string) string {
		return name + namespace
	}

	s.logger.Println("正在从api server获取PodList")
	podList := &corev1.PodList{}
	err := listFunc(PodListUrl, podList)
	if err != nil {
		return nil, errors.Wrap(err, "请求PodList出错")
	}
	s.logger.Printf("获取了%d条PodList数据\n", len(podList.Items))

	podAppNameMap := make(map[string]AppName)
	for _, item := range podList.Items {
		for _, reference := range item.OwnerReferences {
			switch reference.Kind {
			case KindDaemonSet, KindStatefulSet, KindReplicaSet:
				podAppNameMap[keyFunc(item.Name, item.Namespace)] = AppName{
					Name:      reference.Name,
					Namespace: item.Namespace,
				}
			default:
				continue
			}
			break
		}
		// 这里可能遍历item没有结果，因为有部分是直接部署的Pod，暂时不处理
	}

	s.logger.Println("正在从metrics server获取PodMetricsList")
	podMetricsList := &metrics.PodMetricsList{}
	err = listFunc(PodMetricsListUrl, podMetricsList)
	if err != nil {
		return nil, errors.Wrap(err, "请求PodMetricsList出错")
	}
	s.logger.Printf("获取了%d条PodMetricsList数据\n", len(podMetricsList.Items))

	result := make([]*AppPodMetrics, 0, len(podMetricsList.Items))
	for _, podMetrics := range podMetricsList.Items {
		cpu := float32(0)
		mem := float32(0)
		for _, container := range podMetrics.Containers {
			cpu += float32(container.Usage.Cpu().MilliValue()) / 1000
			mem += float32(container.Usage.Memory().MilliValue()) / 1000
		}
		appName, ok := podAppNameMap[keyFunc(podMetrics.Name, podMetrics.Namespace)]
		if !ok {
			// 没有appName代表是直接部署的Pod，不会保存其监控数据
			continue
		}

		m := &AppPodMetrics{
			AppName:   appName,
			Timestamp: uint64(podMetrics.Timestamp.Unix()),
			Cpu:       cpu,
			Mem:       mem,
		}
		result = append(result, m)
	}

	return result, nil
}
