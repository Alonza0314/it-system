package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

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

	loc, err := time.LoadLocation("Asia/Taipei")
	if err != nil {
		s.TaskLog.Warnf("Failed to load location for Asia/Taipei: %v", err)
		loc = time.FixedZone("CST", 8*3600)
	}

	currentTimeStamp := strings.ReplaceAll(time.Now().In(loc).Format(time.RFC3339), ":", "_")
	s.TaskLog.Infof("Starting task ID: %d at %s", task.Id, currentTimeStamp)

	currentTimeStamp = "test"

	repoDir := filepath.Join(s.workspacePath, currentTimeStamp)

	if err := s.prepareRepo(repoDir, currentTimeStamp); err != nil {
		s.TaskLog.Errorf("Failed to prepare repository for task ID: %d, error: %v", task.Id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", false, fmt.Sprintf("Failed to prepare repository: %v", err)))
		return
	}
	s.TaskLog.Infof("Repository prepared successfully for task ID: %d", task.Id)

	if err := s.fetchNfPr(task.NFPrList, repoDir); err != nil {
		s.TaskLog.Errorf("Failed to fetch NF-PRs for task ID: %d, error: %v", task.Id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", false, fmt.Sprintf("Failed to fetch NF-PRs: %v", err)))
		return
	}
	s.TaskLog.Infof("NF-PRs fetched successfully for task ID: %d", task.Id)

	if err := s.makeNf(repoDir); err != nil {
		s.TaskLog.Errorf("Failed to make NF for task ID: %d, error: %v", task.Id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", false, fmt.Sprintf("Failed to make NF: %v", err)))
		return
	}
	s.TaskLog.Infof("NF made successfully for task ID: %d", task.Id)

	for _, test := range task.Tests {
		s.TaskLog.Infof("Running test: %s for task ID: %d", test, task.Id)

		output, err := s.runTest(test, repoDir)
		if err != nil {
			s.TaskLog.Errorf("Failed to run test: %s for task ID: %d, error: %v", test, task.Id, err)

			s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, task.Id, test, false, fmt.Sprintf("Failed to run test: %v\nOutput: %s", err, output)))
			continue
		}

		s.TaskLog.Infof("Test: %s completed for task ID: %d", test, task.Id)
		s.TaskLog.Tracef("Output of test: %s for task ID: %d: %s", test, task.Id, output)

		cleanedOutput := s.normalizeOutput(output)
		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, task.Id, test, !(strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_1) || strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_2) || strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_3)), output))
	}
	s.TaskLog.Infof("All tests completed for task ID: %d", task.Id)

	output, err := s.cleanup(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to cleanup after tests for task ID: %d, error: %v", task.Id, err)
		s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", task.Id, output)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", false, output))
		return
	} else {
		s.TaskLog.Infof("Cleanup completed successfully for task ID: %d", task.Id)
		s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", task.Id, output)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", true, "All tests completed"))
	}
}

func (s *taskServer) normalizeOutput(output string) string {
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	cleaned := ansiEscape.ReplaceAllString(output, "")
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	return cleaned
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

func (s *taskServer) prepareRepo(repoDir, currentTimeStamp string) error {
	prepareRepoCtx, prepareRepoCancel := context.WithTimeout(context.Background(), constant.CLONE_CMD_TIMEOUT)
	defer prepareRepoCancel()
	if err := s.cloneRepo(prepareRepoCtx, repoDir, currentTimeStamp); err != nil {
		if prepareRepoCtx.Err() != nil {
			return fmt.Errorf("prepare repo timed out: %v", prepareRepoCtx.Err())
		}

		return fmt.Errorf("failed to prepare repo: %v", err)
	}

	return nil
}

func (s *taskServer) cloneRepo(ctx context.Context, repoDir, currentTimeStamp string) error {
	if err := os.MkdirAll(s.workspacePath, 0o755); err != nil {
		return err
	}

	if _, err := s.runCmd(
		ctx,
		s.workspacePath,
		"git",
		"clone",
		"--recursive",
		"--jobs",
		strconv.Itoa(runtime.NumCPU()),
		constant.FREE5GC_REPO_URL,
		currentTimeStamp,
	); err != nil {
		return err
	}

	if _, err := os.Stat(repoDir); err != nil {
		return err
	}

	return nil
}

func (s *taskServer) fetchNfPr(nfPrs []model.NfPr, repoDir string) error {
	for _, nfPr := range nfPrs {
		s.TaskLog.Debugf("Fetching NF-PR for NF: %s, PR: %d", nfPr.NfName, nfPr.PR)

		ctx, cancel := context.WithTimeout(context.Background(), constant.FETCH_CMD_TIMEOUT)
		defer cancel()

		nfDir := filepath.Join(repoDir, "NFs", nfPr.NfName)
		if _, err := s.runCmd(
			ctx,
			nfDir,
			"git",
			"fetch",
			"origin",
			fmt.Sprintf("pull/%d/head:pr-%d", nfPr.PR, nfPr.PR),
		); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("fetch NF-PR timed out for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, ctx.Err())
			}
			return fmt.Errorf("failed to fetch NF-PR for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, err)
		}

		if _, err := s.runCmd(
			ctx,
			nfDir,
			"git",
			"checkout",
			fmt.Sprintf("pr-%d", nfPr.PR),
		); err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("checkout NF-PR timed out for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, ctx.Err())
			}
			return fmt.Errorf("failed to checkout NF-PR for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, err)
		}
	}
	return nil
}

func (s *taskServer) makeNf(repoDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constant.TEST_CMD_TIMEOUT)
	defer cancel()

	if _, err := s.runCmd(
		ctx,
		repoDir,
		"make",
	); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("make NF timed out: %v", ctx.Err())
		}
		return fmt.Errorf("failed to make NF: %v", err)
	}

	return nil
}

func (s *taskServer) runTest(testName, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.TEST_CMD_TIMEOUT)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		repoDir,
		"./test.sh",
		testName,
	)
	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("run test timed out for test: %s, error: %v", testName, ctx.Err())
		}
		return output, fmt.Errorf("failed to run test: %s, error: %v", testName, err)
	}

	return output, nil
}

func (s *taskServer) cleanup(repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.CLEANUP_CMD_TIMEOUT)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		repoDir,
		"./force_kill.sh",
	)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("cleanup timed out, error: %v", ctx.Err())
		}
		return "", fmt.Errorf("failed to cleanup: %v", err)
	}

	return output, nil
}
