package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
	"github.com/Alonza0314/it-system/runner/logger"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

// writeScript creates an executable test script in dir, standing in for the
// ci-test-*.sh scripts.
func writeScript(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func newTestLogger() *logger.RunnerLogger {
	return logger.NewRunnerLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true)
}

func newTestTaskServer(t *testing.T) *taskServer {
	t.Helper()

	s := newtaskServer(t.TempDir(), make(chan httpSenderMessage, 16), nil, make(chan uint64, 1), newTestLogger())
	s.testCmdTimeout = 200 * time.Millisecond

	return s
}

func TestHandleRunTestReportsTimeout(t *testing.T) {
	repoDir := t.TempDir()
	writeScript(t, repoDir, "slow.sh", "sleep 30")

	s := newTestTaskServer(t)

	start := time.Now()
	status := s.handleRunTest(context.Background(), 1, "TestSlow", "slow.sh", repoDir)
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Expected the test to be killed at the timeout, it ran for %v", elapsed)
	}

	if status != cconstant.TASK_STATUS_TIMEOUT {
		t.Errorf("Expected status %q, got %q", cconstant.TASK_STATUS_TIMEOUT, status)
	}

	msg := <-s.msgChannel
	if msg.RequestTestOutput == nil || msg.RequestTestOutput.Status != cconstant.TASK_STATUS_TIMEOUT {
		t.Errorf("Expected a %q test output for the controller, got %+v", cconstant.TASK_STATUS_TIMEOUT, msg.RequestTestOutput)
	}
}

func TestHandleRunTestReportsFailure(t *testing.T) {
	repoDir := t.TempDir()
	writeScript(t, repoDir, "fail.sh", "echo '--- FAIL: TestBroken'; exit 1")

	s := newTestTaskServer(t)

	if status := s.handleRunTest(context.Background(), 1, "TestBroken", "fail.sh", repoDir); status != cconstant.TASK_STATUS_FAILED {
		t.Errorf("Expected status %q, got %q", cconstant.TASK_STATUS_FAILED, status)
	}
}

func TestCancelStopsRunningTest(t *testing.T) {
	repoDir := t.TempDir()
	writeScript(t, repoDir, "slow.sh", "sleep 30")

	s := newTestTaskServer(t)
	s.testCmdTimeout = time.Minute

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.setCurrentTask(7, cancel)

	go func() {
		time.Sleep(200 * time.Millisecond)
		s.cancelTask(6) // another task: must be ignored
		time.Sleep(200 * time.Millisecond)
		s.cancelTask(7)
	}()

	start := time.Now()
	status := s.handleRunTest(ctx, 7, "TestSlow", "slow.sh", repoDir)
	elapsed := time.Since(start)

	if status != cconstant.TASK_STATUS_CANCELED {
		t.Errorf("Expected status %q, got %q", cconstant.TASK_STATUS_CANCELED, status)
	}
	if elapsed < 300*time.Millisecond {
		t.Errorf("A cancel for another task stopped the test after %v", elapsed)
	}
	if elapsed > 10*time.Second {
		t.Errorf("Expected the test to be killed by the cancel, it ran for %v", elapsed)
	}
	if len(s.msgChannel) != 0 {
		t.Errorf("A canceled test must not be reported as a test result, got %+v", (<-s.msgChannel).RequestTestOutput)
	}
}

func TestFinishIfCanceled(t *testing.T) {
	s := newTestTaskServer(t)
	// No repository: finishIfCanceled must not touch docker on the machine running the test
	repoDir := filepath.Join(t.TempDir(), "missing")

	if s.finishIfCanceled(context.Background(), 7, repoDir) {
		t.Fatalf("A task that was not canceled must keep running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !s.finishIfCanceled(ctx, 7, repoDir) {
		t.Fatalf("Expected a canceled task to be finished")
	}

	msg := <-s.msgChannel
	out := msg.RequestTestOutput
	if out == nil || out.EndFlag == nil || !*out.EndFlag || out.Id != 7 || out.Status != cconstant.TASK_STATUS_CANCELED {
		t.Errorf("Expected an end-of-task report with status %q, got %+v", cconstant.TASK_STATUS_CANCELED, out)
	}
}

func TestSendTestOutputPassesCancelToTaskServer(t *testing.T) {
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/run/runner/test-output" {
			w.Header().Set(cconstant.TASK_CANCELED_HEADER, "7")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer controller.Close()

	cancelChannel := make(chan uint64, 1)
	s := newHttpSenderServer("runner-1", controller.URL, 1, "token", nil, nil, cancelChannel, newTestLogger())
	s.setStatusRunning()

	end := true
	s.sendTestOutput(&model.RequestTestOutput{EndFlag: &end, Id: 7, Status: cconstant.TASK_STATUS_CANCELED})

	select {
	case id := <-cancelChannel:
		if id != 7 {
			t.Errorf("Expected a cancel for task 7, got %d", id)
		}
	default:
		t.Errorf("Expected the cancel header to reach the task server")
	}
	if s.status != constant.STATUS_IDLE {
		t.Errorf("Expected the runner to be idle after the end of the task, got %q", s.status)
	}
}
