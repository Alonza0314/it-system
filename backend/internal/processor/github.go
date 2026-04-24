package processor

import (
	"backend/model"
	"fmt"
	"net/http"
)

func (p *Processor) GetGithubPRs() (*model.ResponseGetGithubPRs, *model.ErrorDetail) {
	nfs, err := p.itContext.GetPrList()
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to get PR list: %v", err),
		}
	}
	p.GitLog.Debugf("Retrieved %d NFs", len(nfs))
	p.GitLog.Tracef("PRs details: %+v", nfs)

	response := &model.ResponseGetGithubPRs{
		Message: "PRs retrieved successfully",
		NFs:     make([]model.NfPRs, 0, len(nfs)),
	}
	for i, nf := range nfs {
		response.NFs = append(response.NFs, model.NfPRs{
			Name: nf.Name(),
			PRs:  make([]model.PR, 0, len(nf.PRs())),
		})
		for _, pr := range nf.PRs() {
			response.NFs[i].PRs = append(response.NFs[i].PRs, model.PR{
				Number: pr.Number(),
				Title:  pr.Title(),
			})
		}
	}

	return response, nil
}
