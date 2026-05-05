package model

type RequestRegisterRunner struct {
	Name string `json:"name" binding:"required"`
	IP   string `json:"ip" binding:"required"`
}

type ResponseRegisterRunner struct {
	Message string `json:"message" binding:"required"`
	Token   string `json:"token,omitempty"`
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

type RequestRunnerHeartbeat struct {
	Idle        *bool  `json:"idle" binding:"required"`
	OnGoingTask uint64 `json:"onGoingTask,omitempty"`
}

type ResponseRunnerHeartbeat struct {
	Message  string   `json:"message" binding:"required"`
	Id       uint64   `json:"id,omitempty"`
	Tests    []string `json:"tests,omitempty"`
	NFPrList []NfPr   `json:"nfPrList,omitempty"`
}

type RequestTestOutput struct {
	EndFlag  *bool  `json:"endFlag" binding:"required"`
	Id       uint64 `json:"id" binding:"required"`
	TestName string `json:"testName,omitempty"`
	Success  bool   `json:"success,omitempty"`
	Log      string `json:"log,omitempty"`
}

type ResponseRunnerTestOutput struct {
	Message string `json:"message" binding:"required"`
}
