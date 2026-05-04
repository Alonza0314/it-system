package internal

import (
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (b *backend) getAccountRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Login",
			Method:      http.MethodPost,
			Pattern:     "/login",
			HandlerFunc: b.handleLogin,
		},
		{
			Name:        "Logout",
			Method:      http.MethodPost,
			Pattern:     "/logout",
			HandlerFunc: b.handleLogout,
		},
	}
}

func (b *backend) handleLogin(c *gin.Context) {
	b.AccLog.Infof("Login attempt from %s", c.ClientIP())

	var req model.RequestLogin
	if err := c.ShouldBindJSON(&req); err != nil {
		b.AccLog.Warnf("Invalid login request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseLogin{
			Message: "Invalid request",
		})
		return
	}

	response, errDetail := b.Processor.Login(&req)
	if errDetail != nil {
		b.AccLog.Warnf("Login failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseLogin{
			Message: errDetail.Detail,
		})
		return
	}

	b.AccLog.Infof("Login successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleLogout(c *gin.Context) {
	b.AccLog.Infof("Logout successful from %s", c.ClientIP())

	c.Status(http.StatusNoContent)
}
