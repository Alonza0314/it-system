package processor

import (
	"backend/constant"
	"backend/model"
	"fmt"
	"net/http"

	"github.com/free-ran-ue/util"
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

	claims := map[string]interface{}{
		"user": req.Name,
	}
	token, err := util.CreateJWT(p.runnerJwtSecret, constant.RUNNER_JWT_SUBJECT_TAG, p.runnerJwtExpiresIn, claims)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to create JWT: %v", err),
		}
	}

	return &model.ResponseRegisterRunner{
		Message: "Runner registered successfully",
		Token:   token,
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
