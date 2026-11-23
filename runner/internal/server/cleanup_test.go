package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
	"github.com/Alonza0314/it-system/runner/constant"
)

// fakeCommands puts scripts named after commands first in PATH, so a test
// never runs the docker, git or make of the machine running it. Each call is
// appended to the returned file as "<command> <args>".
func fakeCommands(t *testing.T, scripts map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	for name, body := range scripts {
		writeScript(t, dir, name, fmt.Sprintf("echo %q \"$*\" >> %q\n%s", name, calls, body))
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return calls
}

func readCalls(t *testing.T, calls string) string {
	t.Helper()

	b, err := os.ReadFile(calls)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	return string(b)
}

// fakeTaskCommands fakes what handleTask runs. Cloning ci-test creates a
// checkout with the given scripts (file name → body), cloning free5gc creates
// it with an amf submodule directory, "git fetch" runs gitFetch, and other git
// commands succeed and print nothing.
func fakeTaskCommands(t *testing.T, ciTest map[string]string, gitFetch, makeBody string) string {
	t.Helper()

	checkout := t.TempDir()
	for name, body := range ciTest {
		writeScript(t, checkout, name, body)
	}
	if gitFetch == "" {
		gitFetch = "true"
	}

	return fakeCommands(t, map[string]string{
		"git": fmt.Sprintf(`case "$1" in
clone)
	if [ "$#" -eq 3 ]; then cp -R %q "$3"; else mkdir -p %s/NFs/amf; fi
	;;
fetch)
	%s
	;;
esac`, checkout, constant.FREE5GC_REPO, gitFetch),
		"make":   makeBody,
		"docker": "echo 'Total reclaimed space: 9.874GB'",
	})
}

func takeTestOutputs(s *taskServer) []*model.RequestTestOutput {
	var outputs []*model.RequestTestOutput
	for len(s.msgChannel) > 0 {
		outputs = append(outputs, (<-s.msgChannel).RequestTestOutput)
	}

	return outputs
}

type step struct{ name, status string }

// checkSteps checks the steps a task reported, in order, and that only the
// last report ends the task.
func checkSteps(t *testing.T, outputs []*model.RequestTestOutput, want []step) {
	t.Helper()

	got := make([]step, len(outputs))
	for i, out := range outputs {
		got[i] = step{out.TestName, out.Status}
		if end := out.EndFlag != nil && *out.EndFlag; end != (i == len(outputs)-1) {
			t.Errorf("Report %d (%s) has endFlag=%v, only the last report must end the task", i, out.TestName, end)
		}
	}

	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Expected steps %v, got %v", want, got)
	}
}

func checkWorkspaceRemoved(t *testing.T, s *taskServer) {
	t.Helper()

	if _, err := os.Stat(s.workspacePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Expected the workspace to be removed, stat: %v", err)
	}
}

