package client

import (
	"encoding/json"
	"fmt"
	"github.com/packagewjx/workload-classifier/pkg/server"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

const defaultApiHostBaseUrl = "http://workload-classifier.workload-classifier"

func NewApiClient() server.API {
	return &apiClient{}
}

var _ server.API = &apiClient{}

type apiClient struct {
}

func (a *apiClient) QueryAppCharacteristics(appName server.AppName) (*server.AppCharacteristics, error) {
	response, err := http.Get(fmt.Sprintf(defaultApiHostBaseUrl+"/namespaces/%s/appcharacteristics/%s",
		appName.Namespace, appName.Name))
	if err != nil {
		return nil, errors.Wrap(err, "请求时出现异常")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "读取时出现异常")
	}

	dest := &server.AppCharacteristics{}
	err = json.Unmarshal(body, dest)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("解析json异常，json为\n%s", string(body)))
	}

	return dest, nil
}

func (a *apiClient) ReCluster() {
	_, _ = http.Get(defaultApiHostBaseUrl + "/recluster")
}
