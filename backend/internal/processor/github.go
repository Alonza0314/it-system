package processor

import (
	"backend/model"
	"fmt"
	"net/http"
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
		}
	}

	return response, nil
}
