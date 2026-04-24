package internal

import (
	"backend/model"
	"net/http"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (b *backend) getAdminTenantRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Get tenants",
			Method:      http.MethodGet,
			Pattern:     "",
			HandlerFunc: b.handleGetTenants,
		},
		{
			Name:        "Add tenant",
			Method:      http.MethodPost,
			Pattern:     "",
			HandlerFunc: b.handleAddTenant,
		},
		{
			Name:        "Delete tenant",
			Method:      http.MethodDelete,
			Pattern:     "",
			HandlerFunc: b.handleDeleteTenant,
		},
	}
}

func (b *backend) handleGetTenants(c *gin.Context) {
	b.TntLog.Infof("Get tenants request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	response, errDetail := b.Processor.GetTenants()
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseGetTenants{
			Message: errDetail.Detail,
		})
		return
	}

	b.TntLog.Infof("Get tenants successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleAddTenant(c *gin.Context) {
	b.TntLog.Infof("Add tenant request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	var req model.RequestAddTenant
	if err := c.ShouldBindJSON(&req); err != nil {
		b.TntLog.Warnf("Invalid add tenant request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseAddTenant{
			Message: "Invalid request body",
		})
		return
	}

	response, errDetail := b.Processor.AddTenant(&req)
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseAddTenant{
			Message: errDetail.Detail,
		})
		return
	}

	b.TntLog.Infof("Add tenant successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleDeleteTenant(c *gin.Context) {
	b.TntLog.Infof("Delete tenant request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	var req model.RequestDeleteTenant
	if err := c.ShouldBindJSON(&req); err != nil {
		b.TntLog.Warnf("Invalid delete tenant request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseDeleteTenant{
			Message: "Invalid request body",
		})
		return
	}

	response, errDetail := b.Processor.DeleteTenant(&req)
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseDeleteTenant{
			Message: errDetail.Detail,
		})
		return
	}

	b.TntLog.Infof("Delete tenant successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}
