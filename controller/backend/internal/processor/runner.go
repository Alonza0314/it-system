package processor

import (
	"fmt"
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"

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
		"user":                        req.Name,
		constant.USER_LEVEL_CLAIM_TAG: constant.USER_LEVEL_RUNNER,
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

func (p *Processor) RunnerHeartbeat(req *model.RequestRunnerHeartbeat, runner string) (*model.ResponseRunnerHeartbeat, *model.ErrorDetail) {
	task, err := p.itContext.GetFirstPendingTaskAndMoveToOngoing()
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get pending task: %v", err),
		}
	}

	if task == nil {
		if err := p.itContext.HeartbeatWithoutTask(runner); err != nil {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to update runner status: %v", err),
			}
		}
		return nil, nil
	}

	if err := p.itContext.HeartbeatWithTask(runner, task.ID()); err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to update runner status: %v", err),
		}
	}

	nfPrList := make([]model.NfPr, len(task.NFPrList()))
	for i, np := range task.NFPrList() {
		nfPrList[i] = model.NfPr{
			NfName: np.NFName(),
			PR:     np.PR(),
		}
	}

	return &model.ResponseRunnerHeartbeat{
		Message:  "Runner heartbeat successful",
		Id:       task.ID(),
		Tests:    task.Tests(),
		NFPrList: nfPrList,
	}, nil
}

func (p *Processor) TtestOutput(req *model.RequestTestOutput, runner string) *model.ErrorDetail {
	if *req.EndFlag {
		if err := p.itContext.TtestOutputEnd(req.Id); err != nil {
			return &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to end test output: %v", err),
			}
		}

		if err := p.itContext.HeartbeatWithoutTask(runner); err != nil {
			return &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to update runner status: %v", err),
			}
		}
	} else {
		if err := p.itContext.TtestOutputTransfer(req.Id, req.TestName, req.Success, &req.Log); err != nil {
			return &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to transfer test output: %v", err),
			}
		}
	}

	return nil
}
