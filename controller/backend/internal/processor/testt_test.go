package processor_test

import (
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/internal/processor"
	"github.com/Alonza0314/it-system/controller/backend/logger"
	"github.com/Alonza0314/it-system/controller/backend/model"

	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

func newTestProcessor(t *testing.T) *processor.Processor {
	t.Helper()

	dir := t.TempDir()
	p := processor.NewProcessor("admin", "admin", filepath.Join(dir, "test.db"), filepath.Join(dir, "log"), "secret", "runner-secret", 10,
		time.Hour, time.Hour, time.Minute, false, "", time.Hour, logger.NewBackendLogger(loggergoUtil.LEVEL_STRING_ERROR, "", true))
	t.Cleanup(func() {
		if err := processor.ReleaseProcessor(p); err != nil {
			t.Errorf("Failed to release processor: %v", err)
		}
	})

	return p
}

func TestRerunTask(t *testing.T) {
	p := newTestProcessor(t)

	if _, errDetail := p.AddTestcases(&model.RequestAddTestcases{Testcases: []model.Testcase{
		{Name: "TestRegistration", Script: "ci-test-it.sh", Label: "IT"},
		{Name: "TestPaging", Script: "ci-test-it.sh", Label: "IT"},
	}}); errDetail != nil {
		t.Fatalf("Failed to add testcases: %s", errDetail.Detail)
	}

	nfPrList := []model.NfPr{{NfName: "nrf", PR: 95}, {NfName: "amf", PR: 242}}
	if _, errDetail := p.SubmitTask(&model.RequestSubmitTask{
		Tests: []model.Test{
			{Name: "TestRegistration", Script: "old-script.sh"},
			{Name: "TestPaging", Script: "old-script.sh"},
		},
		NFPrList: nfPrList,
	}, "alice"); errDetail != nil {
		t.Fatalf("Failed to submit task: %s", errDetail.Detail)
	}
	tasks, _ := p.GetTasks()
	originalId := tasks.PendingTask[0].Id

	t.Run("RerunCopiesTestsAndNfPrs", func(t *testing.T) {
		if _, errDetail := p.RerunTask(originalId, "bob"); errDetail != nil {
			t.Fatalf("Failed to rerun task: %s", errDetail.Detail)
		}

		tasks, _ := p.GetTasks()
		if len(tasks.PendingTask) != 2 {
			t.Fatalf("Expected 2 pending tasks, got %d", len(tasks.PendingTask))
		}
		rerun := tasks.PendingTask[1]
		if rerun.Id == originalId || rerun.Username != "bob" {
			t.Errorf("Expected a new task owned by bob, got %+v", rerun)
		}

		task, errDetail := p.GetTask(rerun.Id)
		if errDetail != nil {
			t.Fatalf("Failed to get rerun task: %s", errDetail.Detail)
		}
		if !reflect.DeepEqual(task.NFPrList, nfPrList) {
			t.Errorf("Expected NF PR list %v, got %v", nfPrList, task.NFPrList)
		}
		names := make([]string, 0, len(task.Tests))
		for _, test := range task.Tests {
			names = append(names, test.Name)
		}
		if len(names) < 2 || names[0] != "TestRegistration" || names[1] != "TestPaging" {
			t.Errorf("Expected TestRegistration and TestPaging first, got %v", names)
		}
		if len(names) != 6 {
			t.Errorf("Expected 2 testcases plus 4 pipeline steps, got %v", names)
		}

		// The runner receives the scripts of the current testcases, not the original ones
		if _, errDetail := p.RegisterRunner(&model.RequestRegisterRunner{Name: "runner", IP: "127.0.0.1"}); errDetail != nil {
			t.Fatalf("Failed to register runner: %s", errDetail.Detail)
		}
		idle := true
		var assigned *model.ResponseRunnerHeartbeat
		for range 2 {
			if assigned, errDetail = p.RunnerHeartbeat(&model.RequestRunnerHeartbeat{Idle: &idle}, "runner"); errDetail != nil {
				t.Fatalf("Failed to send runner heartbeat: %s", errDetail.Detail)
			}
		}
		if assigned == nil || assigned.Id != rerun.Id {
			t.Fatalf("Expected runner to receive task %d, got %+v", rerun.Id, assigned)
		}
		for _, test := range assigned.Tests[:2] {
			if test.Script != "ci-test-it.sh" {
				t.Errorf("Expected script ci-test-it.sh for %s, got %q", test.Name, test.Script)
			}
		}
	})

	t.Run("RerunFailsForDeletedTestcase", func(t *testing.T) {
		if _, errDetail := p.DeleteTestcases(&model.RequestDeleteTestcases{Testcases: []model.Testcase{{Name: "TestPaging"}}}); errDetail != nil {
			t.Fatalf("Failed to delete testcase: %s", errDetail.Detail)
		}

		_, errDetail := p.RerunTask(originalId, "bob")
		if errDetail == nil || errDetail.HttpStatus != http.StatusBadRequest || !strings.Contains(errDetail.Detail, "TestPaging") {
			t.Errorf("Expected bad request naming TestPaging, got %+v", errDetail)
		}
	})

	t.Run("RerunFailsForUnknownTask", func(t *testing.T) {
		_, errDetail := p.RerunTask(originalId+100, "bob")
		if errDetail == nil || errDetail.HttpStatus != http.StatusNotFound {
			t.Errorf("Expected not found, got %+v", errDetail)
		}
	})
}
