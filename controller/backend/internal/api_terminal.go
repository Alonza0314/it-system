package internal

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/config"
	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/internal/processor"
	"github.com/Alonza0314/it-system/controller/backend/internal/terminal"
	"github.com/Alonza0314/it-system/controller/backend/logger"
	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
	gojwt "github.com/golang-jwt/jwt/v5"
)

const (
	terminalDisabledMessage = "Terminal is disabled on this controller"

	// secrets shorter than this can be brute-forced offline from any token
	// the controller hands out (tenants get one by logging in)
	minTerminalSecretLength = 32
	maxTerminalTicketTTL    = time.Minute
)

// newTerminalHub returns nil when the terminal is off or its configuration
// is not safe to run with; the reason is logged.
func newTerminalHub(cfg *config.Config, p *processor.Processor, log *logger.BackendLogger) *terminal.Hub {
	t := cfg.Backend.Terminal
	if !t.Enabled {
		return nil
	}

	opts, err := terminalOptions(cfg)
	if err != nil {
		log.TermLog.Errorf("Terminal is disabled: %v", err)
		return nil
	}

	if opts.RequireHTTPS {
		log.TermLog.Infof("Terminal is enabled for browsers that come over HTTPS through %v", cfg.Backend.TrustedProxies)
	} else {
		log.TermLog.Warnf("Terminal is enabled over plain HTTP (terminal.allowInsecureHttp): JWTs and everything typed in a terminal can be read on the way. Use it only over a VPN.")
	}
	if len(opts.AllowedUsers) == 0 {
		log.TermLog.Warnf("Terminal: every admin account may open a terminal (terminal.allowedUsers is empty)")
	}
	if len(opts.AllowedClientNets) == 0 {
		log.TermLog.Warnf("Terminal: browsers may connect from any address (terminal.allowedClientCIDRs is empty)")
	}

	return terminal.NewHub(opts, p, log.TermLog)
}

// terminalOptions refuses configurations that would hand a runner shell to
// anyone who reads the public repository.
func terminalOptions(cfg *config.Config) (terminal.Options, error) {
	t := cfg.Backend.Terminal

	proxies, err := parseTrustedProxies(cfg.Backend.TrustedProxies)
	if err != nil {
		return terminal.Options{}, err
	}
	if !t.AllowInsecureHttp && len(proxies) == 0 {
		return terminal.Options{}, fmt.Errorf("the terminal needs HTTPS: put the controller behind an HTTPS reverse proxy and list it in backend.trustedProxies (see deb/https.md), or set terminal.allowInsecureHttp: true to allow plain HTTP")
	}
	if cfg.Backend.JWT.Secret == constant.DEFAULT_JWT_SECRET {
		return terminal.Options{}, fmt.Errorf("backend.jwt.secret is the default from the repository; anyone could sign an admin token")
	}
	if cfg.Backend.RunnerJWT.Secret == constant.DEFAULT_RUNNER_JWT_SECRET {
		return terminal.Options{}, fmt.Errorf("backend.runnerJwt.secret is the default from the repository; anyone could impersonate a runner")
	}
	if len(cfg.Backend.JWT.Secret) < minTerminalSecretLength || len(cfg.Backend.RunnerJWT.Secret) < minTerminalSecretLength {
		return terminal.Options{}, fmt.Errorf("backend.jwt.secret and backend.runnerJwt.secret must be at least %d characters (for example `openssl rand -hex 32`)", minTerminalSecretLength)
	}
	if cfg.Backend.JWT.Secret == cfg.Backend.RunnerJWT.Secret {
		return terminal.Options{}, fmt.Errorf("backend.jwt.secret and backend.runnerJwt.secret must differ; otherwise a tenant whose role is \"runner\" passes as a runner")
	}
	if cfg.Backend.Password == constant.DEFAULT_BACKEND_PASSWORD {
		return terminal.Options{}, fmt.Errorf("backend.password is the default from the repository")
	}
	if t.TicketTTL > maxTerminalTicketTTL {
		return terminal.Options{}, fmt.Errorf("terminal.ticketTTL must be at most %s", maxTerminalTicketTTL)
	}
	for _, origin := range t.AllowedOrigins {
		if strings.Trim(origin, "*?.:/") == "" {
			return terminal.Options{}, fmt.Errorf("terminal.allowedOrigins %q would accept any site", origin)
		}
	}

	nets := make([]*net.IPNet, 0, len(t.AllowedClientCIDRs))
	for _, cidr := range t.AllowedClientCIDRs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return terminal.Options{}, fmt.Errorf("terminal.allowedClientCIDRs: %v", err)
		}
		nets = append(nets, n)
	}

	return terminal.Options{
		TicketTTL:            t.TicketTTL,
		IdleTimeout:          t.IdleTimeout,
		MaxDuration:          t.MaxDuration,
		MaxSessionsPerRunner: t.MaxSessionsPerRunner,
		MaxSessions:          t.MaxSessions,
		AllowedUsers:         t.AllowedUsers,
		AllowedClientNets:    nets,
		RequireHTTPS:         !t.AllowInsecureHttp,
		TrustedProxies:       proxies,
		OriginPatterns:       t.AllowedOrigins,
	}, nil
}

