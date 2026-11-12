package config

import "time"

type Config struct {
	Backend BackendIE `yaml:"backend" valid:"required"`
	Logger  LoggerIE  `yaml:"logger" valid:"required"`
}

type BackendIE struct {
	Username string `yaml:"username" valid:"required"`
	Password string `yaml:"password" valid:"required"`

	Port int `yaml:"port" valid:"required"`

	// TrustedProxies lists the reverse proxies (IPs or CIDRs, such as
	// "127.0.0.1" for a proxy on the same host) whose X-Forwarded-For header
	// is believed when logging the client address. Empty trusts no proxy.
	TrustedProxies []string `yaml:"trustedProxies"`

	JWT JWTIE `yaml:"jwt" valid:"required"`

	RunnerJWT JWTIE `yaml:"runnerJwt" valid:"required"`

	RunnerCheckTimeInterval time.Duration `yaml:"runnerCheckTimeInterval" valid:"required"`

	MaxHistoryLength int `yaml:"maxHistoryLength" valid:"required"`

	FrontendFilePath string `yaml:"frontendFilePath" valid:"required"`

	DBPath  string `yaml:"dbPath" valid:"required"`
	LogPath string `yaml:"logPath" valid:"required"`

	Discord DiscordIE `yaml:"discord"`

	WriteToDbTimer time.Duration `yaml:"writeToDbTimer" valid:"required"`

	Terminal TerminalIE `yaml:"terminal"`
}

type DiscordIE struct {
	Enabled        bool   `yaml:"enabled"`
	WebhookUrlPath string `yaml:"webhookUrlPath"`
}

// TerminalIE configures the admin web terminal to runners (off by default).
// See design/runner-terminal.md.
type TerminalIE struct {
	Enabled bool `yaml:"enabled"`

	// AllowInsecureHttp lets browsers open a terminal over plain HTTP. When
	// false, a browser must come over HTTPS through a reverse proxy listed in
	// backend.trustedProxies (deb/https.md); the controller does not start
	// the terminal without one.
	AllowInsecureHttp bool `yaml:"allowInsecureHttp"`

	// AllowedUsers limits who may open a terminal; empty means every admin.
	AllowedUsers []string `yaml:"allowedUsers"`
	// AllowedClientCIDRs limits where browsers may connect from, by the
	// client address (X-Forwarded-For counts only from backend.trustedProxies);
	// empty means anywhere.
	AllowedClientCIDRs []string `yaml:"allowedClientCIDRs"`
	// AllowedOrigins are extra browser origins (host patterns, such as
	// "localhost:5173") accepted besides the controller's own host.
	AllowedOrigins []string `yaml:"allowedOrigins"`

	TicketTTL            time.Duration `yaml:"ticketTTL"`
	IdleTimeout          time.Duration `yaml:"idleTimeout"`
	MaxDuration          time.Duration `yaml:"maxDuration"`
	MaxSessionsPerRunner int           `yaml:"maxSessionsPerRunner"`
	MaxSessions          int           `yaml:"maxSessions"`
}

type JWTIE struct {
	Secret    string        `yaml:"secret" valid:"required"`
	ExpiresIn time.Duration `yaml:"expiresIn" valid:"required"`
}

type LoggerIE struct {
	Level string `yaml:"level" valid:"required"`
}
