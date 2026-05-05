package server

import "github.com/Alonza0314/it-system/runner/logger"

type execServer struct {
	*logger.RunnerLogger
}

func newExecServer(logger *logger.RunnerLogger) *execServer {
	return &execServer{
		RunnerLogger: logger,
	}
}

func (s *execServer) Start() error {
	return nil
}

func (s *execServer) Stop() error {
	return nil
}
