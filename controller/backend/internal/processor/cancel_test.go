package processor_test

import (
	"net/http"
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/internal/processor"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

func submitTestTask(t *testing.T, p *processor.Processor) uint64 {
	t.Helper()

	if _, errDetail := p.SubmitTask(&model.RequestSubmitTask{
		Tests: []model.Test{
			{Name: "TestRegistration", Script: "ci-test-it.sh"},
			{Name: "TestPaging", Script: "ci-test-it.sh"},
		},
		NFPrList: []model.NfPr{{NfName: "amf", PR: 285}},
	}, "alice"); errDetail != nil {
		t.Fatalf("Failed to submit task: %s", errDetail.Detail)
	}

	tasks, _ := p.GetTasks()
	return tasks.PendingTask[len(tasks.PendingTask)-1].Id
}

func testOutput(id uint64, testName, status string, end bool) *model.RequestTestOutput {
	return &model.RequestTestOutput{
		EndFlag:  &end,
		Id:       id,
		TestName: testName,
		Success:  status == constant.TASK_STATUS_SUCCESS,
		Status:   status,
		Log:      "log of " + testName,
	}
}

func runnerState(t *testing.T, p *processor.Processor, name string) model.Runner {
	t.Helper()

	runners, _ := p.GetRunners()
	for _, r := range runners.Runners {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("Runner %s not found", name)
	return model.Runner{}
}

func TestCancelTask(t *testing.T) {
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

	t.Run("PendingTaskIsDropped", func(t *testing.T) {
		id := submitTestTask(t, p)

		if _, errDetail := p.CancelTask(id); errDetail != nil {
			t.Fatalf("Failed to cancel pending task: %s", errDetail.Detail)
		}

		tasks, _ := p.GetTasks()
		if len(tasks.PendingTask) != 0 || len(tasks.HistoryTask) != 0 {
			t.Errorf("Expected the pending task to be dropped, got %+v", tasks)
		}
	})

	var runningId uint64
	t.Run("RunningTaskMovesToHistoryAsCanceled", func(t *testing.T) {
		runningId = submitTestTask(t, p)

		idle := true
		if _, errDetail := p.RunnerHeartbeat(&model.RequestRunnerHeartbeat{Idle: &idle}, "runner-1"); errDetail != nil {
			t.Fatalf("Heartbeat failed: %s", errDetail.Detail)
		}
		if r := runnerState(t, p, "runner-1"); r.Status != constant.RUNNER_STATUS_RUNNING || r.OnGoingTask != runningId {
			t.Fatalf("Expected runner-1 to run task %d, got %+v", runningId, r)
		}

		canceled, errDetail := p.TtestOutput(testOutput(runningId, constant.TESTCASE_PREPARE_FREE5GC, constant.TASK_STATUS_SUCCESS, false), "runner-1")
		if errDetail != nil || canceled {
			t.Fatalf("Expected output of a running task to be stored, got canceled=%v err=%v", canceled, errDetail)
		}

		if _, errDetail := p.CancelTask(runningId); errDetail != nil {
			t.Fatalf("Failed to cancel running task: %s", errDetail.Detail)
		}

		task, errDetail := p.GetTask(runningId)
		if errDetail != nil {
			t.Fatalf("Failed to get canceled task: %s", errDetail.Detail)
		}
		if task.Status != constant.TASK_STATUS_CANCELED {
			t.Errorf("Expected task status %q, got %q", constant.TASK_STATUS_CANCELED, task.Status)
		}
		for _, test := range task.Tests {
			want := constant.TASK_STATUS_CANCELED
			if test.Name == constant.TESTCASE_PREPARE_FREE5GC {
				want = constant.TASK_STATUS_SUCCESS
			}
			if test.Status != want {
				t.Errorf("Expected %s to be %q, got %q", test.Name, want, test.Status)
			}
		}

		if r := runnerState(t, p, "runner-1"); r.Status != constant.RUNNER_STATUS_IDLE || r.OnGoingTask != 0 {
			t.Errorf("Expected runner-1 to be released, got %+v", r)
		}
	})

	t.Run("RunnerIsToldToStop", func(t *testing.T) {
		canceled, errDetail := p.TtestOutput(testOutput(runningId, "TestRegistration", constant.TASK_STATUS_SUCCESS, false), "runner-1")
		if errDetail != nil || !canceled {
			t.Fatalf("Expected a canceled reply for output of a canceled task, got canceled=%v err=%v", canceled, errDetail)
		}

		canceled, errDetail = p.TtestOutput(testOutput(runningId, "", constant.TASK_STATUS_CANCELED, true), "runner-1")
		if errDetail != nil || !canceled {
			t.Fatalf("Expected a canceled reply for the end of a canceled task, got canceled=%v err=%v", canceled, errDetail)
		}
		if r := runnerState(t, p, "runner-1"); r.Status != constant.RUNNER_STATUS_IDLE {
			t.Errorf("Expected runner-1 to be idle after it ended the task, got %+v", r)
		}

		task, _ := p.GetTask(runningId)
		for _, test := range task.Tests {
			if test.Name == "TestRegistration" && test.Status != constant.TASK_STATUS_CANCELED {
				t.Errorf("Output that arrived after the cancel must not change the result, got %q", test.Status)
			}
		}

		tasks, _ := p.GetTasks()
		count := 0
		for _, h := range tasks.HistoryTask {
			if h.Id == runningId {
				count++
			}
		}
		if len(tasks.OngoingTask) != 0 || count != 1 {
			t.Errorf("Expected the task once in the history and nothing ongoing, got %+v", tasks)
		}
	})

	t.Run("FinishedTaskCannotBeCanceled", func(t *testing.T) {
		_, errDetail := p.CancelTask(runningId)
		if errDetail == nil || errDetail.HttpStatus != http.StatusConflict {
			t.Errorf("Expected %d for a finished task, got %+v", http.StatusConflict, errDetail)
		}
	})

	t.Run("UnknownTask", func(t *testing.T) {
		_, errDetail := p.CancelTask(999999)
		if errDetail == nil || errDetail.HttpStatus != http.StatusNotFound {
			t.Errorf("Expected %d for an unknown task, got %+v", http.StatusNotFound, errDetail)
		}
	})

	t.Run("NormalTaskStillFinishes", func(t *testing.T) {
		id := submitTestTask(t, p)

		idle := true
		if _, errDetail := p.RunnerHeartbeat(&model.RequestRunnerHeartbeat{Idle: &idle}, "runner-1"); errDetail != nil {
			t.Fatalf("Heartbeat failed: %s", errDetail.Detail)
		}

		for _, name := range []string{constant.TESTCASE_PREPARE_FREE5GC, constant.TESTCASE_FETCH_PRS, constant.TESTCASE_MAKE_NF, "TestRegistration", "TestPaging"} {
			if canceled, errDetail := p.TtestOutput(testOutput(id, name, constant.TASK_STATUS_SUCCESS, false), "runner-1"); errDetail != nil || canceled {
				t.Fatalf("Output %s: canceled=%v err=%v", name, canceled, errDetail)
			}
		}
		if canceled, errDetail := p.TtestOutput(testOutput(id, constant.TESTCASE_CLEANUP, constant.TASK_STATUS_SUCCESS, true), "runner-1"); errDetail != nil || canceled {
			t.Fatalf("End of task: canceled=%v err=%v", canceled, errDetail)
		}

		task, _ := p.GetTask(id)
		if task.Status != constant.TASK_STATUS_SUCCESS {
			t.Errorf("Expected task status %q, got %q", constant.TASK_STATUS_SUCCESS, task.Status)
		}
		if r := runnerState(t, p, "runner-1"); r.Status != constant.RUNNER_STATUS_IDLE {
			t.Errorf("Expected runner-1 to be idle, got %+v", r)
		}
	})
}
