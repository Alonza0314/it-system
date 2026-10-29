package processor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/internal/logbrief"
	"github.com/Alonza0314/it-system/controller/backend/model"
)

func (p *Processor) GetTestcases() (*model.ResponseGetTestcases, *model.ErrorDetail) {
	testcaseMap, err := p.itContext.LoadAllFromDb(constant.BUCKET_TESTCASE)
	if err != nil {
		p.ProcLog.Errorf("Failed to load testcases from database: %v", err)
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to load testcases from database: %v", err),
		}
	}
	p.ProcLog.Debugf("Retrieved %d testcases", len(testcaseMap))
	p.ProcLog.Tracef("Testcases details: %+v", testcaseMap)

	testcases := make([]model.Testcase, 0, len(testcaseMap))
	for name, tcBytes := range testcaseMap {
		var testcase model.Testcase
		if err := json.Unmarshal([]byte(tcBytes), &testcase); err != nil {
			p.ProcLog.Errorf("Failed to unmarshal testcase %s: %v", name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to unmarshal testcase %s: %v", name, err),
			}
		}
		testcases = append(testcases, testcase)
	}

	response := &model.ResponseGetTestcases{
		Message:   "Testcases retrieved successfully",
		Testcases: testcases,
	}

	return response, nil
}

func (p *Processor) AddTestcases(req *model.RequestAddTestcases) (*model.ResponseAddTestcases, *model.ErrorDetail) {
	for _, testcase := range req.Testcases {
		exists, err := p.itContext.ExistsInDb(constant.BUCKET_TESTCASE, testcase.Name)
		if err != nil {
			p.ProcLog.Errorf("Failed to check if testcase %s exists in database: %v", testcase.Name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if testcase %s exists in database: %v", testcase.Name, err),
			}
		}
		if exists {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusConflict,
				Detail:     fmt.Sprintf("Testcase %s already exists", testcase.Name),
			}
		}
	}
	p.ProcLog.Debugf("Adding %d testcases", len(req.Testcases))
	p.ProcLog.Tracef("Testcases to add details: %+v", req.Testcases)

	for _, testcase := range req.Testcases {
		currentId := 1
		exists, err := p.itContext.ExistsInDb(constant.BUCKET_LABEL, testcase.Label)
		if err != nil {
			p.ProcLog.Errorf("Failed to check if label %s exists in database: %v", testcase.Label, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if label %s exists in database: %v", testcase.Label, err),
			}
		}
		if exists {
			latestIdString, err := p.itContext.LoadFromDb(constant.BUCKET_LABEL, testcase.Label)
			if err != nil {
				p.ProcLog.Errorf("Failed to load latest ID for label %s from database: %v", testcase.Label, err)
				return nil, &model.ErrorDetail{
					HttpStatus: http.StatusInternalServerError,
					Detail:     fmt.Sprintf("Failed to load latest ID for label %s from database: %v", testcase.Label, err),
				}
			}
			latestId, err := strconv.Atoi(latestIdString)
			if err != nil {
				p.ProcLog.Errorf("Failed to convert latest ID for label %s to integer: %v", testcase.Label, err)
				return nil, &model.ErrorDetail{
					HttpStatus: http.StatusInternalServerError,
					Detail:     fmt.Sprintf("Failed to convert latest ID for label %s to integer: %v", testcase.Label, err),
				}
			}
			currentId = latestId + 1

			if err := p.itContext.SaveToDb(constant.BUCKET_LABEL, testcase.Label, strconv.Itoa(currentId)); err != nil {
				p.ProcLog.Errorf("Failed to update latest ID for label %s in database: %v", testcase.Label, err)
				return nil, &model.ErrorDetail{
					HttpStatus: http.StatusInternalServerError,
					Detail:     fmt.Sprintf("Failed to update latest ID for label %s in database: %v", testcase.Label, err),
				}
			}
		} else {
			if err := p.itContext.SaveToDb(constant.BUCKET_LABEL, testcase.Label, strconv.Itoa(currentId)); err != nil {
				p.ProcLog.Errorf("Failed to save new label %s with ID %d in database: %v", testcase.Label, currentId, err)
				return nil, &model.ErrorDetail{
					HttpStatus: http.StatusInternalServerError,
					Detail:     fmt.Sprintf("Failed to save new label %s with ID %d in database: %v", testcase.Label, currentId, err),
				}
			}
		}

		testcase.Id = currentId

		tcBytes, err := json.Marshal(testcase)
		if err != nil {
			p.ProcLog.Errorf("Failed to marshal testcase %s: %v", testcase.Name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to marshal testcase %s: %v", testcase.Name, err),
			}
		}

		if err := p.itContext.SaveToDb(constant.BUCKET_TESTCASE, testcase.Name, string(tcBytes)); err != nil {
			p.ProcLog.Errorf("Failed to save testcase %s to database: %v", testcase.Name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to save testcase %s to database: %v", testcase.Name, err),
			}
		}
	}

	return &model.ResponseAddTestcases{
		Message: "Testcases added successfully",
	}, nil
}