// parseTrustedProxies reads backend.trustedProxies (addresses or CIDRs, as
// gin accepts them). The terminal believes X-Forwarded-Proto from these.
func parseTrustedProxies(entries []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(entries))
	for _, entry := range entries {
		var (
			prefix netip.Prefix
			err    error
		)
		if strings.Contains(entry, "/") {
			prefix, err = netip.ParsePrefix(entry)
			prefix = prefix.Masked()
		} else {
			var addr netip.Addr
			addr, err = netip.ParseAddr(entry)
			prefix = netip.PrefixFrom(addr.Unmap(), addr.Unmap().BitLen())
		}
		if err != nil {
			return nil, fmt.Errorf("backend.trustedProxies: %v", err)
		}
		if prefix.Bits() == 0 {
			return nil, fmt.Errorf("backend.trustedProxies %q trusts every address; anyone could claim to come over HTTPS or from another address", entry)
		}
		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}

func (b *backend) getAdminRunnerTerminalRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Create Terminal Ticket",
			Method:      http.MethodPost,
			Pattern:     "/ticket",
			HandlerFunc: b.handleTerminalTicket,
		},
	}
}

func (b *backend) getTerminalRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Terminal WebSocket",
			Method:      http.MethodGet,
			Pattern:     "/ws",
			HandlerFunc: b.handleTerminalBrowser,
		},
	}
}

func (b *backend) getRunRunnerTerminalRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Runner Terminal Control",
			Method:      http.MethodGet,
			Pattern:     "/control",
			HandlerFunc: b.handleTerminalControl,
		},
		{
			Name:        "Runner Terminal Session",
			Method:      http.MethodGet,
			Pattern:     "/session",
			HandlerFunc: b.handleTerminalRunnerSession,
		},
	}
}

func (b *backend) handleTerminalTicket(c *gin.Context) {
	if b.terminal == nil {
		c.JSON(http.StatusServiceUnavailable, model.ResponseTerminalTicket{Message: terminalDisabledMessage})
		return
	}

	var req model.RequestTerminalTicket
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ResponseTerminalTicket{Message: "Invalid request body: " + err.Error()})
		return
	}

	user, expires := claimsUser(c)
	client := b.terminal.ClientFor(c.Request, c.ClientIP())
	ticket, ttl, errDetail := b.terminal.IssueTicket(user, req.Runner, client, req.Cols, req.Rows, expires)
	if errDetail != nil {
		b.TermLog.Warnf("Terminal ticket refused: user=%q runner=%q remote=%s: %s", user, req.Runner, client.IP, errDetail.Message)
		c.JSON(errDetail.Status, model.ResponseTerminalTicket{Message: errDetail.Message})
		return
	}

	c.JSON(http.StatusOK, model.ResponseTerminalTicket{
		Message:   "Terminal ticket created",
		Ticket:    ticket,
		ExpiresIn: int(ttl / time.Second),
	})
}

func (b *backend) handleTerminalBrowser(c *gin.Context) {
	if b.terminal == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": terminalDisabledMessage})
		return
	}

	b.terminal.ServeBrowser(c.Writer, c.Request, b.terminal.ClientFor(c.Request, c.ClientIP()))
}

func (b *backend) handleTerminalControl(c *gin.Context) {
	if b.terminal == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": terminalDisabledMessage})
		return
	}

	runner, _ := claimsUser(c)
	b.terminal.ServeControl(c.Writer, c.Request, runner, claimString(c, constant.RUNNER_REGISTRATION_CLAIM_TAG), c.ClientIP())
}

func (b *backend) handleTerminalRunnerSession(c *gin.Context) {
	if b.terminal == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": terminalDisabledMessage})
		return
	}

	runner, _ := claimsUser(c)
	b.terminal.ServeRunnerSession(c.Writer, c.Request, runner, claimString(c, constant.RUNNER_REGISTRATION_CLAIM_TAG), c.Query("id"))
}

func requestClaims(c *gin.Context) gojwt.MapClaims {
	value, ok := c.Get(constant.CLAIMS_CONTEXT_KEY)
	if !ok {
		return nil
	}
	claims, _ := value.(gojwt.MapClaims)

	return claims
}

// claimString returns a string claim of the JWT that the admin or runner
// middleware validated, or "".
func claimString(c *gin.Context, key string) string {
	value, _ := requestClaims(c)[key].(string)

	return value
}

// claimsUser returns the "user" claim and expiry of the JWT that the admin or
// runner middleware validated.
func claimsUser(c *gin.Context) (string, time.Time) {
	claims := requestClaims(c)

	var expires time.Time
	if claims != nil {
		if exp, err := claims.GetExpirationTime(); err == nil && exp != nil {
			expires = exp.Time
		}
	}

	return claimString(c, "user"), expires
}
