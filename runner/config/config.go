package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

type Config struct {
	Runner RunnerIE `yaml:"runner" valid:"required"`
	Logger LoggerIE `yaml:"logger" valid:"required"`
}

type RunnerIE struct {
	Name string `yaml:"name" valid:"required"`

	ControllerIP   string `yaml:"controller_ip" valid:"required,ip"`
	ControllerPort int    `yaml:"controller_port" valid:"required,port"`

	// ControllerURL, such as "https://it.example.org", replaces
	// controller_ip and controller_port when set. Use it when the controller
	// is behind an HTTPS reverse proxy.
	ControllerURL string `yaml:"controller_url"`

	TokenPath string `yaml:"token_path" valid:"required"`

	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" valid:"required,gt=0"`

	HttpSenderChannelSize int `yaml:"http_sender_channel_size" valid:"required,gt=0"`

	WorkspacePath string `yaml:"workspace_path" valid:"required"`

	Terminal TerminalIE `yaml:"terminal"`
}

// TerminalIE lets admins open a shell on this runner from the Runner page
// (off by default). The controller must enable it too.
// See design/runner-terminal.md.
type TerminalIE struct {
	Enabled bool `yaml:"enabled"`

	// User is the OS account the shell runs as; required when enabled.
	// Write "root" to run it as root. An account in the docker group can
	// control docker and so is root-equivalent.
	User string `yaml:"user"`
	// Shell defaults to the user's login shell, or /bin/sh.
	Shell string `yaml:"shell"`
	// Nice is the shell's niceness (default 10), so it yields to tests.
	Nice *int `yaml:"nice"`

	MaxSessions int           `yaml:"max_sessions"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
	MaxDuration time.Duration `yaml:"max_duration"`
}

// ControllerBaseURL returns controller_url when it is set, otherwise
// http://controller_ip:controller_port.
func (r RunnerIE) ControllerBaseURL() (string, error) {
	if r.ControllerURL == "" {
		return "http://" + net.JoinHostPort(r.ControllerIP, strconv.Itoa(r.ControllerPort)), nil
	}

	u, err := url.Parse(r.ControllerURL)
	if err != nil {
		return "", fmt.Errorf("controller_url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil ||
		(u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("controller_url %q must look like https://host[:port]", r.ControllerURL)
	}

	return u.Scheme + "://" + u.Host, nil
}

type LoggerIE struct {
	Level string `yaml:"level" valid:"required"`
}
