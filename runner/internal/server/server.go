package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/Alonza0314/it-system/runner/logger"
)

type Server struct {
	httpSenderServer *httpSenderServer
	heartbeatServer  *heartbeatServer
	execServer       *execServer

	serverChannel chan httpSenderMessage
}

func NewServer(runnerName, controllerIP string, controllerPort, httpSenderChannelSize int, token string, heartbeatInterval time.Duration, logger *logger.RunnerLogger) *Server {
	serverChannel := make(chan httpSenderMessage, httpSenderChannelSize)

	return &Server{
		httpSenderServer: newHttpSenderServer(runnerName, controllerIP, controllerPort, httpSenderChannelSize, token, serverChannel, logger),
		heartbeatServer:  newHeartbeatServer(serverChannel, heartbeatInterval, logger),
		execServer:       newExecServer(logger),

		serverChannel: serverChannel,
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

	if err := s.execServer.Start(); err != nil {
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

	if err := s.execServer.Stop(); err != nil {
		fmt.Fprintf(&b, "execServer: %v\n", err)
	}

	if err := s.httpSenderServer.Stop(); err != nil {
		fmt.Fprintf(&b, "httpSenderServer: %v\n", err)
	}

	close(s.serverChannel)

	if b.Len() == 0 {
		return nil
	}

	return fmt.Errorf("%s", b.String())
}
