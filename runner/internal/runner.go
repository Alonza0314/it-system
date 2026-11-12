package internal

import (
	"fmt"
	"time"

	"github.com/Alonza0314/it-system/runner/config"
	"github.com/Alonza0314/it-system/runner/internal/server"
	"github.com/Alonza0314/it-system/runner/logger"
)

type runner struct {
	name string

	controllerURL string

	server.Server

	*logger.RunnerLogger
}

func NewRunner(config *config.Config, token string, logger *logger.RunnerLogger) *runner {
	controllerURL, err := config.Runner.ControllerBaseURL()
	if err != nil {
		logger.CfgLog.Errorf("Invalid controller address: %v", err)
		return nil
	}

	return &runner{
		name: config.Runner.Name,

		controllerURL: controllerURL,

		Server: *server.NewServer(config.Runner.Name, controllerURL, config.Runner.HttpSenderChannelSize, token, config.Runner.HeartbeatInterval, config.Runner.WorkspacePath, logger),

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
}

func (r *runner) Stop() {
	fmt.Println()
	r.RunLog.Infoln("Stopping runner...")

	if err := r.Server.Stop(); err != nil {
		r.RunLog.Errorf("Failed to stop runner server: %v", err)
	} else {
		r.RunLog.Infoln("Runner server stopped successfully")
	}
}
