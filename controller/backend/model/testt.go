package model

type ResponseGetTestcases struct {
	Message   string     `json:"message" binding:"required"`
	Testcases []Testcase `json:"testcases,omitempty"`
}

type RequestAddTestcases struct {
	Testcases []Testcase `json:"testcases" binding:"required"`
}

type ResponseAddTestcases struct {
	Message string `json:"message" binding:"required"`
}

type RequestDeleteTestcases struct {
	Testcases []Testcase `json:"testcases" binding:"required"`
}

type ResponseDeleteTestcases struct {
	Message string `json:"message" binding:"required"`
}

type Testcase struct {
	Name string `json:"name" binding:"required"`
	Link string `json:"link,omitempty"`
}

type ResponseGetTasks struct {
	Message     string       `json:"message" binding:"required"`
	PendingTask []TaskSimple `json:"pendingTask,omitempty"`
	OngoingTask []TaskSimple `json:"ongoingTask,omitempty"`
	HistoryTask []TaskSimple `json:"historyTask,omitempty"`
}

type TaskSimple struct {
	Id         uint64 `json:"id" binding:"required"`
	Username   string `json:"username" binding:"required"`
	CreateTime int64  `json:"createTime" binding:"required"`
}

type ResponseGetTask struct {
	Message    string       `json:"message" binding:"required"`
	Id         uint64       `json:"id,omitempty"`
	Username   string       `json:"username,omitempty"`
	Status     string       `json:"status,omitempty"`
	CreateTime int64        `json:"createTime,omitempty"`
	Tests      []TestDetail `json:"tests,omitempty"`
	NFPrList   []NfPr       `json:"nfPrList,omitempty"`
}

type TestDetail struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status" binding:"required"`
}

type RequestSubmitTask struct {
	Tests    []string `json:"tests" binding:"required"`
	NFPrList []NfPr   `json:"nfPrList" binding:"required"`
}

type NfPr struct {
	NfName string `json:"nfName" binding:"required"`
	PR     int    `json:"pr" binding:"required"`
}

type ResponseSubmitTask struct {
	Message string `json:"message" binding:"required"`
}

type ResponseCancelTask struct {
	Message string `json:"message" binding:"required"`
}

type ResponseDeleteTasksHistory struct {
	Message string `json:"message" binding:"required"`
}
