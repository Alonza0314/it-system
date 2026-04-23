package model

type RequestLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ResponseLogin struct {
	Message string `json:"message"`
	Token   string `json:"token,omitempty"`
}
