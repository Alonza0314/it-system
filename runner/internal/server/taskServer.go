package server

import (
	"context"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/logger"
)

type taskServer struct {
	taskChannel chan model.ResponseRunnerHeartbeat

	taskCtx    context.Context
	taskCancel context.CancelFunc

	*logger.RunnerLogger
}

func newtaskServer(taskChannel chan model.ResponseRunnerHeartbeat, logger *logger.RunnerLogger) *taskServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &taskServer{
		taskChannel: taskChannel,

		taskCtx:    ctx,
		taskCancel: cancel,

		RunnerLogger: logger,
	}
}

func (s *taskServer) Start() error {
	go func() {
		for {
			select {
			case <-s.taskCtx.Done():
				return
			case task := <-s.taskChannel:
				s.handleTask(task)
			}
		}
	}()

	return nil
}

func (s *taskServer) Stop() error {
	s.taskCancel()

	return nil
}

func (s *taskServer) handleTask(task model.ResponseRunnerHeartbeat) {
	s.TaskLog.Infof("Received task from controller, task ID: %d", task.Id)
	s.TaskLog.Tracef("Task tests: %v", task.Tests)
	s.TaskLog.Tracef("Task NF-PR list: %v", task.NFPrList)
}
