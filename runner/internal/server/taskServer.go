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
	"sync"
	"syscall"
	"time"

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"
)

type taskServer struct {
	workspacePath string

	msgChannel    chan httpSenderMessage
	taskChannel   chan model.ResponseRunnerHeartbeat
	cancelChannel chan uint64

	taskCtx    context.Context
	taskCancel context.CancelFunc

	// the task being run, so a cancel from the controller can stop it
	currentLock   sync.Mutex
	currentId     uint64
	currentCancel context.CancelFunc

	testCmdTimeout time.Duration

	*logger.RunnerLogger
}

func newtaskServer(workspace string, msgChannel chan httpSenderMessage, taskChannel chan model.ResponseRunnerHeartbeat, cancelChannel chan uint64, logger *logger.RunnerLogger) *taskServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &taskServer{
		workspacePath: workspace,

		msgChannel:    msgChannel,
		taskChannel:   taskChannel,
		cancelChannel: cancelChannel,

		taskCtx:    ctx,
		taskCancel: cancel,

		testCmdTimeout: constant.TEST_CMD_TIMEOUT,

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

	go func() {
		for {
			select {
			case <-s.taskCtx.Done():
				return
			case id, ok := <-s.cancelChannel:
				if !ok {
					return
				}
				s.cancelTask(id)
			}
		}
	}()

	return nil
}

func (s *taskServer) setCurrentTask(id uint64, cancel context.CancelFunc) {
	s.currentLock.Lock()
	defer s.currentLock.Unlock()

	s.currentId, s.currentCancel = id, cancel
}

// cancelTask stops the task the controller no longer runs. The command of the
// running step is killed through its context; a cancel for a task that has
// already ended is ignored.
func (s *taskServer) cancelTask(id uint64) {
	s.currentLock.Lock()
	defer s.currentLock.Unlock()

	if s.currentId != id || s.currentCancel == nil {
		s.TaskLog.Debugf("Ignoring cancel for task ID: %d, it is not running", id)
		return
	}

	s.TaskLog.Warnf("Task ID: %d was canceled on the controller, stopping it", id)
	s.currentCancel()
}

