package context_test

import (
	"reflect"
	"testing"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

func findTaskDetail(details []model.TaskDetail, id uint64) *model.TaskDetail {
	for _, d := range details {
		if d.Id == id {
			return &d
		}
	}
	return nil
}

func TestTaskDetails(t *testing.T) {
	tests := []model.Test{{Name: "TestRegistration", Script: "ci-test-it.sh"}}
	nfPrList := []model.NfPr{{NfName: "nrf", PR: 95}, {NfName: "amf", PR: 242}}

	if err := ctx.CreateTask("tester", 1777143861, tests, nfPrList); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	pending := ctx.GetPendingTaskDetails()
	if len(pending) == 0 {
		t.Fatalf("Expected the new task in pending details")
	}
	id := pending[len(pending)-1].Id

	checkDetail := func(queue string, details []model.TaskDetail, expectedStatus string) {
		t.Helper()

		detail := findTaskDetail(details, id)
		if detail == nil {
			t.Fatalf("Task %d not found in %s details", id, queue)
		}
		if detail.Username != "tester" || detail.CreateTime != 1777143861 {
			t.Errorf("%s: unexpected username or create time: %+v", queue, detail)
		}
		if detail.Status != expectedStatus {
			t.Errorf("%s: expected status %s, got %s", queue, expectedStatus, detail.Status)
		}
		if !reflect.DeepEqual(detail.NFPrList, nfPrList) {
			t.Errorf("%s: expected NF PR list %v, got %v", queue, nfPrList, detail.NFPrList)
		}

		names := make([]string, len(detail.Tests))
		for i, test := range detail.Tests {
			names[i] = test.Name
		}
		expectedNames := []string{
			"TestRegistration",
			constant.TESTCASE_PREPARE_FREE5GC,
			constant.TESTCASE_FETCH_PRS,
			constant.TESTCASE_MAKE_NF,
			constant.TESTCASE_CLEANUP,
		}
		if !reflect.DeepEqual(names, expectedNames) {
			t.Errorf("%s: expected tests %v, got %v", queue, expectedNames, names)
		}
	}

	checkDetail("pending", pending, constant.TASK_STATUS_PENDING)

	task, err := ctx.GetFirstPendingTaskAndMoveToOngoing()
	if err != nil || task == nil || task.ID() != id {
		t.Fatalf("Failed to move task %d to ongoing: task %v, error %v", id, task, err)
	}
	checkDetail("ongoing", ctx.GetOngoingTaskDetails(), constant.TASK_STATUS_RUNNING)

	if err := ctx.TtestOutputEnd(id); err != nil {
		t.Fatalf("Failed to move task to history: %v", err)
	}
	// None of the tests ran, so the task ends as failed
	checkDetail("history", ctx.GetHistoryTaskDetails(), constant.TASK_STATUS_FAILED)
}
