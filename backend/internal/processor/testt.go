package processor

import (
	"backend/model"
	"fmt"
	"net/http"
)

func (p *Processor) GetTestcases() (*model.ResponseGetTestcases, *model.ErrorDetail) {
	testcases, err := p.itContext.GetTestcases()
	if err != nil {
		p.ProcLog.Errorf("Failed to get testcases: %v", err)
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get testcases: %v", err),
		}
	}
	p.ProcLog.Debugf("Retrieved %d testcases", len(testcases))
	p.ProcLog.Tracef("Testcases details: %+v", testcases)

	response := &model.ResponseGetTestcases{
		Message:   "Testcases retrieved successfully",
		Testcases: make([]model.Testcase, 0, len(testcases)),
	}
	for _, tc := range testcases {
		response.Testcases = append(response.Testcases, model.Testcase{
			Name: tc.Name(),
			Link: tc.Link(),
		})
	}

	return response, nil
}

func (p *Processor) AddTestcases(req *model.RequestAddTestcases) (*model.ResponseAddTestcases, *model.ErrorDetail) {
	testcases := p.itContext.ConvertToTestcase(req.Testcases)
	p.ProcLog.Debugf("Adding %d testcases", len(testcases))
	p.ProcLog.Tracef("Testcases to add details: %+v", testcases)

	if err := p.itContext.AddTestcases(testcases); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to add testcases: %v", err),
		}
	}

	return &model.ResponseAddTestcases{
		Message: "Testcases added successfully",
	}, nil
}

func (p *Processor) DeleteTestcases(req *model.RequestDeleteTestcases) (*model.ResponseDeleteTestcases, *model.ErrorDetail) {
	testcases := p.itContext.ConvertToTestcase(req.Testcases)
	p.ProcLog.Debugf("Deleting %d testcases", len(testcases))
	p.ProcLog.Tracef("Testcases to delete details: %+v", testcases)

	if err := p.itContext.DeleteTestcases(testcases); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to delete testcases: %v", err),
		}
	}

	return &model.ResponseDeleteTestcases{
		Message: "Testcases deleted successfully",
	}, nil
}