// finishIfCanceled ends a canceled task: it removes the containers and images
// the interrupted step left behind and reports the end of the task, so the
// controller marks this runner idle. It returns false if the task was not
// canceled.
func (s *taskServer) finishIfCanceled(ctx context.Context, id uint64, repoDir string) bool {
	if ctx.Err() == nil {
		return false
	}

	if _, err := os.Stat(repoDir); err == nil {
		if output, err := s.forceKill(repoDir); err != nil {
			s.TaskLog.Warnf("Failed to stop containers of canceled task ID: %d, error: %v", id, err)
			s.TaskLog.Tracef("Output of force kill for task ID: %d: %s", id, output)
		}
		if output, err := s.cleanImages(repoDir); err != nil {
			s.TaskLog.Warnf("Failed to clean docker images of canceled task ID: %d, error: %v", id, err)
			s.TaskLog.Tracef("Output of clean images for task ID: %d: %s", id, output)
		}
	}
	s.removeWorkspace()

	s.TaskLog.Infof("Canceled task ID: %d stopped and cleaned up", id)
	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_CANCELED, "Task canceled on the controller"))

	return true
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

	// Not derived from s.taskCtx: stopping the runner keeps its old behavior
	// and does not go through the cancel path.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.setCurrentTask(task.Id, cancel)
	defer s.setCurrentTask(0, nil)

	if !s.handlePrepeareRepo(ctx, task.Id, repoDir, currentTimeStamp, slices.ContainsFunc(task.NFPrList, func(nfPr model.NfPr) bool { return nfPr.NfName == cconstant.FREE5GC }), task.NFPrList) {
		s.finishIfCanceled(ctx, task.Id, repoDir)
		return
	}

	f5gcDir := filepath.Join(repoDir, constant.FREE5GC_REPO)

	if !s.handleFetchNfPr(ctx, task.Id, task.NFPrList, f5gcDir) {
		s.finishIfCanceled(ctx, task.Id, repoDir)
		return
	}

	if !s.handleMakeNf(ctx, task.Id, repoDir) {
		s.finishIfCanceled(ctx, task.Id, repoDir)
		return
	}

	retryTests := make([]model.Test, 0)
	for _, test := range task.Tests {
		if test.Name == cconstant.TESTCASE_PREPARE_FREE5GC || test.Name == cconstant.TESTCASE_FETCH_PRS || test.Name == cconstant.TESTCASE_MAKE_NF || test.Name == cconstant.TESTCASE_CLEANUP || test.Name == cconstant.FREE5GC {
			continue
		}

		if s.finishIfCanceled(ctx, task.Id, repoDir) {
			return
		}

		s.TaskLog.Infof("Running test: %s for task ID: %d", test, task.Id)

		status := s.handleRunTest(ctx, task.Id, test.Name, test.Script, repoDir)
		switch status {
		case cconstant.TASK_STATUS_CANCELED:
			s.finishIfCanceled(ctx, task.Id, repoDir)
			return
		case cconstant.TASK_STATUS_FAILED:
			s.TaskLog.Warnf("Test: %s failed with status: %s for task ID: %d", test, status, task.Id)
			retryTests = append(retryTests, test)
		case cconstant.TASK_STATUS_TIMEOUT:
			s.TaskLog.Warnf("Test: %s timed out for task ID: %d", test, task.Id)
			if !s.handleSilentCleanup(task.Id, test.Name, repoDir) {
				return
			}
			retryTests = append(retryTests, test)
		default: // success case, do nothing
			s.TaskLog.Infof("Test: %s succeeded for task ID: %d", test, task.Id)
		}
	}

	for _, test := range retryTests {
		if s.finishIfCanceled(ctx, task.Id, repoDir) {
			return
		}

		s.TaskLog.Infof("Retrying test once: %s for task ID: %d", test, task.Id)

		status := s.handleRunTest(ctx, task.Id, test.Name, test.Script, repoDir)
		if status == cconstant.TASK_STATUS_CANCELED {
			s.finishIfCanceled(ctx, task.Id, repoDir)
			return
		}
		if status == cconstant.TASK_STATUS_TIMEOUT {
			if !s.handleSilentCleanup(task.Id, test.Name, repoDir) {
				return
			}
		}
	}

	if s.finishIfCanceled(ctx, task.Id, repoDir) {
		return
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

	return !strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_1) &&
		!strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_2) &&
		!strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_3) &&
		!strings.Contains(cleanedOutput, constant.FAIL_MESSAGE_4)
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

