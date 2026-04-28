package constant

// log
const (
	CFG_LOG  = "CFG"
	ACC_LOG  = "ACC"
	BCK_LOG  = "BCK"
	PROC_LOG = "PROC"
	TEST_LOG = "TEST"
	TNT_LOG  = "TNT"
	GIT_LOG  = "GIT"
	RUN_LOG  = "RUN"
)

// user level
const (
	USER_LEVEL_CLAIM_TAG = "user_level"
	USER_LEVEL_ADMIN     = "admin"
	USER_LEVEL_DEFAULT   = "default"
)

// github
const (
	GITHUB_FREE5GC_BASE_API_URL = "https://api.github.com/repos/free5gc/%s/pulls"
	AMF                         = "amf"
	AUSF                        = "ausf"
	BSF                         = "bsf"
	CHF                         = "chf"
	N3IWF                       = "n3iwf"
	NEF                         = "nef"
	NRF                         = "nrf"
	NSSF                        = "nssf"
	PCF                         = "pcf"
	SMF                         = "smf"
	TNGF                        = "tngf"
	UDM                         = "udm"
	UDR                         = "udr"
	UPF                         = "upf"
	GO_UPF                      = "go-upf"
)

var NF_LIST = []string{AMF, AUSF, BSF, CHF, N3IWF, NEF, NRF, NSSF, PCF, SMF, TNGF, UDM, UDR, UPF}

// db
const (
	BUCKET_TENANT   = "tenant"
	BUCKET_TESTCASE = "testcase"
	BUCKET_TASK_ID  = "taskId"
	BUCKET_RUNNER   = "runner"
)

// task status
const (
	TASK_STATUS_PENDING  = "pending"
	TASK_STATUS_RUNNING  = "running"
	TASK_STATUS_SUCCESS  = "success"
	TASK_STATUS_FAILED   = "failed"
	TASK_STATUS_CANCELED = "canceled"
)

// runner name
const (
	RUNNER_JWT_SUBJECT_TAG = "runner"
)

// runner status
const (
	RUNNER_STATUS_OFFLINE = "offline"
	RUNNER_STATUS_IDLE    = "idle"
	RUNNER_STATUS_RUNNING = "running"
)
