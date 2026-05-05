package internal

import (
	"net/http"

	"github.com/Alonza0314/it-system/controller/backend/model"

	"github.com/free-ran-ue/util"
	"github.com/gin-gonic/gin"
)

func (b *backend) getRunnerRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Get Runners",
			Method:      http.MethodGet,
			Pattern:     "",
			HandlerFunc: b.handleGetRunners,
		},
	}
}

func (b *backend) getAdminRunnerRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Register Runner",
			Method:      http.MethodPost,
			Pattern:     "",
			HandlerFunc: b.handleRegisterRunner,
		},
		{
			Name:        "Delete Runner",
			Method:      http.MethodDelete,
			Pattern:     "",
			HandlerFunc: b.handleDeleteRunner,
		},
	}
}

func (b *backend) getRunRunnerRoutes() util.Routes {
	return util.Routes{
		{
			Name:        "Runner Heartbeat",
			Method:      http.MethodPost,
			Pattern:     "/heartbeat",
			HandlerFunc: b.handleRunnerHeartbeat,
		},
		{
			Name:        "Test Output",
			Method:      http.MethodPost,
			Pattern:     "/test-output",
			HandlerFunc: b.handleTestOutput,
		},
	}
}

func (b *backend) handleRegisterRunner(c *gin.Context) {
	b.RunLog.Infof("Register Runner request from %s", c.ClientIP())

	var req model.RequestRegisterRunner
	if err := c.ShouldBindJSON(&req); err != nil {
		b.RunLog.Warnf("Invalid request body from %s: %v", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseRegisterRunner{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	response, errDetail := b.Processor.RegisterRunner(&req)
	if errDetail != nil {
		b.RunLog.Errorf("Register Runner failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseRegisterRunner{
			Message: errDetail.Detail,
		})
		return
	}

	b.RunLog.Infof("Register Runner successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleDeleteRunner(c *gin.Context) {
	b.RunLog.Infof("Delete Runner request from %s", c.ClientIP())

	runnerName := c.Query("name")
	if runnerName == "" {
		b.RunLog.Warnf("Runner name is empty for delete request from %s", c.ClientIP())
		c.JSON(http.StatusBadRequest, model.ResponseDeleteRunner{
			Message: "Runner name is required",
		})
		return
	}

	response, errDetail := b.Processor.DeleteRunner(runnerName)
	if errDetail != nil {
		b.RunLog.Errorf("Delete Runner failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseDeleteRunner{
			Message: errDetail.Detail,
		})
		return
	}

	b.RunLog.Infof("Delete Runner successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleGetRunners(c *gin.Context) {
	b.RunLog.Infof("Get Runners request from %s, user: %s", c.ClientIP(), c.GetHeader("user"))

	response, errDetail := b.Processor.GetRunners()
	if errDetail != nil {
		b.RunLog.Errorf("Get Runners failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseGetRunners{
			Message: errDetail.Detail,
			Runners: nil,
		})
		return
	}

	b.RunLog.Infof("Get Runners successful for %s", c.ClientIP())
	c.JSON(http.StatusOK, response)
}

func (b *backend) handleRunnerHeartbeat(c *gin.Context) {
	b.RunLog.Infof("Runner Heartbeat request from %s, runner: %s", c.ClientIP(), c.GetHeader("user"))

	var req model.RequestRunnerHeartbeat
	if err := c.ShouldBindJSON(&req); err != nil {
		b.RunLog.Warnf("Invalid request body from %s: %v", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseRunnerHeartbeat{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	response, errDetail := b.Processor.RunnerHeartbeat(&req, c.GetHeader("user"))
	if errDetail != nil {
		b.RunLog.Errorf("Runner Heartbeat failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseRunnerHeartbeat{
			Message: errDetail.Detail,
		})
		return
	}

	b.RunLog.Infof("Runner Heartbeat successful for %s", c.ClientIP())
	if response == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (b *backend) handleTestOutput(c *gin.Context) {
	b.RunLog.Infof("Test Output request from %s, runner: %s", c.ClientIP(), c.GetHeader("user"))

	var req model.RequestTestOutput
	if err := c.ShouldBindJSON(&req); err != nil {
		b.RunLog.Warnf("Invalid request body from %s: %v", c.ClientIP(), err)
		c.JSON(http.StatusBadRequest, model.ResponseRunnerTestOutput{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if errDetail := b.Processor.TtestOutput(&req, c.GetHeader("user")); errDetail != nil {
		b.RunLog.Errorf("Test Output failed for %s: %s", c.ClientIP(), errDetail.Detail)
		c.JSON(errDetail.HttpStatus, model.ResponseRunnerTestOutput{
			Message: errDetail.Detail,
		})
		return
	}

	b.RunLog.Infof("Test Output successful for %s", c.ClientIP())
	c.Status(http.StatusNoContent)
}
