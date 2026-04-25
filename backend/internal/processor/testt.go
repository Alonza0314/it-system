package processor

import (
	"backend/constant"
	"backend/model"
	"fmt"
	"net/http"
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
