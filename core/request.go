package core

import (
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Request struct
type Request struct {
	Method   string  `json:"method"`
	Url      string  `json:"url"`
	UrlParam string  `json:"urlParam"`
	ClientIp string  `json:"clientIp"`
	Headers  MapData `json:"headers"`
	Body     MapData `json:"body"`
}

// CallService call the service api
func (g *Garden) CallService(span opentracing.Span, service, action string, request *Request) (int, string, error) {
	s := g.cfg.Routes[service]
	if len(s) == 0 {
		return 404, NotFound, errors.New("service not found")
	}
	route := s[action]
	if len(route.Path) == 0 {
		return 404, NotFound, errors.New("service route not found")
	}

	// just gateway can request out route
	if strings.ToLower(route.Type) == "out" && strings.Compare(g.cfg.Service.ServiceName, "gateway") != 0 {
		return 404, NotFound, errors.New("just gateway can request out type route")
	}
	// gateway can't call rpc route
	if strings.ToLower(route.Type) == "in" && strings.Compare(g.cfg.Service.ServiceName, "gateway") == 0 {
		return 404, NotFound, errors.New("gateway can't call in type route")
	}

	serviceAddr, nodeIndex, err := g.selectServiceHttpAddr(service)
	if err != nil {
		return 404, NotFound, err
	}
	var result string
	var code int
	url := "http://" + serviceAddr + route.Path

	// service limiter
	if route.Limiter != "" {
		second, quantity, err := limiterAnalyze(route.Limiter)
		if err != nil {
			g.Log(DebugLevel, "Limiter", err)
		} else if !limiterInspect(serviceAddr+"/"+service+"/"+action, second, quantity) {
			span.SetTag("break", "service limiter")
			return 403, ServerLimiter, errors.New("server limiter")
		}
	}

	// service fusing
	if route.Fusing != "" {
		second, quantity, err := fusingAnalyze(route.Fusing)
		if err != nil {
			g.Log(DebugLevel, "Fusing", err)
		} else if !fusingInspect(serviceAddr+"/"+service+"/"+action, second, quantity) {
			span.SetTag("break", "service fusing")
			return 403, ServerFusing, errors.New("server fusing")
		}
	}

	// service call retry
	retry, err := retryAnalyze(g.cfg.Service.CallRetry)
	if err != nil {
		g.Log(DebugLevel, "Retry", err)
		retry = []int{0}
	}

	for i, r := range retry {
		sm := serviceOperate{
			operate:     "incWaiting",
			serviceName: service,
			nodeIndex:   nodeIndex,
		}
		g.serviceManager <- sm

		code, result, err = g.requestService(span, url, request)

		sm.operate = "decWaiting"
		g.serviceManager <- sm

		if code == 404 {
			return code, NotFound, err
		}

		if err != nil {
			if i == len(retry)-1 {
				addFusingQuantity(service + "/" + action)
				return code, ServerError, err
			}
			time.Sleep(time.Millisecond * time.Duration(r))
			continue
		}

		break
	}

	return code, result, nil
}

func (g *Garden) requestService(span opentracing.Span, url string, request *Request) (int, string, error) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// encapsulation request body
	var s string
	for k, v := range request.Body {
		s += fmt.Sprintf("%v=%v&", k, v)
	}
	s = strings.Trim(s, "&")
	r, err := http.NewRequest(request.Method, url, strings.NewReader(s))
	if err != nil {
		return 500, "", err
	}

	// New request request
	for k, v := range request.Headers {
		r.Header.Add(k, v.(string))
	}
	// Add the body format header
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Increase calls to the downstream service security validation key
	r.Header.Set("Call-Key", g.cfg.Service.CallKey)

	// add request opentracing span header
	opentracing.GlobalTracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))

	res, err := client.Do(r)
	if err != nil {
		return 500, "", err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return res.StatusCode, "", errors.New("http status " + strconv.Itoa(res.StatusCode))
	}
	body2, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 500, "", err
	}
	return 200, string(body2), nil
}
