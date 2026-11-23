package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cconstant "github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/runner/constant"
)

const cloneTimeStamp = "2026-11-23T23_19_55+08_00"

// fakeClone fakes the two git clones of prepareRepo and counts the attempts.
// Like git, cloning ci-test fails when its directory is there and not empty.
// The free5gc clone of an attempt listed in hang starts writing and then
// hangs, like on a slow connection; one listed in fail fails like a dropped
// connection.
func fakeClone(t *testing.T, hang, fail string) string {
	t.Helper()

	attempt := filepath.Join(t.TempDir(), "attempt")

	return fakeCommands(t, map[string]string{"git": fmt.Sprintf(`[ "$1" = clone ] || exit 0
if [ "$#" -eq 3 ]; then
	if [ -n "$(ls -A "$3" 2>/dev/null)" ]; then
		echo "fatal: destination path '$3' already exists and is not an empty directory."
		exit 128
	fi
	echo $(( $(cat %[1]q 2>/dev/null || echo 0) + 1 )) > %[1]q
	echo "Cloning into '$3'..."
	mkdir -p "$3" && touch "$3/force_kill.sh"
	exit 0
fi
n=$(cat %[1]q)
echo "Cloning into 'free5gc'..."
mkdir -p free5gc/.git
case " %[2]s " in *" $n "*) sleep 30 ;; esac
case " %[3]s " in *" $n "*) echo "fatal: early EOF"; exit 128 ;; esac
mkdir -p free5gc/NFs/amf`, attempt, hang, fail)})
}

// countClones returns how many times ci-test and free5gc were cloned.
func countClones(t *testing.T, calls string) (int, int) {
	t.Helper()

	c := readCalls(t, calls)

	return strings.Count(c, "git clone "+constant.CI_TEST_REPO_URL+" "), strings.Count(c, "git clone --recursive ")
}

func TestPrepareRepoRetries(t *testing.T) {
	for _, tc := range []struct {
		name         string
		hang, fail   string // attempts whose free5gc clone hangs or fails
		wantAttempts int
		wantErr      string // empty: the clone succeeds
		wantLog      []string
		notInLog     []string
	}{
		{
			name:         "SucceedsFirstTime",
			wantAttempts: 1,
			wantLog:      []string{"Cloning into 'free5gc'..."},
			notInLog:     []string{"[runner]"},
		},
		{
			// task 627: 28 seconds in, only 6.4 MB of free5gc was there
			name:         "FirstTimesOutSecondSucceeds",
			hang:         "1",
			wantAttempts: 2,
			wantLog:      []string{"\n[runner] clone attempt 1 of 2 timed out after 1s, retrying\n"},
			notInLog:     []string{"attempt 2 of 2"},
		},
		{
			name:         "FirstFailsSecondSucceeds",
			fail:         "1",
			wantAttempts: 2,
			wantLog:      []string{"fatal: early EOF", "\n[runner] clone attempt 1 of 2 failed: exit status 128, retrying\n"},
			notInLog:     []string{"attempt 2 of 2"},
		},
		{
			name:         "BothTimeOut",
			hang:         "1 2",
			wantAttempts: 2,
			wantErr:      "prepare repo timed out after 2 attempts of 1s: context deadline exceeded",
			wantLog: []string{
				"\n[runner] clone attempt 1 of 2 timed out after 1s, retrying\n",
				"\n[runner] clone attempt 2 of 2 timed out after 1s\n",
			},
		},
		{
			name:         "TimesOutThenFails",
			hang:         "1",
			fail:         "2",
			wantAttempts: 2,
			wantErr:      "failed to prepare repo after 2 attempts: attempt 1 timed out after 1s; attempt 2 failed: exit status 128",
			wantLog:      []string{"fatal: early EOF", "\n[runner] clone attempt 2 of 2 failed: exit status 128\n"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := fakeClone(t, tc.hang, tc.fail)

			s := newTestTaskServer(t)
			s.cloneCmdTimeout = time.Second
			repoDir := filepath.Join(s.workspacePath, cloneTimeStamp)

			start := time.Now()
			output, err := s.prepareRepo(context.Background(), 1, repoDir, cloneTimeStamp)
			if elapsed := time.Since(start); elapsed > 15*time.Second {
				t.Fatalf("Expected each attempt to stop at its timeout, prepare ran for %v", elapsed)
			}

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Expected the clone to succeed, got %v\noutput:\n%s", err, output)
				}
				// the second attempt starts from an empty directory, or the
				// fake git fails like git does
				if _, err := os.Stat(filepath.Join(repoDir, constant.FREE5GC_REPO, "NFs", "amf")); err != nil {
					t.Errorf("Expected a free5gc checkout, stat: %v", err)
				}
			} else if err == nil || err.Error() != tc.wantErr {
				t.Errorf("Expected error %q, got %v", tc.wantErr, err)
			}

			ciTest, free5gc := countClones(t, calls)
			if ciTest != tc.wantAttempts || free5gc != tc.wantAttempts {
				t.Errorf("Expected %d attempts, ci-test was cloned %d times and free5gc %d times", tc.wantAttempts, ciTest, free5gc)
			}
			for _, want := range tc.wantLog {
				if !strings.Contains(output, want) {
					t.Errorf("Expected the output to contain %q, got:\n%s", want, output)
				}
			}
			for _, unwanted := range tc.notInLog {
				if strings.Contains(output, unwanted) {
					t.Errorf("Expected the output not to contain %q, got:\n%s", unwanted, output)
				}
			}
		})
	}
}

