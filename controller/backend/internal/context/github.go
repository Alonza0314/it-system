package context

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/constant"

	"github.com/free-ran-ue/util"
)

type pr struct {
	Num      int     `json:"number"`
	Tit      string  `json:"title"`
	Stat     string  `json:"state"`
	MergedAt *string `json:"merged_at"`
}

func (p *pr) Number() int {
	return p.Num
}

func (p *pr) Title() string {
	return p.Tit
}

// State is open, closed or merged. GitHub reports a merged PR as closed
// with merged_at set.
func (p *pr) State() string {
	if p.MergedAt != nil {
		return constant.GITHUB_PR_STATE_MERGED
	}
	return p.Stat
}

type githubContext struct {
	pullsApiUrl string
}

func newGithubContext() *githubContext {
	return &githubContext{
		pullsApiUrl: constant.GITHUB_FREE5GC_BASE_API_URL,
	}
}

// getPr returns nil without error when the PR does not exist.
func (ctx *githubContext) getPr(nf string, number int) (*pr, error) {
	apiUrl := fmt.Sprintf(ctx.pullsApiUrl+"/%d", nf, number)
	responseRaw, err := util.SendHttpRequest(apiUrl, http.MethodGet, nil, nil)
	if err != nil {
		return nil, err
	}

	if responseRaw.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if responseRaw.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get PR from %s: status code %d", apiUrl, responseRaw.StatusCode)
	}

	var p pr
	if err := json.Unmarshal(responseRaw.Body, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PR response from %s: %v", apiUrl, err)
	}

	return &p, nil
}

func (ctx *githubContext) getPrList(nf string) ([]pr, error) {
	apiUrl := fmt.Sprintf(ctx.pullsApiUrl, nf)
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
