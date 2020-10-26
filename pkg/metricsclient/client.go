package metricsclient

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	metrics "k8s.io/metrics/pkg/apis/metrics/v1alpha1"
	"net/http"
)

const DefaultKubeApiServerBaseUrl = "localhost:8001"

type Client interface {
	QueryAllPodMetrics() (*metrics.PodMetricsList, error)

	QueryNodeMetrics(nodeName string) (*metrics.NodeMetrics, error)
}

type httpClient struct {
	baseUrl string
}

var _ Client = &httpClient{}

func (h *httpClient) QueryAllPodMetrics() (*metrics.PodMetricsList, error) {
	panic("implement me")
}

func (h *httpClient) QueryNodeMetrics(nodeName string) (*metrics.NodeMetrics, error) {
	response, err := http.Get(fmt.Sprintf("%s/apis/metrics.k8s.io/v1beta1/nodes/%s", h.baseUrl, nodeName))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("请求节点监控出现异常，节点名为%s", nodeName))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取出现异常")
	}

	dest := &metrics.NodeMetrics{}
	err = json.Unmarshal(body, dest)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("反序列化异常，值为%s", string(body)))
	}

	return dest, nil
}

func NewHttpMetricsClient(baseUrl string) Client {
	return &httpClient{baseUrl: baseUrl}
}
