package server

import (
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const minDuration = 1

type ServerContext struct {
	// Kubernetes Metrics Server的
	MetricsServerPodListUrl string
	// 给每个应用保留的数据的时间长度，单位是天。最小值是1。
	MetricDuration int
	// 本服务器监听端口
	Port int
	// 从metrics server获取数据的周期。至少为15s。
	ScrapeInterval time.Duration
}

type Server interface {
	Start() error
	Shutdown()
}

func NewServer(ctx *ServerContext) (Server, error) {
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	return &server{
		ctx:    ctx,
		dao:    nil,
		stop:   false,
		logger: log.New(os.Stdout, "workload server: ", log.LstdFlags|log.Lmsgprefix),
	}, nil
}

type server struct {
	ctx    *ServerContext
	dao    Dao
	stop   bool
	logger *log.Logger
}

func (s *server) Shutdown() {
	panic("implement me")
}

func checkContext(ctx *ServerContext) error {
	if ctx.Port > 65535 || ctx.Port < 1024 {
		return fmt.Errorf("端口号应该在1024到65535之间，现在为%d", ctx.Port)
	}

	if ctx.MetricDuration < minDuration {
		return fmt.Errorf("MetricDuration应该至少为%d，现在为%d", minDuration, ctx.MetricDuration)
	}

	_, err := url.Parse(ctx.MetricsServerPodListUrl)
	if err != nil {
		return errors.Wrap(err, "解析k8s metrics server URL异常")
	}

	if ctx.ScrapeInterval < time.Second*15 {
		return fmt.Errorf("时间不能短于15s，现在是%fs", ctx.ScrapeInterval.Seconds())
	}

	return nil
}

func (s *server) Start() error {

	go s.scrapper()

	return nil
}

// 用于从metrics server获取数据并保存到数据库的goroutine主函数
func (s *server) scrapper() {
	for !s.stop {
		select {
		case <-time.After(s.ctx.ScrapeInterval):
			metrics, err := scrapePodMetrics(s.ctx.MetricsServerPodListUrl)
			if err != nil {
				panic(err)
			}
			err = s.dao.SaveAllAppPodMetrics(metrics)
			if err != nil {
				panic(err)
			}
		default:
			panic("不应该运行到这里")
		}
	}
}

func scrapePodMetrics(url string) ([]*AppPodMetrics, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("请求容器监控数据出错，网址为%s", url))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取容器监控数据出错")
	}
	print(body)
	return nil, nil
}
