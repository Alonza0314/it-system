package processor_test

import (
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

// The status of a finished task comes from its tests: a cleanup that failed
// is shown as a failed step, but it does not fail a task whose tests passed.
func TestFinalTaskStatus(t *testing.T) {
	p := newTestProcessor(t)

	if _, errDetail := p.AddTestcases(&model.RequestAddTestcases{Testcases: []model.Testcase{
		{Name: "TestRegistration", Script: "ci-test-it.sh", Label: "IT"},
		{Name: "TestPaging", Script: "ci-test-it.sh", Label: "IT"},
	}}); errDetail != nil {
		t.Fatalf("Failed to add testcases: %s", errDetail.Detail)
	}
	if _, errDetail := p.RegisterRunner(&model.RequestRegisterRunner{Name: "runner-1", IP: "10.0.0.1"}); errDetail != nil {
		t.Fatalf("Failed to register runner: %s", errDetail.Detail)
	}

	type report struct{ name, status string }
	built := []report{
		{constant.TESTCASE_PREPARE_FREE5GC, constant.TASK_STATUS_SUCCESS},
		{constant.TESTCASE_FETCH_PRS, constant.TASK_STATUS_SUCCESS},
		{constant.TESTCASE_MAKE_NF, constant.TASK_STATUS_SUCCESS},
	}
	with := func(reports ...report) []report {
		return append(append([]report{}, built...), reports...)
	}

	for _, tc := range []struct {
		name string
		// the last report ends the task
		reports []report
		want    string
	}{
		{
			name: "TestsPassCleanupFails",
			reports: with(
				report{"TestRegistration", constant.TASK_STATUS_SUCCESS},
				report{"TestPaging", constant.TASK_STATUS_SUCCESS},
				report{constant.TESTCASE_CLEANUP, constant.TASK_STATUS_FAILED},
			),
			want: constant.TASK_STATUS_SUCCESS,
		},
		{
			name: "TestsPassCleanupSucceeds",
			reports: with(
				report{"TestRegistration", constant.TASK_STATUS_SUCCESS},
				report{"TestPaging", constant.TASK_STATUS_SUCCESS},
				report{constant.TESTCASE_CLEANUP, constant.TASK_STATUS_SUCCESS},
			),
			want: constant.TASK_STATUS_SUCCESS,
		},
		{
			name: "TestFailsCleanupFails",
			reports: with(
				report{"TestRegistration", constant.TASK_STATUS_FAILED},
				report{"TestPaging", constant.TASK_STATUS_SUCCESS},
				report{constant.TESTCASE_CLEANUP, constant.TASK_STATUS_FAILED},
			),
			want: constant.TASK_STATUS_FAILED,
		},
		{
			name: "TestTimesOutCleanupFails",
			reports: with(
				report{"TestRegistration", constant.TASK_STATUS_TIMEOUT},
				report{"TestPaging", constant.TASK_STATUS_SUCCESS},
				report{constant.TESTCASE_CLEANUP, constant.TASK_STATUS_FAILED},
			),
			want: constant.TASK_STATUS_TIMEOUT,
		},
		{
			name: "MakeFailsCleanupSucceeds",
			reports: []report{
				{constant.TESTCASE_PREPARE_FREE5GC, constant.TASK_STATUS_SUCCESS},
				{constant.TESTCASE_FETCH_PRS, constant.TASK_STATUS_SUCCESS},
				{constant.TESTCASE_MAKE_NF, constant.TASK_STATUS_FAILED},
				{constant.TESTCASE_CLEANUP, constant.TASK_STATUS_SUCCESS},
			},
			want: constant.TASK_STATUS_FAILED,
		},
		{
			// the cleanup never reported: the end report has no test name
			name: "EndsWithoutCleanup",
			reports: with(
				report{"TestRegistration", constant.TASK_STATUS_SUCCESS},
				report{"TestPaging", constant.TASK_STATUS_SUCCESS},
				report{"", constant.TASK_STATUS_SUCCESS},
			),
			want: constant.TASK_STATUS_FAILED,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := submitTestTask(t, p)

			idle := true
			if _, errDetail := p.RunnerHeartbeat(&model.RequestRunnerHeartbeat{Idle: &idle}, "runner-1"); errDetail != nil {
				t.Fatalf("Heartbeat failed: %s", errDetail.Detail)
			}
			if r := runnerState(t, p, "runner-1"); r.OnGoingTask != id {
				t.Fatalf("Expected runner-1 to run task %d, got %+v", id, r)
			}

			for i, r := range tc.reports {
				if _, errDetail := p.TtestOutput(testOutput(id, r.name, r.status, i == len(tc.reports)-1), "runner-1"); errDetail != nil {
					t.Fatalf("Failed to report %s: %s", r.name, errDetail.Detail)
				}
			}

			task, errDetail := p.GetTask(id)
			if errDetail != nil {
				t.Fatalf("Failed to get task: %s", errDetail.Detail)
			}
			if task.Status != tc.want {
				t.Errorf("Expected task status %q, got %q", tc.want, task.Status)
			}

			// every step keeps the status it reported
			reported := make(map[string]string, len(tc.reports))
			for _, r := range tc.reports {
				reported[r.name] = r.status
			}
			for _, test := range task.Tests {
				if want, ok := reported[test.Name]; ok && test.Status != want {
					t.Errorf("Expected step %s to show %q, got %q", test.Name, want, test.Status)
				}
			}
		})
	}
}
