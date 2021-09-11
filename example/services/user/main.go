package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"goms"
	"log"
	"net/http"
)

var (
	rpcPort  = flag.String("rpc_port", "9010", "Rpc listen port")
	httpPort = flag.String("http_port", "9510", "Http listen port")
)

func main() {
	flag.Parse()
	goms.Init(*rpcPort, *httpPort, "user", "goms")
	log.Fatal(goms.GinServer(*httpPort, route, nil))
}

func route(r *gin.Engine) {
	r.Use(goms.CheckCallSafeMiddleware())
	r.Any("login", func(c *gin.Context) {
		c.JSON(http.StatusOK, goms.Any{
			"code": 0,
			"msg":  "success",
			"data": goms.Any{
				"method":   goms.GetMethod(c),
				"urlParam": goms.GetUrlParam(c),
				"headers":  goms.GetHeaders(c),
				"body":     goms.GetBody(c),
			},
		})
	})
	r.Any("register", func(c *gin.Context) {
		c.JSON(http.StatusOK, goms.Any{
			"code": 0,
			"msg":  "success",
			"data": goms.Any{
				"method":   goms.GetMethod(c),
				"urlParam": goms.GetUrlParam(c),
				"headers":  goms.GetHeaders(c),
				"body":     goms.GetBody(c),
			},
		})
	})
}