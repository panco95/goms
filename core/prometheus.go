package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//PushGateway upload metrics data
func (g *Garden) PushGateway(job string, data MapData) (string, error) {
	client := &http.Client{
		Timeout: time.Millisecond * time.Duration(5000),
	}

	url := fmt.Sprintf("http://%s/metrics/job/%s/instance/%s", g.cfg.Service.PushGatewayAddress, job, job)
	r, err := http.NewRequest("POST", url, strings.NewReader(GenMetricsData(data)))
	if err != nil {
		return "", err
	}

	res, err := client.Do(r)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body2, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body2), nil
}

// GenMetricsData format
func GenMetricsData(data MapData) string {
	body := ""
	for k, v := range data {
		body += fmt.Sprintf("%s %v\n", k, v)
	}
	return body
}