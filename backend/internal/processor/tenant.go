package processor

import (
	"backend/constant"
	"backend/model"
	"fmt"
	"net/http"
)

func (p *Processor) GetTenants() (*model.ResponseGetTenants, *model.ErrorDetail) {
	tenantMap, err := p.itContext.LoadAllFromDb(constant.BUCKET_TENANT)
	if err != nil {
		return nil, &model.ErrorDetail{
			HttpStatus: http.StatusInternalServerError,
			Detail:     fmt.Sprintf("Failed to load tenants from database: %v", err),
		}
	}

	tenants := make([]model.Tenant, 0, len(tenantMap))
	for username, role := range tenantMap {
		tenants = append(tenants, model.Tenant{
			Username: username,
			Role:     role,
		})
	}

	response := &model.ResponseGetTenants{
		Message: "Tenants retrieved successfully",
		Tenants: tenants,
	}
	return response, nil
}

func (p *Processor) AddTenant(req *model.RequestAddTenant) (*model.ResponseAddTenant, *model.ErrorDetail) {
	for _, tenant := range req.Tenants {
		exists, err := p.itContext.ExistsInDb(constant.BUCKET_TENANT, tenant.Username)
		if err != nil {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if tenant %s exists in database: %v", tenant.Username, err),
			}
		}
		if exists {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusConflict,
				Detail:     fmt.Sprintf("Tenant %s already exists", tenant.Username),
			}
		}
	}

	for _, tenant := range req.Tenants {
		if err := p.itContext.SaveToDb(constant.BUCKET_TENANT, tenant.Username, tenant.Role); err != nil {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to save tenant %s to database: %v", tenant.Username, err),
			}
		}
	}

	response := &model.ResponseAddTenant{
		Message: "Tenants added successfully",
	}
	return response, nil
}

func (p *Processor) DeleteTenant(req *model.RequestDeleteTenant) (*model.ResponseDeleteTenant, *model.ErrorDetail) {
	for _, tenant := range req.Tenants {
		exists, err := p.itContext.ExistsInDb(constant.BUCKET_TENANT, tenant.Username)
		if err != nil {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to check if tenant %s exists in database: %v", tenant.Username, err),
			}
		}
		if !exists {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusNotFound,
				Detail:     fmt.Sprintf("Tenant %s not found", tenant.Username),
			}
		}
	}

	for _, tenant := range req.Tenants {
		if err := p.itContext.RemoveFromDb(constant.BUCKET_TENANT, tenant.Username); err != nil {
			return nil, &model.ErrorDetail{
				HttpStatus: http.StatusInternalServerError,
				Detail:     fmt.Sprintf("Failed to remove tenant %s from database: %v", tenant.Username, err),
			}
		}
	}

	response := &model.ResponseDeleteTenant{
		Message: "Tenants deleted successfully",
	}
	return response, nil
}
