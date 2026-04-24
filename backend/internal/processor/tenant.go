package processor

import "backend/model"

func (p *Processor) GetTenants() (*model.ResponseGetTenants, *model.ErrorDetail) {
	// TODO: Replace with real logic to get tenants, e.g., from a database
	response := &model.ResponseGetTenants{
		Message: "Tenants retrieved successfully",
		Tenants: p.tmpTenants,
	}
	return response, nil
}

func (p *Processor) AddTenant(req *model.RequestAddTenant) (*model.ResponseAddTenant, *model.ErrorDetail) {
	// TODO: Replace with real logic to add tenant, e.g., to a database
	for _, tenant := range req.Tenants {
		p.tmpTenants = append(p.tmpTenants, tenant)
	}
	response := &model.ResponseAddTenant{
		Message: "Tenants added successfully",
	}
	return response, nil
}

func (p *Processor) DeleteTenant(req *model.RequestDeleteTenant) (*model.ResponseDeleteTenant, *model.ErrorDetail) {
	// TODO: Replace with real logic to delete tenant, e.g., from a database
	remainingTenants := make([]model.Tenant, 0)
	for _, tenant := range p.tmpTenants {
		notFound := true
		for _, tnt := range req.Tenants {
			if tenant.Username == tnt.Username {
				notFound = false
				break
			}
		}
		if notFound {
			remainingTenants = append(remainingTenants, tenant)
		}
	}
	p.tmpTenants = remainingTenants
	response := &model.ResponseDeleteTenant{
		Message: "Tenants deleted successfully",
	}
	return response, nil
}
