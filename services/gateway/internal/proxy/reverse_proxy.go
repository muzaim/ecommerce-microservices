package proxy

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func ReverseProxyHandler(targetHost string) gin.HandlerFunc {
	target, err := url.Parse(targetHost)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(c *gin.Context) {
		c.Request.Host = target.Host
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
