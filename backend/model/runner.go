package model

type RequestRegisterRunner struct {
	Name string `json:"name" binding:"required"`
	IP   string `json:"ip" binding:"required"`
}

type ResponseRegisterRunner struct {
	Message string `json:"message" binding:"required"`
}

type ResponseDeleteRunner struct {
	Message string `json:"message" binding:"required"`
}

type ResponseGetRunners struct {
	Message string   `json:"message" binding:"required"`
	Runners []Runner `json:"runners,omitempty"`
}

type Runner struct {
	Name        string `json:"name" binding:"required"`
	IP          string `json:"ip" binding:"required"`
	OnGoingTask uint64 `json:"onGoingTask" binding:"required"`
	Status      string `json:"status" binding:"required"`
}