func TestHandleCleanup(t *testing.T) {
	const prune = "docker system prune -a -f --volumes"

	for _, tc := range []struct {
		name       string
		forceKill  string // force_kill.sh; empty: no ci-test checkout
		docker     string
		wantStatus string
		wantLog    []string
		wantPrune  bool
	}{
		{
			name:       "Success",
			forceKill:  "echo 'Total reclaimed space: 0B'",
			docker:     "echo 'Total reclaimed space: 9.874GB'",
			wantStatus: cconstant.TASK_STATUS_SUCCESS,
			wantLog:    []string{"Total reclaimed space: 0B", "Total reclaimed space: 9.874GB"},
			wantPrune:  true,
		},
		{
			// task 563: prune is killed while it removes the build cache
			name:       "PruneTimesOut",
			forceKill:  "echo 'Total reclaimed space: 0B'",
			docker:     "echo 'Deleted Images:'; sleep 30",
			wantStatus: cconstant.TASK_STATUS_FAILED,
			wantLog:    []string{"Deleted Images:", "[runner] docker system prune timed out after 1s"},
			wantPrune:  true,
		},
		{
			name:       "PruneFails",
			forceKill:  "true",
			docker:     "echo 'Error response from daemon: a prune operation is already running'; exit 1",
			wantStatus: cconstant.TASK_STATUS_FAILED,
			wantLog:    []string{"a prune operation is already running", "[runner] docker system prune failed: exit status 1"},
			wantPrune:  true,
		},
		{
			name:       "ForceKillTimesOutPruneStillRuns",
			forceKill:  "sleep 30",
			docker:     "echo 'Total reclaimed space: 9.874GB'",
			wantStatus: cconstant.TASK_STATUS_FAILED,
			wantLog:    []string{"[runner] force_kill.sh timed out after 1s", "Total reclaimed space: 9.874GB"},
			wantPrune:  true,
		},
		{
			name:       "ForceKillFailsPruneStillRuns",
			forceKill:  "echo 'Error response from daemon: removal in progress'; exit 3",
			docker:     "echo 'Total reclaimed space: 9.874GB'",
			wantStatus: cconstant.TASK_STATUS_FAILED,
			wantLog:    []string{"removal in progress", "[runner] force_kill.sh failed: exit status 3", "Total reclaimed space: 9.874GB"},
			wantPrune:  true,
		},
		{
			name:       "NoCheckout",
			docker:     "echo 'Total reclaimed space: 9.874GB'",
			wantStatus: cconstant.TASK_STATUS_SUCCESS,
			wantLog:    []string{"No ci-test checkout"},
			wantPrune:  false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := fakeCommands(t, map[string]string{"docker": tc.docker})

			s := newTestTaskServer(t)
			s.forceKillCmdTimeout = time.Second
			s.cleanImagesCmdTimeout = time.Second

			// a task's checkout is inside the workspace
			repoDir := filepath.Join(s.workspacePath, "2026-11-18T01_51_00+08_00")
			if tc.forceKill != "" {
				if err := os.MkdirAll(repoDir, 0o755); err != nil {
					t.Fatal(err)
				}
				writeScript(t, repoDir, "force_kill.sh", tc.forceKill)
			}

			start := time.Now()
			s.handleCleanup(563, repoDir)
			if elapsed := time.Since(start); elapsed > 10*time.Second {
				t.Fatalf("Expected the cleanup to stop at its timeouts, it ran for %v", elapsed)
			}

			outputs := takeTestOutputs(s)
			checkSteps(t, outputs, []step{{cconstant.TESTCASE_CLEANUP, tc.wantStatus}})
			if len(outputs) == 1 {
				out := outputs[0]
				if out.Id != 563 || out.Success != (tc.wantStatus == cconstant.TASK_STATUS_SUCCESS) {
					t.Errorf("Unexpected cleanup report: %+v", out)
				}
				for _, want := range tc.wantLog {
					if !strings.Contains(out.Log, want) {
						t.Errorf("Expected the cleanup log to contain %q, got:\n%s", want, out.Log)
					}
				}
			}

			if pruned := strings.Contains(readCalls(t, calls), prune); pruned != tc.wantPrune {
				t.Errorf("Expected %q to run: %v, ran: %v", prune, tc.wantPrune, pruned)
			}
			checkWorkspaceRemoved(t, s)
		})
	}
}

func TestHandleTaskEndsWithCleanup(t *testing.T) {
	calls := fakeTaskCommands(t, map[string]string{
		"force_kill.sh": "echo 'Total reclaimed space: 0B'",
		"ci-test-it.sh": "echo ok",
	}, "", "echo built")

	s := newTestTaskServer(t)
	s.testCmdTimeout = 10 * time.Second

	s.handleTask(model.ResponseRunnerHeartbeat{Id: 1, Tests: []model.Test{{Name: "TestRegistration", Script: "ci-test-it.sh"}}})

	checkSteps(t, takeTestOutputs(s), []step{
		{cconstant.TESTCASE_PREPARE_FREE5GC, cconstant.TASK_STATUS_SUCCESS},
		{cconstant.TESTCASE_FETCH_PRS, cconstant.TASK_STATUS_SUCCESS},
		{cconstant.TESTCASE_MAKE_NF, cconstant.TASK_STATUS_SUCCESS},
		{"TestRegistration", cconstant.TASK_STATUS_SUCCESS},
		{cconstant.TESTCASE_CLEANUP, cconstant.TASK_STATUS_SUCCESS},
	})
	if !strings.Contains(readCalls(t, calls), "docker system prune") {
		t.Errorf("Expected the cleanup to prune docker")
	}
	checkWorkspaceRemoved(t, s)
}

