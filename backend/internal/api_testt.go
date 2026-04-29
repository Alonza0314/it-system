package internal

import (
	"backend/model"
	"net/http"
	"strconv"

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
		{
			Name:        "Get tasks",
			Method:      http.MethodGet,
			Pattern:     "/tasks",
			HandlerFunc: b.handleGetTasks,
		},
		{
			Name:        "Get task",
			Method:      http.MethodGet,
			Pattern:     "/task",
			HandlerFunc: b.handleGetTask,
		},
		{
			Name:        "Submit task",
			Method:      http.MethodPost,
			Pattern:     "/task",
			HandlerFunc: b.handleSubmitTask,
		},
		{
			Name:        "Cancel task",
			Method:      http.MethodDelete,
			Pattern:     "/task",
			HandlerFunc: b.handleCancelTask,
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
		{
			Name:        "Delete tasks history",
			Method:      http.MethodDelete,
			Pattern:     "/history",
			HandlerFunc: b.handleDeleteTasksHistory,
		},
	}
}

func (b *backend) handleGetTestcases(c *gin.Context) {
	b.TestLog.Infof("Get testcases request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

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
	b.TestLog.Infof("Add testcases request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

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
	b.TestLog.Infof("Delete testcases request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

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

func (b *backend) handleGetTasks(c *gin.Context) {
	b.TestLog.Infof("Get tasks request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	response, errDetail := b.Processor.GetTasks()
	if errDetail != nil {
		b.TestLog.Errorf("Get tasks failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseGetTasks{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Get tasks successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleGetTask(c *gin.Context) {
	b.TestLog.Infof("Get task request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	id := c.Query("id")
	if id == "" {
		b.TestLog.Warnf("Get task request missing id from %s", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Missing id parameter",
		})
		return
	}

	uint64Id, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		b.TestLog.Warnf("Invalid task id %s from %s", id, c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid id parameter",
		})
		return
	}

	response, errDetail := b.Processor.GetTask(uint64Id)
	if errDetail != nil {
		b.TestLog.Errorf("Get task failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseGetTask{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Get task successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleSubmitTask(c *gin.Context) {
	b.TestLog.Infof("Submit task request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	var req model.RequestSubmitTask
	if err := c.ShouldBindJSON(&req); err != nil {
		b.TestLog.Warnf("Invalid submit task request from %s: %v\n", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseSubmitTask{
			Message: "Invalid request",
		})
		return
	}

	response, errDetail := b.Processor.SubmitTask(&req, c.GetHeader("user"))
	if errDetail != nil {
		b.TestLog.Warnf("Submit task failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseSubmitTask{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Submit task successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleCancelTask(c *gin.Context) {
	b.TestLog.Infof("Cancel task request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	id := c.Query("id")
	if id == "" {
		b.TestLog.Warnf("Cancel task request missing id from %s", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Missing id parameter",
		})
		return
	}

	uint64Id, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		b.TestLog.Warnf("Invalid task id %s from %s", id, c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid id parameter",
		})
		return
	}

	response, errDetail := b.Processor.CancelTask(uint64Id)
	if errDetail != nil {
		b.TestLog.Warnf("Cancel task failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, gin.H{
			"message": errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Cancel task successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleDeleteTasksHistory(c *gin.Context) {
	b.TestLog.Infof("Delete tasks history request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	response, errDetail := b.Processor.DeleteTasksHistory()
	if errDetail != nil {
		b.TestLog.Warnf("Delete tasks history failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseDeleteTasksHistory{
			Message: errDetail.Detail,
		})
		return
	}

	b.TestLog.Infof("Delete tasks history successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}
