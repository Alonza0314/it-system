package server

import (
	"context"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"
)

type heartbeatServer struct {
	serverChan chan httpSenderMessage

	heartbeatInterval time.Duration

	heartbeatCtx    context.Context
	heartbeatCancel context.CancelFunc

	ticker *time.Ticker

	*logger.RunnerLogger
}

func newHeartbeatServer(serverChan chan httpSenderMessage, heartbeatInterval time.Duration, logger *logger.RunnerLogger) *heartbeatServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &heartbeatServer{
		serverChan: serverChan,

		heartbeatInterval: heartbeatInterval,

		heartbeatCtx:    ctx,
		heartbeatCancel: cancel,

		ticker: time.NewTicker(heartbeatInterval),

		RunnerLogger: logger,
	}
}

func (s *heartbeatServer) Start() error {
	go func() {
		for {
			select {
			case <-s.heartbeatCtx.Done():
				return
			case <-s.ticker.C:
				go s.sendHeartbeat()
			}
		}
	}()

	return nil
}

func (s *heartbeatServer) Stop() error {
	s.heartbeatCancel()
	s.ticker.Stop()

	return nil
}

func (s *heartbeatServer) sendHeartbeat() {
	t := true
	request := &model.RequestRunnerHeartbeat{
		Idle:        &t,
		OnGoingTask: 0,
	}

	s.serverChan <- newHttpSenderMessage(constant.MSG_TYPE_HEARTBEAT, request, nil)
}
