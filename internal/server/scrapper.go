package server

import (
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"net/http"
	"time"
)

// 用于从metrics server获取数据并保存到数据库的goroutine主函数
func (s *serverImpl) scrapper(ctx context.Context) {
	s.logger.Println("监控数据获取线程启动")
	tickCh := time.Tick(s.config.ScrapeInterval)
	for {
		select {
		case <-tickCh:
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
