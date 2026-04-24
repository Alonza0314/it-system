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