func (p *Processor) DeleteTestcases(req *model.RequestDeleteTestcases) (*model.ResponseDeleteTestcases, *model.ErrorDetail) {
	for _, testcase := range req.Testcases {
		exists, err := p.itContext.ExistsInDb(constant.BUCKET_TESTCASE, testcase.Name)
		if err != nil {
			p.ProcLog.Errorf("Failed to check if testcase %s exists in database: %v", testcase.Name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if testcase %s exists in database: %v", testcase.Name, err),
			}
		}
		if !exists {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusNotFound,
				Detail:     fmt.Sprintf("Testcase %s not found", testcase.Name),
			}
		}
	}
	p.ProcLog.Debugf("Deleting %d testcases", len(req.Testcases))
	p.ProcLog.Tracef("Testcases to delete details: %+v", req.Testcases)

	for _, testcase := range req.Testcases {
		if err := p.itContext.RemoveFromDb(constant.BUCKET_TESTCASE, testcase.Name); err != nil {
			p.ProcLog.Errorf("Failed to remove testcase %s from database: %v", testcase.Name, err)
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to remove testcase %s from database: %v", testcase.Name, err),
			}
		}
	}

	return &model.ResponseDeleteTestcases{
		Message: "Testcases deleted successfully",
	}, nil
}

func (p *Processor) GetTasks() (*model.ResponseGetTasks, *model.ErrorDetail) {
	pendingTasks, ongoingTasks, historyTasks := p.itContext.GetPendingTasks(), p.itContext.GetOngoingTasks(), p.itContext.GetHistoryTasks()
	p.ProcLog.Debugf("Retrieved %d pending tasks, %d ongoing tasks, and %d history tasks", len(pendingTasks), len(ongoingTasks), len(historyTasks))
	p.ProcLog.Tracef("Pending tasks details: %+v", pendingTasks)
	p.ProcLog.Tracef("Ongoing tasks details: %+v", ongoingTasks)
	p.ProcLog.Tracef("History tasks details: %+v", historyTasks)

	return &model.ResponseGetTasks{
		Message:     "Tasks retrieved successfully",
		PendingTask: pendingTasks,
		OngoingTask: ongoingTasks,
		HistoryTask: historyTasks,
	}, nil
}

func (p *Processor) GetTasksDetail() (*model.ResponseGetTasksDetail, *model.ErrorDetail) {
	pendingTasks, ongoingTasks, historyTasks := p.itContext.GetPendingTaskDetails(), p.itContext.GetOngoingTaskDetails(), p.itContext.GetHistoryTaskDetails()
	p.ProcLog.Debugf("Retrieved details of %d pending tasks, %d ongoing tasks, and %d history tasks", len(pendingTasks), len(ongoingTasks), len(historyTasks))

	return &model.ResponseGetTasksDetail{
		Message:     "Tasks retrieved successfully",
		PendingTask: pendingTasks,
		OngoingTask: ongoingTasks,
		HistoryTask: historyTasks,
	}, nil
}

