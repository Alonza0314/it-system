package internal

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"

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
		{
			Name:        "Get Github PR",
			Method:      http.MethodGet,
			Pattern:     "/pr",
			HandlerFunc: b.handleGetGithubPR,
		},
	}
}

func (b *backend) handleGetGithubPRs(c *gin.Context) {
	b.GitLog.Infof("Get Github PRs request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	nf := c.Query("nf")
	if nf == "" {
		c.JSON(http.StatusBadRequest, model.ResponseGetGithubPRs{
			Message: "NF parameter is required",
		})
		return
	}

	if exist := slices.Contains(constant.NF_LIST, nf); !exist {
		c.JSON(http.StatusBadRequest, model.ResponseGetGithubPRs{
			Message: "Invalid NF parameter, must be one of: " + strings.Join(constant.NF_LIST, ", "),
		})
		return
	}

	if nf == constant.UPF {
		nf = constant.GO_UPF
	}

	response, errDetail := b.Processor.GetGithubPRs(nf)
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseGetGithubPRs{
			Message: errDetail.Detail,
		})
		return
	}

	b.GitLog.Infof("Get Github PRs successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleGetGithubPR(c *gin.Context) {
	b.GitLog.Infof("Get Github PR request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	nf := c.Query("nf")
	if nf == "" {
		c.JSON(http.StatusBadRequest, model.ResponseGetGithubPR{
			Message: "NF parameter is required",
		})
		return
	}

	if exist := slices.Contains(constant.NF_LIST, nf); !exist {
		c.JSON(http.StatusBadRequest, model.ResponseGetGithubPR{
			Message: "Invalid NF parameter, must be one of: " + strings.Join(constant.NF_LIST, ", "),
		})
		return
	}

	number, err := strconv.Atoi(c.Query("number"))
	if err != nil || number <= 0 {
		c.JSON(http.StatusBadRequest, model.ResponseGetGithubPR{
			Message: "Invalid number parameter, must be a positive integer",
		})
		return
	}

	if nf == constant.UPF {
		nf = constant.GO_UPF
	}

	response, errDetail := b.Processor.GetGithubPR(nf, number)
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseGetGithubPR{
			Message: errDetail.Detail,
		})
		return
	}

	b.GitLog.Infof("Get Github PR successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}
