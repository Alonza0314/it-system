package model

type ResponseGetGithubPRs struct {
	Message string  `json:"message" binding:"required"`
	NFs     []NfPRs `json:"nfs,omitempty"`
}

type NfPRs struct {
	Name string `json:"name" binding:"required"`
	PRs  []PR   `json:"prs,omitempty"`
}

type PR struct {
	Number int    `json:"number" binding:"required"`
	Title  string `json:"title" binding:"required"`
}
