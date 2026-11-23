package processor_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/internal/processor"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

func pendingTasks(t *testing.T, p *processor.Processor) int {
	t.Helper()

	tasks, errDetail := p.GetTasks()
	if errDetail != nil {
		t.Fatalf("Failed to get tasks: %s", errDetail.Detail)
	}

	return len(tasks.PendingTask)
}

func TestSubmitTaskRejectsDuplicateNf(t *testing.T) {
	p := newTestProcessor(t)
	tests := []model.Test{{Name: "TestRegistration", Script: "ci-test-it.sh"}}

	for _, tc := range []struct {
		name     string
		nfPrList []model.NfPr
		wantErr  string // empty: the task is created
	}{
		{
			// task 629: the runner tested only udr#87
			name:     "SameNfTwice",
			nfPrList: []model.NfPr{{NfName: "udr", PR: 82}, {NfName: "udr", PR: 87}},
			wantErr:  "NF selected more than once: udr",
		},
		{
			name: "TwoNfsMoreThanOnce",
			nfPrList: []model.NfPr{
				{NfName: "udr", PR: 82}, {NfName: "amf", PR: 325}, {NfName: "amf", PR: 306},
				{NfName: "udr", PR: 87}, {NfName: "udr", PR: 86},
			},
			wantErr: "NF selected more than once: amf, udr",
		},
		{
			name:     "DifferentNfs",
			nfPrList: []model.NfPr{{NfName: "nrf", PR: 95}, {NfName: "amf", PR: 242}, {NfName: "free5gc", PR: 100}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := pendingTasks(t, p)

			_, errDetail := p.SubmitTask(&model.RequestSubmitTask{Tests: tests, NFPrList: tc.nfPrList}, "alice")

			created := pendingTasks(t, p) - before
			if tc.wantErr == "" {
				if errDetail != nil {
					t.Fatalf("Failed to submit task: %s", errDetail.Detail)
				}
				if created != 1 {
					t.Errorf("Expected 1 new task, got %d", created)
				}
				return
			}

			if errDetail == nil || errDetail.HttpStatus != http.StatusBadRequest || errDetail.Detail != tc.wantErr {
				t.Errorf("Expected bad request %q, got %+v", tc.wantErr, errDetail)
			}
			if created != 0 {
				t.Errorf("Expected no task to be created, got %d", created)
			}
		})
	}
}

// A task with the same NF twice that was stored before the check is not run
// again.
func TestRerunTaskRejectsDuplicateNf(t *testing.T) {
	p := newTestProcessor(t)

	if _, errDetail := p.AddTestcases(&model.RequestAddTestcases{Testcases: []model.Testcase{
		{Name: "TestRegistration", Script: "ci-test-it.sh", Label: "IT"},
	}}); errDetail != nil {
		t.Fatalf("Failed to add testcases: %s", errDetail.Detail)
	}
	if err := p.CreateTaskUnchecked("djhih", time.Now().Unix(),
		[]model.Test{{Name: "TestRegistration", Script: "ci-test-it.sh"}},
		[]model.NfPr{{NfName: "udr", PR: 82}, {NfName: "udr", PR: 87}},
	); err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	tasks, _ := p.GetTasks()
	id := tasks.PendingTask[0].Id

	_, errDetail := p.RerunTask(id, "bob")
	if errDetail == nil || errDetail.HttpStatus != http.StatusBadRequest || errDetail.Detail != "NF selected more than once: udr" {
		t.Errorf("Expected bad request naming udr, got %+v", errDetail)
	}
	if pending := pendingTasks(t, p); pending != 1 {
		t.Errorf("Expected only the original task, got %d pending tasks", pending)
	}
}