// A test timed out and force_kill.sh cannot stop its containers: the task
// stops running tests and still gets the whole cleanup.
func TestHandleTaskCleansUpWhenTimedOutTestCannotBeStopped(t *testing.T) {
	calls := fakeTaskCommands(t, map[string]string{
		"force_kill.sh": "echo 'Error response from daemon: removal in progress'; exit 1",
		"ci-test-it.sh": `[ "$2" = TestSlow ] && sleep 30; echo ok`,
	}, "", "echo built")

	s := newTestTaskServer(t)
	s.testCmdTimeout = time.Second

	s.handleTask(model.ResponseRunnerHeartbeat{Id: 1, Tests: []model.Test{
		{Name: "TestSlow", Script: "ci-test-it.sh"},
		{Name: "TestRegistration", Script: "ci-test-it.sh"},
	}})

	checkSteps(t, takeTestOutputs(s), []step{
		{cconstant.TESTCASE_PREPARE_FREE5GC, cconstant.TASK_STATUS_SUCCESS},
		{cconstant.TESTCASE_FETCH_PRS, cconstant.TASK_STATUS_SUCCESS},
		{cconstant.TESTCASE_MAKE_NF, cconstant.TASK_STATUS_SUCCESS},
		{"TestSlow", cconstant.TASK_STATUS_TIMEOUT},
		{cconstant.TESTCASE_CLEANUP, cconstant.TASK_STATUS_FAILED},
	})
	if !strings.Contains(readCalls(t, calls), "docker system prune") {
		t.Errorf("Expected the cleanup to prune docker")
	}
	checkWorkspaceRemoved(t, s)
}

// A prepare, fetch or make step that fails is reported under its own name, and
// the task still ends with its cleanup (11-10: two make failures in a row were
// followed by "parent snapshot ... does not exist" in the next task).
func TestHandleTaskCleansUpAfterFailedStep(t *testing.T) {
	ciTest := map[string]string{
		"force_kill.sh": "echo 'Total reclaimed space: 0B'",
		"ci-test-it.sh": "echo ok",
	}
	prepared := []step{{cconstant.TESTCASE_PREPARE_FREE5GC, cconstant.TASK_STATUS_SUCCESS}}
	fetched := append(prepared, step{cconstant.TESTCASE_FETCH_PRS, cconstant.TASK_STATUS_SUCCESS})

	for _, tc := range []struct {
		name      string
		fake      func(t *testing.T) string
		wantSteps []step
		wantPrune bool
	}{
		{
			name: "PrepareFails",
			fake: func(t *testing.T) string {
				return fakeCommands(t, map[string]string{
					"git":    "echo 'fatal: could not create work tree dir: No space left on device'; exit 128",
					"docker": "echo 'Total reclaimed space: 9.874GB'",
				})
			},
			wantSteps: []step{
				{cconstant.TESTCASE_PREPARE_FREE5GC, cconstant.TASK_STATUS_FAILED},
				{cconstant.TESTCASE_CLEANUP, cconstant.TASK_STATUS_SUCCESS},
			},
			// ci-test was not cloned: nothing ran on docker
			wantPrune: false,
		},
		{
			name: "FetchFails",
			fake: func(t *testing.T) string {
				return fakeTaskCommands(t, ciTest, "echo \"fatal: couldn't find remote ref pull/238/head\"; exit 128", "echo built")
			},
			wantSteps: append(prepared,
				step{cconstant.TESTCASE_FETCH_PRS, cconstant.TASK_STATUS_FAILED},
				step{cconstant.TESTCASE_CLEANUP, cconstant.TASK_STATUS_SUCCESS},
			),
			wantPrune: true,
		},
		{
			name: "MakeFails",
			fake: func(t *testing.T) string {
				return fakeTaskCommands(t, ciTest, "", "echo 'failed to prepare extraction snapshot'; exit 2")
			},
			wantSteps: append(fetched,
				step{cconstant.TESTCASE_MAKE_NF, cconstant.TASK_STATUS_FAILED},
				step{cconstant.TESTCASE_CLEANUP, cconstant.TASK_STATUS_SUCCESS},
			),
			wantPrune: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := tc.fake(t)

			s := newTestTaskServer(t)
			s.handleTask(model.ResponseRunnerHeartbeat{
				Id:       1,
				Tests:    []model.Test{{Name: "TestRegistration", Script: "ci-test-it.sh"}},
				NFPrList: []model.NfPr{{NfName: "amf", PR: 238}},
			})

			outputs := takeTestOutputs(s)
			checkSteps(t, outputs, tc.wantSteps)
			for _, out := range outputs {
				if out.Status == cconstant.TASK_STATUS_FAILED && out.Log == "" {
					t.Errorf("Expected the failed step %s to report its output", out.TestName)
				}
			}
			if pruned := strings.Contains(readCalls(t, calls), "docker system prune"); pruned != tc.wantPrune {
				t.Errorf("Expected docker to be pruned: %v, pruned: %v", tc.wantPrune, pruned)
			}
			checkWorkspaceRemoved(t, s)
		})
	}
}
