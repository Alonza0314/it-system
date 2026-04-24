package model

type ResponseGetTenants struct {
	Message string   `json:"message" binding:"required"`
	Tenants []Tenant `json:"tenants,omitempty"`
}

type RequestAddTenant struct {
	Tenants []Tenant `json:"tenants" binding:"required"`
}

type ResponseAddTenant struct {
	Message string `json:"message" binding:"required"`
}

type RequestDeleteTenant struct {
	Tenants []Tenant `json:"tenants" binding:"required"`
}

type ResponseDeleteTenant struct {
	Message string `json:"message" binding:"required"`
}

type Tenant struct {
	Username string `json:"username" binding:"required"`
	Role     string `json:"role" binding:"required"`
}
