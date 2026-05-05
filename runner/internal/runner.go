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

	controllerIP   string
	controllerPort int

	server.Server

	*logger.RunnerLogger
}

func NewRunner(config *config.Config, token string, logger *logger.RunnerLogger) *runner {
	return &runner{
		name: config.Runner.Name,

		controllerIP:   config.Runner.ControllerIP,
		controllerPort: config.Runner.ControllerPort,

		Server: *server.NewServer(config.Runner.Name, config.Runner.ControllerIP, config.Runner.ControllerPort, config.Runner.HttpSenderChannelSize, token, config.Runner.HeartbeatInterval, config.Runner.WorkspacePath, logger),

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

	r.RunLog.Infof("Runner started with controller IP: %s, controller Port: %d", r.controllerIP, r.controllerPort)
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
