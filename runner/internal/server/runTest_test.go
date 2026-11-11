package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
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

func newTestTaskServer() *taskServer {
	s := newtaskServer("", make(chan httpSenderMessage, 16), nil, logger.NewRunnerLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true))
	s.testCmdTimeout = 200 * time.Millisecond

	return s
}

func TestHandleRunTestReportsTimeout(t *testing.T) {
	repoDir := t.TempDir()
	writeScript(t, repoDir, "slow.sh", "sleep 30")

	s := newTestTaskServer()

	start := time.Now()
	status := s.handleRunTest(1, "TestSlow", "slow.sh", repoDir)
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

	s := newTestTaskServer()

	if status := s.handleRunTest(1, "TestBroken", "fail.sh", repoDir); status != cconstant.TASK_STATUS_FAILED {
		t.Errorf("Expected status %q, got %q", cconstant.TASK_STATUS_FAILED, status)
	}
}
