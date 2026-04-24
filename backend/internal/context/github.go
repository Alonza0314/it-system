package context

import (
	"backend/constant"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/free-ran-ue/util"
)

type nf struct {
	name string
	prs  []pr
}

func (n *nf) Name() string {
	return n.name
}

func (n *nf) PRs() []pr {
	return n.prs
}

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

type githubContext struct {
	nfs []string
}

func newGithubContext() *githubContext {
	return &githubContext{
		nfs: constant.NF_LIST,
	}
}

func (gCtx *githubContext) getPrList() ([]nf, error) {
	nfs := make([]nf, 0, len(gCtx.nfs))

	for _, n := range gCtx.nfs {
		apiUrl := fmt.Sprintf(constant.GITHUB_FREE5GC_BASE_API_URL, n)
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

		nf := nf{
			name: n,
			prs:  prList,
		}
		nfs = append(nfs, nf)
	}

	return nfs, nil
}
