package context

import (
	"backend/constant"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/free-ran-ue/util"
)

type pr struct {
	Num int    `json:"number"`
	Tit string `json:"title"`
}

func (p *pr) Number() int {
	return p.Num
}

func (p *pr) Title() string {
	return p.Tit
}

type githubContext struct{}

func newGithubContext() *githubContext {
	return &githubContext{}
}

func (gCtx *githubContext) getPrList(nf string) ([]pr, error) {
	apiUrl := fmt.Sprintf(constant.GITHUB_FREE5GC_BASE_API_URL, nf)
	responseRaw, err := util.SendHttpRequest(apiUrl, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}

	if responseRaw.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get PRs from %s: status code %d", apiUrl, responseRaw.StatusCode)
	}

	var prList []pr
	if err := json.Unmarshal(responseRaw.Body, &prList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PR response from %s: %v", apiUrl, err)
	}

	return prList, nil
}
