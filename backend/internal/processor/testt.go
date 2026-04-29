package processor

import (
	"backend/constant"
	"backend/model"
	"fmt"
	"net/http"
	"time"
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
	for name, link := range testcaseMap {
		testcases = append(testcases, model.Testcase{
			Name: name,
			Link: link,
		})
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
		if err := p.itContext.SaveToDb(constant.BUCKET_TESTCASE, testcase.Name, testcase.Link); err != nil {
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
