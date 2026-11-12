package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/config"
	"github.com/Alonza0314/it-system/controller/backend/internal/terminal"
	"github.com/Alonza0314/it-system/controller/backend/logger"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"

	"github.com/gin-gonic/gin"
)

func TestTerminalOptions(t *testing.T) {
	good := func() *config.Config {
		cfg := &config.Config{}
		cfg.Backend.Password = "a-real-password"
		cfg.Backend.JWT.Secret = strings.Repeat("a", 32)
		cfg.Backend.RunnerJWT.Secret = strings.Repeat("b", 32)
		cfg.Backend.Terminal = config.TerminalIE{
			Enabled:            true,
			AllowInsecureHttp:  true,
			AllowedClientCIDRs: []string{"10.8.0.0/16"},
			AllowedOrigins:     []string{"localhost:5173"},
		}
		return cfg
	}

	opts, err := terminalOptions(good())
	if err != nil {
		t.Fatalf("good config refused: %v", err)
	}
	if opts.RequireHTTPS {
		t.Error("allowInsecureHttp: true still requires HTTPS")
	}

	behindProxy := good()
	behindProxy.Backend.Terminal.AllowInsecureHttp = false
	behindProxy.Backend.TrustedProxies = []string{"127.0.0.1", "::1", "10.8.0.0/16"}
	opts, err = terminalOptions(behindProxy)
	if err != nil {
		t.Fatalf("config behind an HTTPS proxy refused: %v", err)
	}
	if !opts.RequireHTTPS || len(opts.TrustedProxies) != 3 || opts.TrustedProxies[0].String() != "127.0.0.1/32" {
		t.Errorf("RequireHTTPS = %v, TrustedProxies = %v", opts.RequireHTTPS, opts.TrustedProxies)
	}

	cases := []struct {
		name   string
		change func(*config.Config)
		want   string
	}{
		{"no HTTPS proxy and plain HTTP not allowed", func(c *config.Config) { c.Backend.Terminal.AllowInsecureHttp = false }, "needs HTTPS"},
		{"proxy list trusts everyone", func(c *config.Config) { c.Backend.TrustedProxies = []string{"0.0.0.0/0"} }, "trusts every address"},
		{"bad proxy entry", func(c *config.Config) { c.Backend.TrustedProxies = []string{"not-an-ip"} }, "trustedProxies"},
		{"default jwt secret", func(c *config.Config) { c.Backend.JWT.Secret = "free5gc-it-system" }, "jwt.secret is the default"},
		{"default runner secret", func(c *config.Config) { c.Backend.RunnerJWT.Secret = "free5gc-it-runner" }, "runnerJwt.secret is the default"},
		{"short jwt secret", func(c *config.Config) { c.Backend.JWT.Secret = "free5gc-it-system " }, "at least 32"},
		{"short runner secret", func(c *config.Config) { c.Backend.RunnerJWT.Secret = "secret" }, "at least 32"},
		{"same secrets", func(c *config.Config) { c.Backend.RunnerJWT.Secret = c.Backend.JWT.Secret }, "must differ"},
		{"default password", func(c *config.Config) { c.Backend.Password = "0000" }, "password is the default"},
		{"long ticket lifetime", func(c *config.Config) { c.Backend.Terminal.TicketTTL = 2 * time.Minute }, "ticketTTL"},
		{"wildcard origin", func(c *config.Config) { c.Backend.Terminal.AllowedOrigins = []string{"*"} }, "any site"},
		{"wildcard origin with port", func(c *config.Config) { c.Backend.Terminal.AllowedOrigins = []string{"*:*"} }, "any site"},
		{"bad CIDR", func(c *config.Config) { c.Backend.Terminal.AllowedClientCIDRs = []string{"10.8.0.0"} }, "allowedClientCIDRs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := good()
			c.change(cfg)
			_, err := terminalOptions(cfg)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("err = %v, want one containing %q", err, c.want)
			}
		})
	}
}

// The handlers build the terminal client from gin's ClientIP and
// Hub.ClientFor; behind the proxy both must come from the proxy's headers,
// and from nobody else's.
func TestTerminalClientBehindProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.NewBackendLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true)

	cfg := &config.Config{}
	cfg.Backend.TrustedProxies = []string{"127.0.0.1"}
	proxies, err := parseTrustedProxies(cfg.Backend.TrustedProxies)
	if err != nil {
		t.Fatal(err)
	}
	hub := terminal.NewHub(terminal.Options{RequireHTTPS: true, TrustedProxies: proxies}, nil, nil)
	defer hub.Close()

	router := gin.New()
	setTrustedProxies(router, cfg.Backend.TrustedProxies, log)
	var got terminal.Client
	router.GET("/ws", func(c *gin.Context) { got = hub.ClientFor(c.Request, c.ClientIP()) })

	cases := []struct {
		name   string
		remote string
		want   terminal.Client
	}{
		{"through the proxy", "127.0.0.1:4000", terminal.Client{IP: "198.51.100.9", HTTPS: true}},
		{"straight to :8888 with forged headers", "203.0.113.7:4000", terminal.Client{IP: "203.0.113.7", HTTPS: false}},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/ws", nil)
		req.RemoteAddr = c.remote
		req.Header.Set("X-Forwarded-For", "198.51.100.9")
		req.Header.Set("X-Forwarded-Proto", "https")
		router.ServeHTTP(httptest.NewRecorder(), req)

		if got != c.want {
			t.Errorf("%s: client = %+v, want %+v", c.name, got, c.want)
		}
	}
}
