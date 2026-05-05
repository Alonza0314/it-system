package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"
)

type taskServer struct {
	workspacePath string

	msgChannel  chan httpSenderMessage
	taskChannel chan model.ResponseRunnerHeartbeat

	taskCtx    context.Context
	taskCancel context.CancelFunc

	*logger.RunnerLogger
}

func newtaskServer(workspace string, msgChannel chan httpSenderMessage, taskChannel chan model.ResponseRunnerHeartbeat, logger *logger.RunnerLogger) *taskServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &taskServer{
		workspacePath: workspace,

		msgChannel:  msgChannel,
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

func (s *taskServer) buildRequestTestOutput(endFlag bool, testId uint64, testname string, success bool, log string) *model.RequestTestOutput {
	return &model.RequestTestOutput{
		EndFlag:  &endFlag,
		Id:       testId,
		TestName: testname,
		Success:  success,
		Log:      log,
	}
}

func (s *taskServer) handleTask(task model.ResponseRunnerHeartbeat) {
	s.TaskLog.Infof("Received task from controller, task ID: %d", task.Id)
	s.TaskLog.Tracef("Task tests: %v", task.Tests)
	s.TaskLog.Tracef("Task NF-PR list: %v", task.NFPrList)

	if err := s.prepareRepo(); err != nil {
		s.TaskLog.Errorf("Failed to prepare repository for task ID: %d, error: %v", task.Id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", false, fmt.Sprintf("Failed to prepare repository: %v", err)))
		return
	}
	s.TaskLog.Infof("Repository prepared successfully for task ID: %d", task.Id)
}

func (s *taskServer) prepareRepo() error {
	prepareRepoCtx, prepareRepoCancel := context.WithTimeout(context.Background(), constant.CLONE_CMD_TIMEOUT)
	defer prepareRepoCancel()
	if err := s.cloneRepo(prepareRepoCtx); err != nil {
		if prepareRepoCtx.Err() != nil {
			return fmt.Errorf("prepare repo timed out: %v", prepareRepoCtx.Err())
		}

		return fmt.Errorf("failed to prepare repo: %v", err)
	}

	return nil
}

func (s *taskServer) cloneRepo(ctx context.Context) error {
	if err := os.MkdirAll(s.workspacePath, 0o755); err != nil {
		return err
	}

	repoDir := filepath.Join(s.workspacePath, constant.FREE5GC_REPO)

	if _, err := s.runCmd(
		ctx,
		s.workspacePath,
		"git",
		"clone",
		"--recursive",
		"--jobs",
		strconv.Itoa(runtime.NumCPU()),
		constant.FREE5GC_REPO_URL,
	); err != nil {
		return err
	}

	if _, err := os.Stat(repoDir); err != nil {
		return err
	}

	return nil
}

func (s *taskServer) runCmd(ctx context.Context, dir, cmd string, args ...string) (string, error) {
	cmdWithCtx := exec.CommandContext(ctx, cmd, args...)
	cmdWithCtx.Dir = dir

	output, err := cmdWithCtx.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %s %v: %w, output: %s", cmd, args, err, string(output))
	}

	return string(output), nil
}
