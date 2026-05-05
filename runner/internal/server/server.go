package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/logger"
)

type Server struct {
	httpSenderServer *httpSenderServer
	heartbeatServer  *heartbeatServer
	taskServer       *taskServer

	msgChannel  chan httpSenderMessage
	taskChannel chan model.ResponseRunnerHeartbeat
}

func NewServer(runnerName, controllerIP string, controllerPort, httpSenderChannelSize int, token string, heartbeatInterval time.Duration, logger *logger.RunnerLogger) *Server {
	msgChannel := make(chan httpSenderMessage, httpSenderChannelSize)
	taskChannel := make(chan model.ResponseRunnerHeartbeat, httpSenderChannelSize)

	return &Server{
		httpSenderServer: newHttpSenderServer(runnerName, controllerIP, controllerPort, httpSenderChannelSize, token, msgChannel, taskChannel, logger),
		heartbeatServer:  newHeartbeatServer(msgChannel, heartbeatInterval, logger),
		taskServer:       newtaskServer(taskChannel, logger),

		msgChannel:  msgChannel,
		taskChannel: taskChannel,
	}
}

func (s *Server) Start() error {
	if err := s.httpSenderServer.Start(); err != nil {
		return err
	}

	if err := s.heartbeatServer.Start(); err != nil {
		_ = s.httpSenderServer.Stop()
		return err
	}

	if err := s.taskServer.Start(); err != nil {
		_ = s.heartbeatServer.Stop()
		_ = s.httpSenderServer.Stop()
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	b := strings.Builder{}

	if err := s.heartbeatServer.Stop(); err != nil {
		fmt.Fprintf(&b, "heartbeatServer: %v\n", err)
	}

	if err := s.taskServer.Stop(); err != nil {
		fmt.Fprintf(&b, "taskServer: %v\n", err)
	}

	if err := s.httpSenderServer.Stop(); err != nil {
		fmt.Fprintf(&b, "httpSenderServer: %v\n", err)
	}

	close(s.taskChannel)
	close(s.msgChannel)

	if b.Len() == 0 {
		return nil
	}

	return fmt.Errorf("%s", b.String())
}