func (p *Processor) GetTask(id uint64) (*model.ResponseGetTask, *model.ErrorDetail) {
	task, err := p.itContext.GetTask(id)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusNotFound,
			Detail:     fmt.Sprintf("Task with ID %d not found", id),
		}
	}

	testDetails := make([]model.TestDetail, 0, len(task.Tests()))
	for _, pipeline := range task.Pipelines() {
		testDetails = append(testDetails, model.TestDetail{
			Name:   pipeline.Name(),
			Status: pipeline.Status(),
		})
	}

	response := &model.ResponseGetTask{
		Message:    "Task retrieved successfully",
		Id:         task.ID(),
		Username:   task.Username(),
		Status:     task.Status(),
		CreateTime: task.CreateTime(),
		Tests:      testDetails,
	}

	for _, nfPr := range task.NFPrList() {
		response.NFPrList = append(response.NFPrList, model.NfPr{
			NfName: nfPr.NFName(),
			PR:     nfPr.PR(),
		})
	}

	return response, nil
}

func (p *Processor) SubmitTask(req *model.RequestSubmitTask, username string) (*model.ResponseSubmitTask, *model.ErrorDetail) {
	nowTime := time.Now().Unix()
	if err := p.itContext.CreateTask(username, nowTime, req.Tests, req.NFPrList); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to create task: %v", err),
		}
	}

	return &model.ResponseSubmitTask{
		Message: "Task submitted successfully",
	}, nil
}

// RerunTask submits a new task, owned by username, with the testcases and
// NF PRs of task id. Scripts come from the current testcases instead of the
// original task: the runner always tests the latest ci-test, and history
// tasks loaded from the database do not keep their scripts.
func (p *Processor) RerunTask(id uint64, username string) (*model.ResponseRerunTask, *model.ErrorDetail) {
	task, err := p.itContext.GetTask(id)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusNotFound,
			Detail:     fmt.Sprintf("Task with ID %d not found", id),
		}
	}

	testcases, errDetail := p.GetTestcases()
	if errDetail != nil {
		return nil, errDetail
	}
	scripts := make(map[string]string, len(testcases.Testcases))
	for _, testcase := range testcases.Testcases {
		scripts[testcase.Name] = testcase.Script
	}

	req := &model.RequestSubmitTask{}
	missing := make([]string, 0)
	for _, pipeline := range task.Pipelines() {
		switch pipeline.Name() {
		case constant.TESTCASE_PREPARE_FREE5GC, constant.TESTCASE_FETCH_PRS, constant.TESTCASE_MAKE_NF, constant.TESTCASE_CLEANUP:
			continue
		}
		script := scripts[pipeline.Name()]
		if script == "" {
			missing = append(missing, pipeline.Name())
			continue
		}
		req.Tests = append(req.Tests, model.Test{Name: pipeline.Name(), Script: script})
	}
	if len(missing) > 0 {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusBadRequest,
			Detail:     fmt.Sprintf("Testcases not found or without script: %s", strings.Join(missing, ", ")),
		}
	}
	if len(req.Tests) == 0 {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusBadRequest,
			Detail:     fmt.Sprintf("Task with ID %d has no testcase to re-run", id),
		}
	}

	for _, nfPr := range task.NFPrList() {
		req.NFPrList = append(req.NFPrList, model.NfPr{
			NfName: nfPr.NFName(),
			PR:     nfPr.PR(),
		})
	}

	if _, errDetail := p.SubmitTask(req, username); errDetail != nil {
		return nil, errDetail
	}

	return &model.ResponseRerunTask{
		Message: fmt.Sprintf("Task %d submitted again", id),
	}, nil
}

func (p *Processor) CancelTask(id uint64) (*model.ResponseCancelTask, *model.ErrorDetail) {
	if err := p.itContext.CancelTask(id); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to cancel task: %v", err),
		}
	}

	return &model.ResponseCancelTask{
		Message: "Task cancelled successfully",
	}, nil
}

func (p *Processor) DeleteTasksHistory() (*model.ResponseDeleteTasksHistory, *model.ErrorDetail) {
	if err := p.itContext.DeleteHistory(); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to delete tasks history: %v", err),
		}
	}

	return &model.ResponseDeleteTasksHistory{
		Message: "Tasks history deleted successfully",
	}, nil
}

func (p *Processor) GetTestLog(id uint64, testName string) (*model.ResponseGetTestLog, *model.ErrorDetail) {
	log, err := p.itContext.GetTestLog(id, testName)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get test log: %v", err),
		}
	}

	return &model.ResponseGetTestLog{
		Message: "Test log retrieved successfully",
		Log:     log,
		Brief:   logbrief.Summarize(log),
	}, nil
}
