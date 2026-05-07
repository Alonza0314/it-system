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

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
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

	repoDir := filepath.Join(s.workspacePath, currentTimeStamp)

	if !s.handlePrepeareRepo(task.Id, repoDir, currentTimeStamp) {
		return
	}

	if !s.handleFetchNfPr(task.Id, task.NFPrList, repoDir) {
		return
	}

	if !s.handleMakeNf(task.Id, repoDir) {
		return
	}

	for _, test := range task.Tests {
		if test == cconstant.TESTCASE_PREPARE_FREE5GC || test == cconstant.TESTCASE_FETCH_PRS || test == cconstant.TESTCASE_MAKE_NF || test == cconstant.TESTCASE_CLEANUP {
			continue
		}

		s.TaskLog.Infof("Running test: %s for task ID: %d", test, task.Id)

		s.handleRunTest(task.Id, test, repoDir)
	}

	if !s.handleCleanup(task.Id, repoDir) {
		return
	}

	s.TaskLog.Infof("All tests completed for task ID: %d", task.Id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", true, "All tests completed successfully"))

}

func (s *taskServer) normalizeOutput(output string) string {
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	cleaned := ansiEscape.ReplaceAllString(output, "")
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")

	return cleaned
}

func (s *taskServer) isTestSuccess(output string) bool {
	cleanedOutput := s.normalizeOutput(output)

	return !strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_1) && !strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_2) && !strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_3)
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

func (s *taskServer) handlePrepeareRepo(id uint64, repoDir, currentTimeStamp string) bool {
	output, err := s.prepareRepo(repoDir, currentTimeStamp)
	if err != nil {
		s.TaskLog.Errorf("Failed to prepare repository for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, output))
		return false
	}
	s.TaskLog.Infof("Repository prepared successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_PREPARE_FREE5GC, true, output))

	return true
}

func (s *taskServer) handleFetchNfPr(id uint64, nfPrs []model.NfPr, repoDir string) bool {
	output, err := s.fetchNfPr(nfPrs, repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to fetch NF-PRs for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, output))
		return false
	}
	s.TaskLog.Infof("NF-PRs fetched successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_FETCH_PRS, true, output))

	return true
}

func (s *taskServer) handleMakeNf(id uint64, repoDir string) bool {
	output, err := s.makeNf(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to make NF for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, output))
		return false
	}
	s.TaskLog.Infof("NF made successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_MAKE_NF, true, output))

	return true
}

func (s *taskServer) handleRunTest(id uint64, testName, repoDir string) {
	output, err := s.runTest(testName, repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to run test: %s for task ID: %d, error: %v", testName, id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, testName, false, fmt.Sprintf("Failed to run test: %v\nOutput: %s", err, output)))
		return
	}

	s.TaskLog.Infof("Test: %s completed for task ID: %d", testName, id)
	s.TaskLog.Tracef("Output of test: %s for task ID: %d: %s", testName, id, output)

	cleanedOutput := s.normalizeOutput(output)
	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, testName, s.isTestSuccess(cleanedOutput), output))
}

func (s *taskServer) handleCleanup(id uint64, repoDir string) bool {
	output, err := s.cleanup(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to cleanup after tests for task ID: %d, error: %v", id, err)
		s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_CLEANUP, false, output))

		return false
	}

	s.TaskLog.Infof("Cleanup completed successfully for task ID: %d", id)
	s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_CLEANUP, true, output))

	return true
}

func (s *taskServer) prepareRepo(repoDir, currentTimeStamp string) (string, error) {
	prepareRepoCtx, prepareRepoCancel := context.WithTimeout(context.Background(), constant.CLONE_CMD_TIMEOUT)
	defer prepareRepoCancel()

	output, err := s.cloneRepo(prepareRepoCtx, repoDir, currentTimeStamp)
	if err != nil {
		if prepareRepoCtx.Err() != nil {
			return output, fmt.Errorf("prepare repo timed out: %v", prepareRepoCtx.Err())
		}

		return output, fmt.Errorf("failed to prepare repo: %v", err)
	}

	return output, nil
}

func (s *taskServer) cloneRepo(ctx context.Context, repoDir, currentTimeStamp string) (string, error) {
	if err := os.MkdirAll(s.workspacePath, 0o755); err != nil {
		return "", err
	}

	output, err := s.runCmd(
		ctx,
		s.workspacePath,
		"git",
		"clone",
		"--recursive",
		"--jobs",
		strconv.Itoa(runtime.NumCPU()),
		constant.FREE5GC_REPO_URL,
		currentTimeStamp,
	)
	if err != nil {
		return output, err
	}

	if _, err := os.Stat(repoDir); err != nil {
		return output, err
	}

	return output, nil
}

func (s *taskServer) fetchNfPr(nfPrs []model.NfPr, repoDir string) (string, error) {
	var output string

	for _, nfPr := range nfPrs {
		s.TaskLog.Debugf("Fetching NF-PR for NF: %s, PR: %d", nfPr.NfName, nfPr.PR)

		ctx, cancel := context.WithTimeout(context.Background(), constant.FETCH_CMD_TIMEOUT)
		defer cancel()

		nfDir := filepath.Join(repoDir, "NFs", nfPr.NfName)
		if fetchOutput, err := s.runCmd(
			ctx,
			nfDir,
			"git",
			"fetch",
			"origin",
			fmt.Sprintf("pull/%d/head:pr-%d", nfPr.PR, nfPr.PR),
		); err != nil {
			if ctx.Err() != nil {
				return fetchOutput, fmt.Errorf("fetch NF-PR timed out for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, ctx.Err())
			}
			return fetchOutput, fmt.Errorf("failed to fetch NF-PR for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, err)
		} else {
			output += fetchOutput + "\n"
		}

		if checkoutOutput, err := s.runCmd(
			ctx,
			nfDir,
			"git",
			"checkout",
			fmt.Sprintf("pr-%d", nfPr.PR),
		); err != nil {
			if ctx.Err() != nil {
				return output + checkoutOutput, fmt.Errorf("checkout NF-PR timed out for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, ctx.Err())
			}
			return output + checkoutOutput, fmt.Errorf("failed to checkout NF-PR for NF: %s, PR: %d, error: %v", nfPr.NfName, nfPr.PR, err)
		} else {
			output += checkoutOutput + "\n"
		}
	}
	return output, nil
}

func (s *taskServer) makeNf(repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.TEST_CMD_TIMEOUT)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		repoDir,
		"make",
	)
	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("make NF timed out: %v", ctx.Err())
		}
		return output, fmt.Errorf("failed to make NF: %v", err)
	}

	return output, nil
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
			return output, fmt.Errorf("cleanup timed out, error: %v", ctx.Err())
		}
		return output, fmt.Errorf("failed to cleanup: %v", err)
	}

	if err := os.RemoveAll(s.workspacePath); err != nil {
		s.TaskLog.Warnf("Failed to remove workspace directory: %v", err)
	}

	return output, nil
}
