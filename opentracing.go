package goms

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	zkOt "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	zkHttp "github.com/openzipkin/zipkin-go/reporter/http"
	"net/http"
	"reflect"
)

// InitOpenTracing 初始化opentracing分布式链路追踪组件
func InitOpenTracing(service, addr, address string) error {
	trace, err := initZipkin(service, addr, address)
	if err != nil {
		return err
	}
	opentracing.SetGlobalTracer(trace)
	return nil
}

// 初始化zipkin组件
func initZipkin(service, addr, address string) (opentracing.Tracer, error) {
	reporter := zkHttp.NewReporter(addr)
	endpoint, err := zipkin.NewEndpoint(service, address)
	if err != nil {
		return nil, err
	}
	trace, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return zkOt.Wrap(trace), nil
}

// StartSpanFromHeader 从请求头获取span
// 如果header中没有span，会新建root span，如果有，则会新建child span
func StartSpanFromHeader(header http.Header) opentracing.Span {
	var span opentracing.Span
	wireContext, _ := opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(header))
	span = opentracing.StartSpan(
		"http",
		//ext.RPCServerOption(wireContext),
		opentracing.ChildOf(wireContext),
	)
	return span
}

// RequestTracing http请求链路跟踪
func RequestTracing(ctx interface{}, span opentracing.Span) {
	t := reflect.TypeOf(ctx)
	switch t.String() {
	case "*gin.Context":
		c := ctx.(*gin.Context)
		requestTracingGin(c, span)
		break
	default:
		break
	}
}

// RequestTracing http请求链路跟踪：gin框架支持
func requestTracingGin(c *gin.Context, span opentracing.Span) {
	request := Request{
		GetMethod(c),
		GetUrl(c),
		GetUrlParam(c),
		GetClientIp(c),
		GetHeaders(c),
		GetBody(c)}
	span.SetTag("Request", request)

	c.Set("span", span)
	c.Set("request", &request)
}