// A clone that fails on every attempt is reported as a failed prepare step,
// with what every attempt printed.
func TestHandlePrepareRepoReportsFailedClone(t *testing.T) {
	calls := fakeClone(t, "1 2", "")

	s := newTestTaskServer(t)
	s.cloneCmdTimeout = time.Second

	if s.handlePrepeareRepo(context.Background(), 628, filepath.Join(s.workspacePath, cloneTimeStamp), cloneTimeStamp, false, nil) {
		t.Fatalf("Expected the prepare step to fail")
	}

	outputs := takeTestOutputs(s)
	if len(outputs) != 1 {
		t.Fatalf("Expected one report, got %d", len(outputs))
	}
	out := outputs[0]
	if out.Id != 628 || out.TestName != cconstant.TESTCASE_PREPARE_FREE5GC || out.Status != cconstant.TASK_STATUS_FAILED || out.Success {
		t.Errorf("Expected a failed %s report for task 628, got %+v", cconstant.TESTCASE_PREPARE_FREE5GC, out)
	}
	if out.EndFlag != nil && *out.EndFlag {
		t.Errorf("The prepare report must not end the task: the cleanup ends it")
	}
	if strings.Count(out.Log, "Cloning into 'free5gc'...") != 2 || !strings.Contains(out.Log, "[runner] clone attempt 2 of 2 timed out after 1s") {
		t.Errorf("Expected the output of both attempts, got:\n%s", out.Log)
	}
	if ciTest, _ := countClones(t, calls); ciTest != 2 {
		t.Errorf("Expected 2 attempts, got %d", ciTest)
	}
}

// A task canceled on the controller during a clone is not cloned again and
// not reported: handleTask ends it through finishIfCanceled.
func TestPrepareRepoDoesNotRetryCanceledTask(t *testing.T) {
	calls := fakeClone(t, "1 2", "")

	s := newTestTaskServer(t)
	s.cloneCmdTimeout = time.Minute

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(300*time.Millisecond, cancel)

	start := time.Now()
	output, err := s.prepareRepo(ctx, 7, filepath.Join(s.workspacePath, cloneTimeStamp), cloneTimeStamp)
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Expected the cancel to stop the clone, it ran for %v", elapsed)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected a canceled error, got %v", err)
	}
	if strings.Contains(output, "[runner]") {
		t.Errorf("Expected no attempt note for a canceled clone, got:\n%s", output)
	}
	if ciTest, _ := countClones(t, calls); ciTest != 1 {
		t.Errorf("Expected no second attempt after the cancel, ci-test was cloned %d times", ciTest)
	}

	ctx, cancel = context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	if s.handlePrepeareRepo(ctx, 7, filepath.Join(s.workspacePath, "canceled"), "canceled", false, nil) {
		t.Errorf("Expected the canceled prepare step to stop the task")
	}
	if len(s.msgChannel) != 0 {
		t.Errorf("A canceled prepare must not be reported, got %+v", (<-s.msgChannel).RequestTestOutput)
	}
}
