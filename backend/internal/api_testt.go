package internal

import (
	"backend/model"
	"net/http"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (b *backend) getTestRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Get testcases",
			Method:      http.MethodGet,
			Pattern:     "/testcase",
			HandlerFunc: b.handleGetTestcases,
		},
	}
}

func (b *backend) getAdminTestRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Add testcases",
			Method:      http.MethodPost,
			Pattern:     "/testcase",
			HandlerFunc: b.handleAddTestcases,
		},
		{
			Name:        "Delete testcases",
			Method:      http.MethodDelete,
			Pattern:     "/testcase",
			HandlerFunc: b.handleDeleteTestcases,
		},
	}
}

func (b *backend) handleGetTestcases(c *gin.Context) {
	b.TestLog.Infof("Get testcases request from %s", c.ClientIP())

	response, errDetail := b.Processor.GetTestcases()
	if errDetail != nil {
		c.JSON(errDetail.HttpStatus, model.ResponseGetTestcases{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Get testcases successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleAddTestcases(c *gin.Context) {
	b.TestLog.Infof("Add testcases request from %s", c.ClientIP())

	var req model.RequestAddTestcases
	if err := c.ShouldBindJSON(&req); err != nil {
		b.TestLog.Warnf("Invalid add testcases request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseAddTestcases{
			Message: "Invalid request",
		})
		return
	}

	response, errDetail := b.Processor.AddTestcases(&req)
	if errDetail != nil {
		b.TestLog.Warnf("Add testcases failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseAddTestcases{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Add testcases successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleDeleteTestcases(c *gin.Context) {
	b.TestLog.Infof("Delete testcases request from %s", c.ClientIP())

	var req model.RequestDeleteTestcases
	if err := c.ShouldBindJSON(&req); err != nil {
		b.TestLog.Warnf("Invalid delete testcases request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseDeleteTestcases{
			Message: "Invalid request",
		})
		return
	}

	response, errDetail := b.Processor.DeleteTestcases(&req)
	if errDetail != nil {
		b.TestLog.Warnf("Delete testcases failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseDeleteTestcases{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Delete testcases successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)

}
