package model

type ResponseGetGithubPRs struct {
	Message string `json:"message" binding:"required"`
	PRs     []PR   `json:"prs,omitempty"`
}

type ResponseGetGithubPR struct {
	Message string `json:"message" binding:"required"`
	PR      *PR    `json:"pr,omitempty"`
}

type PR struct {
	Number int    `json:"number" binding:"required"`
	Title  string `json:"title" binding:"required"`
	State  string `json:"state,omitempty"`
}
