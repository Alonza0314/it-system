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
	Id     int    `json:"id,omitempty"`
	Name   string `json:"name" binding:"required"`
	Link   string `json:"link,omitempty"`
	Script string `json:"script,omitempty"`
	Label  string `json:"label,omitempty"`
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
	Tests    []Test `json:"tests" binding:"required"`
	NFPrList []NfPr `json:"nfPrList" binding:"required"`
}

type Test struct {
	Name   string `json:"name" binding:"required"`
	Script string `json:"script" binding:"required"`
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

type ResponseGetTestLog struct {
	Message string          `json:"message" binding:"required"`
	Log     string          `json:"log,omitempty"`
	Brief   []LogBriefEntry `json:"brief,omitempty"`
}

// LogBriefEntry is one key message picked out of a test log
type LogBriefEntry struct {
	// test: test results and assertions, error: errors, event: key NF events
	Category string `json:"category" binding:"required"`
	Message  string `json:"message" binding:"required"`
	// 1-based line of the first occurrence in the log
	Line  int `json:"line" binding:"required"`
	Count int `json:"count" binding:"required"`
	// Container that printed the line, e.g. nrf, for lines from the compose log
	Source string `json:"source,omitempty"`
	// Timestamp of the first occurrence, for lines printed by an NF
	Time string `json:"time,omitempty"`
	// Values that differ between merged lines, e.g. the NFs that were suspended
	Items []string `json:"items,omitempty"`
}
