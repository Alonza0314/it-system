package constant

import (
	"net/http"
	"time"
)

// log
const (
	CFG_LOG  = "CFG"
	RUN_LOG  = "RUN"
	HTTP_LOG = "HTTP"
	TASK_LOG = "TASK"
)

type HttpSenderMessageType string

// http sender message type
const (
	MSG_TYPE_HEARTBEAT   HttpSenderMessageType = "heartbeat"
	MSG_TYPE_TEST_OUTPUT HttpSenderMessageType = "test_output"
)

// controller api
const (
	HTTP_API = "http://%s:%d"

	HTTP_RETRY_TIMES = 3

	AUTHENTICATION_HEADER_KEY   = "Authorization"
	AUTHENTICATION_HEADER_VALUE = "Bearer %s"
	USER_HEADER_KEY             = "user"

	API_RUNNER_HEARTBEAT_ACTION   = http.MethodPost
	API_RUNNER_TEST_OUTPUT_ACTION = http.MethodPost

	API_RUNNER_HEARTBEAT   = HTTP_API + "/api/run/runner/heartbeat"
	API_RUNNER_TEST_OUTPUT = HTTP_API + "/api/run/runner/test-output"
)

type RunnerStatus string

// runner status
const (
	STATUS_IDLE    RunnerStatus = "idle"
	STATUS_RUNNING RunnerStatus = "running"
)

// task
const (
	FREE5GC_REPO     = "free5gc"
	FREE5GC_REPO_URL = "https://github.com/free5gc/free5gc.git"

	CLONE_CMD_TIMEOUT = 30 * time.Second
	FETCH_CMD_TIMEOUT = 30 * time.Second
	TEST_CMD_TIMEOUT  = 5 * time.Minute
)
