package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/logger"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"

	"github.com/gin-gonic/gin"
)

func TestSetTrustedProxies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.NewBackendLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true)

	cases := []struct {
		name    string
		proxies []string
		remote  string
		want    string
	}{
		{"no proxy trusted: header ignored", nil, "203.0.113.7:4000", "203.0.113.7"},
		{"no proxy trusted: even from localhost", nil, "127.0.0.1:4000", "127.0.0.1"},
		{"trusted proxy: header used", []string{"127.0.0.1"}, "127.0.0.1:4000", "198.51.100.9"},
		{"untrusted peer: header ignored", []string{"127.0.0.1"}, "203.0.113.7:4000", "203.0.113.7"},
		{"trusted CIDR", []string{"10.8.0.0/16"}, "10.8.0.3:4000", "198.51.100.9"},
		{"invalid entry falls back to none", []string{"not-an-ip"}, "127.0.0.1:4000", "127.0.0.1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			router := gin.New()
			setTrustedProxies(router, c.proxies, log)

			var got string
			router.GET("/ip", func(ctx *gin.Context) { got = ctx.ClientIP() })

			req := httptest.NewRequest(http.MethodGet, "/ip", nil)
			req.RemoteAddr = c.remote
			req.Header.Set("X-Forwarded-For", "198.51.100.9")
			router.ServeHTTP(httptest.NewRecorder(), req)

			if got != c.want {
				t.Errorf("ClientIP() = %q, want %q", got, c.want)
			}
		})
	}
}
