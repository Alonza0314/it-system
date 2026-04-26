package processor

import (
	"backend/model"
	"fmt"
	"net/http"
)

func (p *Processor) RegisterRunner(req *model.RequestRegisterRunner) (*model.ResponseRegisterRunner, *model.ErrorDetail) {
	if p.itContext.RunnerExists(req.Name) {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusConflict,
			Detail:     "Runner with the same name already exists",
		}
	}

	if err := p.itContext.RegisterRunner(req.Name, req.IP); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to register runner: %v", err),
		}
	}

	return &model.ResponseRegisterRunner{
		Message: "Runner registered successfully",
	}, nil
}

func (p *Processor) DeleteRunner(name string) (*model.ResponseDeleteRunner, *model.ErrorDetail) {
	if !p.itContext.RunnerExists(name) {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusNotFound,
			Detail:     "Runner not found",
		}
	}

	if err := p.itContext.DeleteRunner(name); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to delete runner: %v", err),
		}
	}

	return &model.ResponseDeleteRunner{
		Message: "Runner deleted successfully",
	}, nil
}

func (p *Processor) GetRunners() (*model.ResponseGetRunners, *model.ErrorDetail) {
	return &model.ResponseGetRunners{
		Message: "Runners retrieved successfully",
		Runners: p.itContext.GetRunners(),
	}, nil
}
