package internal

import (
	"fmt"
	"time"

	"github.com/Alonza0314/it-system/runner/config"
	"github.com/Alonza0314/it-system/runner/internal/server"
	"github.com/Alonza0314/it-system/runner/internal/terminal"
	"github.com/Alonza0314/it-system/runner/logger"
)

type runner struct {
	name string

	controllerURL string

	server.Server

	terminal *terminal.Agent

	*logger.RunnerLogger
}

func NewRunner(config *config.Config, token string, logger *logger.RunnerLogger) *runner {
	controllerURL, err := config.Runner.ControllerBaseURL()
	if err != nil {
		logger.CfgLog.Errorf("Invalid controller address: %v", err)
		return nil
	}

	var agent *terminal.Agent
	if config.Runner.Terminal.Enabled {
		agent, err = terminal.NewAgent(config.Runner.Name, controllerURL, token, config.Runner.Terminal, logger.TermLog)
		if err != nil {
			logger.CfgLog.Errorf("Invalid terminal configuration: %v", err)
			return nil
		}
	}

	return &runner{
		name: config.Runner.Name,

		controllerURL: controllerURL,

		Server: *server.NewServer(config.Runner.Name, controllerURL, config.Runner.HttpSenderChannelSize, token, config.Runner.HeartbeatInterval, config.Runner.WorkspacePath, logger),

		terminal: agent,

		RunnerLogger: logger,
	}
}

func (r *runner) Start() {
	r.RunLog.Infoln("Starting runner...")

	go func() {
		if err := r.Server.Start(); err != nil {
			r.RunLog.Errorf("Failed to start runner server: %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond)

	r.RunLog.Infof("Runner started with controller: %s", r.controllerURL)

	if r.terminal != nil {
		r.terminal.Start()
		r.TermLog.Warnf("Terminal is enabled: admins of the controller can open a shell on this runner as %q", r.terminal.ShellUser())
	}
}

func (r *runner) Stop() {
	fmt.Println()
	r.RunLog.Infoln("Stopping runner...")

	if r.terminal != nil {
		r.terminal.Stop()
	}

	if err := r.Server.Stop(); err != nil {
		r.RunLog.Errorf("Failed to stop runner server: %v", err)
	} else {
		r.RunLog.Infoln("Runner server stopped successfully")
	}
}
