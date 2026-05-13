package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
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

func (s *taskServer) buildRequestTestOutput(endFlag bool, testId uint64, testname string, success bool, status string, log string) *model.RequestTestOutput {
	return &model.RequestTestOutput{
		EndFlag:  &endFlag,
		Id:       testId,
		TestName: testname,
		Success:  success,
		Status:   status,
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

	if !s.handlePrepeareRepo(task.Id, repoDir, currentTimeStamp, slices.ContainsFunc(task.NFPrList, func(nfPr model.NfPr) bool { return nfPr.NfName == cconstant.FREE5GC }), task.NFPrList) {
		return
	}

	if !s.handleFetchNfPr(task.Id, task.NFPrList, repoDir) {
		return
	}

	if !s.handleMakeNf(task.Id, repoDir) {
		return
	}

	retryTests := make([]string, 0)
	for _, test := range task.Tests {
		if test == cconstant.TESTCASE_PREPARE_FREE5GC || test == cconstant.TESTCASE_FETCH_PRS || test == cconstant.TESTCASE_MAKE_NF || test == cconstant.TESTCASE_CLEANUP || test == cconstant.FREE5GC {
			continue
		}

		s.TaskLog.Infof("Running test: %s for task ID: %d", test, task.Id)

		status := s.handleRunTest(task.Id, test, repoDir)
		switch status {
		case cconstant.TASK_STATUS_FAILED:
			s.TaskLog.Warnf("Test: %s failed with status: %s for task ID: %d", test, status, task.Id)
			retryTests = append(retryTests, test)
		case cconstant.TASK_STATUS_TIMEOUT:
			s.TaskLog.Warnf("Test: %s timed out for task ID: %d", test, task.Id)
			if !s.handleSilentCleanup(task.Id, test, repoDir) {
				return
			}
			retryTests = append(retryTests, test)
		default: // success case, do nothing
			s.TaskLog.Infof("Test: %s succeeded for task ID: %d", test, task.Id)
		}
	}

	for _, test := range retryTests {
		s.TaskLog.Infof("Retrying test once: %s for task ID: %d", test, task.Id)

		status := s.handleRunTest(task.Id, test, repoDir)
		if status == cconstant.TASK_STATUS_TIMEOUT {
			if !s.handleSilentCleanup(task.Id, test, repoDir) {
				return
			}
		}
	}

	if !s.handleCleanup(task.Id, repoDir) {
		return
	}

	s.TaskLog.Infof("All tests completed for task ID: %d", task.Id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, task.Id, "", true, cconstant.TASK_STATUS_SUCCESS, "All tests completed successfully"))

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
	cmdWithCtx := exec.Command(cmd, args...)
	cmdWithCtx.Dir = dir
	s.configureCommand(cmdWithCtx)

	var output bytes.Buffer
	cmdWithCtx.Stdout, cmdWithCtx.Stderr = &output, &output

	if err := cmdWithCtx.Start(); err != nil {
		return output.String(), fmt.Errorf("command failed to start: %s %v: %w, output: %s", cmd, args, err, output.String())
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmdWithCtx.Wait()
	}()

	select {
	case err := <-waitCh:
		if err != nil {
			return output.String(), fmt.Errorf("command failed: %s %v: %w, output: %s", cmd, args, err, output.String())
		}
	case <-ctx.Done():
		if err := s.terminateCommand(cmdWithCtx); err != nil {
			s.TaskLog.Warnf("Failed to terminate command after timeout: %s %v, error: %v", cmd, args, err)
		}
		<-waitCh

		return output.String(), ctx.Err()
	}

	return output.String(), nil
}

