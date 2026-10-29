package processor

import (
	"fmt"
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/model"
)

func (p *Processor) GetGithubPRs(nf string) (*model.ResponseGetGithubPRs, *model.ErrorDetail) {
	prs, err := p.itContext.GetPrList(nf)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get PR list: %v", err),
		}
	}
	p.GitLog.Debugf("Retrieved %d PRs", len(prs))
	p.GitLog.Tracef("PRs details: %+v", prs)

	response := &model.ResponseGetGithubPRs{
		Message: "PRs retrieved successfully",
		PRs:     make([]model.PR, len(prs)),
	}

	for i, pr := range prs {
		response.PRs[i] = model.PR{
			Number: pr.Number(),
			Title:  pr.Title(),
			State:  pr.State(),
		}
	}

	return response, nil
}

func (p *Processor) GetGithubPR(nf string, number int) (*model.ResponseGetGithubPR, *model.ErrorDetail) {
	pr, err := p.itContext.GetPr(nf, number)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get PR: %v", err),
		}
	}
	if pr == nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusNotFound,
			Detail:     fmt.Sprintf("PR #%d not found in free5gc/%s", number, nf),
		}
	}
	p.GitLog.Tracef("PR details: %+v", pr)

	return &model.ResponseGetGithubPR{
		Message: "PR retrieved successfully",
		PR: &model.PR{
			Number: pr.Number(),
			Title:  pr.Title(),
			State:  pr.State(),
		},
	}, nil
}
