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