func (s *taskServer) handlePrepeareRepo(id uint64, repoDir, currentTimeStamp string, fetchFree5gcPr bool, nfPrs []model.NfPr) bool {
	output, err := s.prepareRepo(repoDir, currentTimeStamp)
	if err != nil {
		s.TaskLog.Errorf("Failed to prepare repository for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("Repository prepared successfully for task ID: %d", id)

	if fetchFree5gcPr {
		s.TaskLog.Infof("Fetching free5GC PR for task ID: %d", id)

		prNum := 0
		for _, nfPr := range nfPrs {
			if nfPr.NfName == cconstant.FREE5GC {
				prNum = nfPr.PR
				break
			}
		}
		if prNum == 0 {
			s.TaskLog.Warnf("No PR number found for free5GC in task ID: %d, skipping fetch", id)
			s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_PREPARE_FREE5GC, true, cconstant.TASK_STATUS_SUCCESS, output+"\nNo PR number for free5GC, skipping fetch"))

			return true
		}

		ctx, cancel := context.WithTimeout(context.Background(), constant.FETCH_CMD_TIMEOUT)
		defer cancel()

		if fetchOutput, err := s.runCmd(
			ctx,
			repoDir,
			"git",
			"fetch",
			"origin",
			fmt.Sprintf("pull/%d/head:pr-%d", prNum, prNum),
		); err != nil {
			if ctx.Err() != nil {
				s.TaskLog.Errorf("Fetch free5GC PR timed out for task ID: %d, error: %v", id, ctx.Err())
				s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_PREPARE_FREE5GC, false, cconstant.TASK_STATUS_TIMEOUT, output+"\n"+fetchOutput))

				return false
			}
			s.TaskLog.Errorf("Failed to fetch free5GC PR for task ID: %d, error: %v", id, err)
			s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_PREPARE_FREE5GC, false, cconstant.TASK_STATUS_FAILED, output+"\n"+fetchOutput))

			return false
		} else {
			output += "\n" + fetchOutput
		}

		if checkoutOutput, err := s.runCmd(
			ctx,
			repoDir,
			"git",
			"checkout",
			fmt.Sprintf("pr-%d", prNum),
		); err != nil {
			if ctx.Err() != nil {
				s.TaskLog.Errorf("Checkout free5GC PR timed out for task ID: %d, error: %v", id, ctx.Err())
				s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_PREPARE_FREE5GC, false, cconstant.TASK_STATUS_TIMEOUT, output+"\n"+checkoutOutput))

				return false
			}
			s.TaskLog.Errorf("Failed to checkout free5GC PR for task ID: %d, error: %v", id, err)
			s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_PREPARE_FREE5GC, false, cconstant.TASK_STATUS_FAILED, output+"\n"+checkoutOutput))

			return false
		} else {
			output += "\n" + checkoutOutput
		}

		s.TaskLog.Infof("free5GC PR fetched and checked out successfully for task ID: %d", id)
	}

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_PREPARE_FREE5GC, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleFetchNfPr(id uint64, nfPrs []model.NfPr, repoDir string) bool {
	output, err := s.fetchNfPr(nfPrs, repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to fetch NF-PRs for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("NF-PRs fetched successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_FETCH_PRS, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleMakeNf(id uint64, repoDir string) bool {
	output, err := s.makeNf(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to make NF for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("NF made successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_MAKE_NF, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleRunTest(id uint64, testName, repoDir string) string {
	output, err := s.runTest(testName, repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to run test: %s for task ID: %d, error: %v", testName, id, err)

		status := cconstant.TASK_STATUS_FAILED
		if errors.Is(err, context.DeadlineExceeded) {
			status = cconstant.TASK_STATUS_TIMEOUT
		}

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, testName, false, status, output))

		return status
	}

	s.TaskLog.Infof("Test: %s completed for task ID: %d", testName, id)
	s.TaskLog.Tracef("Output of test: %s for task ID: %d: %s", testName, id, output)

	cleanedOutput := s.normalizeOutput(output)
	if s.isTestSuccess(cleanedOutput) {
		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, testName, true, cconstant.TASK_STATUS_SUCCESS, output))

		return cconstant.TASK_STATUS_SUCCESS
	}

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, testName, false, cconstant.TASK_STATUS_FAILED, output))

	return cconstant.TASK_STATUS_FAILED
}

func (s *taskServer) handleCleanup(id uint64, repoDir string) bool {
	output, err := s.forceKill(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to cleanup after tests for task ID: %d, error: %v", id, err)
		s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_CLEANUP, false, cconstant.TASK_STATUS_FAILED, output))

		return false
	}
	s.removeWorkspace()

	s.TaskLog.Infof("Cleanup completed successfully for task ID: %d", id)
	s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_CLEANUP, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleSilentCleanup(id uint64, testName, repoDir string) bool {
	output, err := s.forceKill(repoDir)
	if err == nil {
		s.TaskLog.Infof("Silent cleanup after timeout completed successfully for task ID: %d, test: %s", id, testName)
		s.TaskLog.Tracef("Output of silent cleanup for task ID: %d, test: %s: %s", id, testName, output)

		return true
	}

	s.TaskLog.Errorf("Silent cleanup after timeout failed for task ID: %d, test: %s, error: %v", id, testName, err)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_CLEANUP, false, cconstant.TASK_STATUS_FAILED, output))

	return false
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
		if nfPr.NfName == cconstant.FREE5GC {
			continue
		}

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
	ctx, cancel := context.WithTimeout(context.Background(), constant.MAKE_CMD_TIMEOUT)
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

func (s *taskServer) forceKill(repoDir string) (string, error) {
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

	return output, nil
}

func (s *taskServer) removeWorkspace() {
	if err := os.RemoveAll(s.workspacePath); err != nil {
		s.TaskLog.Warnf("Failed to remove workspace directory: %v", err)
	}
}

func (s *taskServer) configureCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func (s *taskServer) terminateCommand(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