// The handle* steps below do not report a failure when ctx is canceled: the
// task was canceled on the controller and handleTask reports its end.
func (s *taskServer) handlePrepeareRepo(ctx context.Context, id uint64, repoDir, currentTimeStamp string, fetchFree5gcPr bool, nfPrs []model.NfPr) bool {
	output, err := s.prepareRepo(ctx, repoDir, currentTimeStamp)
	if err != nil {
		if ctx.Err() != nil {
			return false
		}
		s.TaskLog.Errorf("Failed to prepare repository for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("Repository prepared successfully for task ID: %d", id)

	f5gcDir := filepath.Join(repoDir, constant.FREE5GC_REPO)
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

		taskCtx := ctx
		ctx, cancel := context.WithTimeout(taskCtx, constant.FETCH_CMD_TIMEOUT)
		defer cancel()

		if fetchOutput, err := s.runCmd(
			ctx,
			f5gcDir,
			"git",
			"fetch",
			"origin",
			fmt.Sprintf("pull/%d/head:pr-%d", prNum, prNum),
		); err != nil {
			if taskCtx.Err() != nil {
				return false
			}
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
			f5gcDir,
			"git",
			"checkout",
			fmt.Sprintf("pr-%d", prNum),
		); err != nil {
			if taskCtx.Err() != nil {
				return false
			}
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

func (s *taskServer) handleFetchNfPr(ctx context.Context, id uint64, nfPrs []model.NfPr, f5gcDir string) bool {
	output, err := s.fetchNfPr(ctx, nfPrs, f5gcDir)
	if err != nil {
		if ctx.Err() != nil {
			return false
		}
		s.TaskLog.Errorf("Failed to fetch NF-PRs for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("NF-PRs fetched successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_FETCH_PRS, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleMakeNf(ctx context.Context, id uint64, repoDir string) bool {
	output, err := s.makeNf(ctx, repoDir)
	if err != nil {
		if ctx.Err() != nil {
			return false
		}
		s.TaskLog.Errorf("Failed to make NF for task ID: %d, error: %v", id, err)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, "", false, cconstant.TASK_STATUS_FAILED, output))
		return false
	}
	s.TaskLog.Infof("NF made successfully for task ID: %d", id)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_MAKE_NF, true, cconstant.TASK_STATUS_SUCCESS, output))

	return true
}

func (s *taskServer) handleRunTest(ctx context.Context, id uint64, testName, testScript, repoDir string) string {
	output, err := s.runTest(ctx, testName, testScript, repoDir)
	if err != nil {
		if ctx.Err() != nil {
			return cconstant.TASK_STATUS_CANCELED
		}
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
	output1, err := s.forceKill(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to cleanup after tests for task ID: %d, error: %v", id, err)
		s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output1)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_CLEANUP, false, cconstant.TASK_STATUS_FAILED, output1))

		return false
	}

	output2, err := s.cleanImages(repoDir)
	if err != nil {
		s.TaskLog.Errorf("Failed to clean docker images for task ID: %d, err: %v", id, err)
		s.TaskLog.Tracef("output of clean image for task ID: %d: %s", id, output2)

		s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(true, id, cconstant.TESTCASE_CLEANUP, false, cconstant.TASK_STATUS_FAILED, output1+output2))

		return false
	}
	s.removeWorkspace()

	s.TaskLog.Infof("Cleanup completed successfully for task ID: %d", id)
	s.TaskLog.Tracef("Output of cleanup for task ID: %d: %s", id, output1+output2)

	s.msgChannel <- newHttpSenderMessage(constant.MSG_TYPE_TEST_OUTPUT, nil, s.buildRequestTestOutput(false, id, cconstant.TESTCASE_CLEANUP, true, cconstant.TASK_STATUS_SUCCESS, output1+output2))

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

func (s *taskServer) prepareRepo(ctx context.Context, repoDir, currentTimeStamp string) (string, error) {
	prepareRepoCtx, prepareRepoCancel := context.WithTimeout(ctx, constant.CLONE_CMD_TIMEOUT)
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

	output1, err := s.runCmd(
		ctx,
		s.workspacePath,
		"git",
		"clone",
		constant.CI_TEST_REPO_URL,
		currentTimeStamp,
	)
	if err != nil {
		return output1, err
	}

	output2, err := s.runCmd(
		ctx,
		repoDir,
		"git",
		"clone",
		"--recursive",
		"--jobs",
		strconv.Itoa(runtime.NumCPU()),
		constant.FREE5GC_REPO_URL,
	)
	if err != nil {
		return output1 + "\n" + output2, err
	}

	if _, err := os.Stat(repoDir); err != nil {
		return output1 + "\n" + output2, err
	}

	return output1 + "\n" + output2, nil
}

// nfRepoDir is where an NF's submodule is in the free5gc repository:
// NFs/<nf>, except webconsole, which is at the top level.
func nfRepoDir(f5gcDir, nf string) string {
	if nf == cconstant.WEBCONSOLE {
		return filepath.Join(f5gcDir, nf)
	}

	return filepath.Join(f5gcDir, "NFs", nf)
}

func (s *taskServer) fetchNfPr(taskCtx context.Context, nfPrs []model.NfPr, f5gcDir string) (string, error) {
	var output string

	for _, nfPr := range nfPrs {
		if nfPr.NfName == cconstant.FREE5GC {
			continue
		}

		s.TaskLog.Debugf("Fetching NF-PR for NF: %s, PR: %d", nfPr.NfName, nfPr.PR)

		ctx, cancel := context.WithTimeout(taskCtx, constant.FETCH_CMD_TIMEOUT)
		defer cancel()

		nfDir := nfRepoDir(f5gcDir, nfPr.NfName)
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

	output += s.listCommits(f5gcDir, nfPrs)

	if s.shouldTidyTestModule(nfPrs) {
		tidyOutput, err := s.tidyTestModule(f5gcDir)
		output += "Running go mod tidy in test module\n" + tidyOutput
		if err != nil {
			return output, err
		}
	}

	return output, nil
}

// listCommits reports the commit that ci-test, free5gc and every free5gc
// submodule are checked out at, so the task log shows which NFs are built
// from a PR and which from the default branch.
func (s *taskServer) listCommits(f5gcDir string, nfPrs []model.NfPr) string {
	ctx, cancel := context.WithTimeout(context.Background(), constant.FETCH_CMD_TIMEOUT)
	defer cancel()

	prs := make(map[string]int, len(nfPrs))
	for _, nfPr := range nfPrs {
		prs[nfPr.NfName] = nfPr.PR
	}

	b := strings.Builder{}
	b.WriteString("Commits under test:\n")

	writeLine := func(path, commit string) {
		fmt.Fprintf(&b, "%-12s %s", path, commit)
		if pr, ok := prs[filepath.Base(path)]; ok {
			fmt.Fprintf(&b, " (PR #%d)", pr)
		}
		b.WriteString("\n")
	}

	for _, repo := range []struct{ path, dir string }{
		{constant.CI_TEST_REPO, filepath.Dir(f5gcDir)},
		{constant.FREE5GC_REPO, f5gcDir},
	} {
		commit, err := s.runCmd(ctx, repo.dir, "git", "log", "-1", "--oneline")
		if err != nil {
			s.TaskLog.Warnf("Failed to read commit of %s: %v", repo.path, err)
			writeLine(repo.path, "unknown")
			continue
		}
		writeLine(repo.path, strings.TrimSpace(commit))
	}

	submodules, err := s.runCmd(ctx, f5gcDir, "git", "submodule", "foreach", "--quiet", `echo "$sm_path $(git log -1 --oneline)"`)
	if err != nil {
		s.TaskLog.Warnf("Failed to read commits of free5gc submodules: %v", err)
		return b.String() + submodules + "\n"
	}
	for line := range strings.SplitSeq(strings.TrimSpace(submodules), "\n") {
		if line == "" {
			continue
		}
		path, commit, _ := strings.Cut(line, " ")
		writeLine(path, commit)
	}

	return b.String()
}

func (s *taskServer) shouldTidyTestModule(nfPrs []model.NfPr) bool {
	for _, nfPr := range nfPrs {
		if nfPr.NfName == cconstant.FREE5GC {
			return len(nfPrs) > 1
		}
	}

	return false
}

func (s *taskServer) tidyTestModule(f5gcDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.TEST_CMD_TIMEOUT)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		filepath.Join(f5gcDir, "test"),
		"go",
		"mod",
		"tidy",
	)
	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("go mod tidy timed out in test module: %v", ctx.Err())
		}
		return output, fmt.Errorf("failed to run go mod tidy in test module: %v", err)
	}

	return output, nil
}

func (s *taskServer) makeNf(taskCtx context.Context, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(taskCtx, constant.MAKE_CMD_TIMEOUT)
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

func (s *taskServer) runTest(taskCtx context.Context, testName, testScript, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(taskCtx, s.testCmdTimeout)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		repoDir,
		"./"+testScript,
		"--test",
		testName,
		"--build",
	)
	if err != nil {
		if ctx.Err() != nil {
			// %w keeps context.DeadlineExceeded, which handleRunTest checks to
			// report the test as timed out and clean up its containers
			return output, fmt.Errorf("run test timed out for test: %s, error: %w", testName, ctx.Err())
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

func (s *taskServer) cleanImages(repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.CLEANUP_CMD_TIMEOUT)
	defer cancel()

	output, err := s.runCmd(
		ctx,
		repoDir,
		"docker",
		"system",
		"prune",
		"-a",
		"-f",
		"--volumes",
	)
	if err != nil {
		if ctx.Err() != nil {
			return output, fmt.Errorf("clean images timed out, error: %v", ctx.Err())
		}
		return output, fmt.Errorf("failed to clean images: %v", err)
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
