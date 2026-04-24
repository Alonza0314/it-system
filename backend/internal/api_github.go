package internal

import (
	"backend/model"
	"net/http"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (b *backend) getGithubRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Get Github PRs",
			Method:      http.MethodGet,
			Pattern:     "",
			HandlerFunc: b.handleGetGithubPRs,
		},
	}
}

func (b *backend) handleGetGithubPRs(c *gin.Context) {
	b.GitLog.Infof("Get Github PRs request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	response, errDetail := b.Processor.GetGithubPRs()
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseGetGithubPRs{
			Message: errDetail.Detail,
		})
		return
	}

	b.GitLog.Infof("Get Github PRs successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}
