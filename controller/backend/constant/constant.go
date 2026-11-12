package constant

// log
const (
	CFG_LOG  = "CFG"
	API_LOG  = "API"
	ACC_LOG  = "ACC"
	BCK_LOG  = "BCK"
	PROC_LOG = "PROC"
	TEST_LOG = "TEST"
	TNT_LOG  = "TNT"
	GIT_LOG  = "GIT"
	RUN_LOG  = "RUN"
	DCR_LOG  = "DCR"
	CTX_LOG  = "CTX"
	TERM_LOG = "TERM"
)

// user level
const (
	USER_LEVEL_CLAIM_TAG = "user_level"
	USER_LEVEL_ADMIN     = "admin"
	USER_LEVEL_DEFAULT   = "default"
	USER_LEVEL_RUNNER    = "runner"
)

// github
const (
	GITHUB_FREE5GC_BASE_API_URL = "https://api.github.com/repos/free5gc/%s/pulls"
	GITHUB_PR_STATE_MERGED      = "merged"
	FREE5GC                     = "free5gc"
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
	WEBCONSOLE                  = "webconsole"
)

var NF_LIST = []string{FREE5GC, AMF, AUSF, BSF, CHF, N3IWF, NEF, NRF, NSSF, PCF, SMF, TNGF, UDM, UDR, UPF, WEBCONSOLE}

// db
const (
	BUCKET_TENANT     = "tenant"
	BUCKET_DISCORD_ID = "discord_id"
	BUCKET_TESTCASE   = "testcase"
	BUCKET_LABEL      = "label"
	BUCKET_TASK_ID    = "taskId"
	BUCKET_RUNNER     = "runner"
	BUCKET_HISTORY    = "history"
)

// BUCKET_RUNNER_REGISTRATION maps a runner name to the id of its current
// registration (see RUNNER_REGISTRATION_CLAIM_TAG).
const BUCKET_RUNNER_REGISTRATION = "runner_registration"

// task status
const (
	TASK_STATUS_PENDING  = "pending"
	TASK_STATUS_RUNNING  = "running"
	TASK_STATUS_SUCCESS  = "success"
	TASK_STATUS_FAILED   = "failed"
	TASK_STATUS_TIMEOUT  = "timeout"
	TASK_STATUS_CANCELED = "canceled"
)

// TASK_CANCELED_HEADER is set on the reply to a runner's test output when the
// task is no longer running on the controller: a user canceled it, or it was
// lost when the controller restarted. The value is the task id. The reply is
// still 204, so runners that do not know the header keep working as before.
const TASK_CANCELED_HEADER = "X-Task-Canceled"

// runner name
const (
	RUNNER_JWT_SUBJECT_TAG = "runner"
	// claim of a runner token holding the id of the registration it was issued
	// for; deleting or re-registering the runner invalidates it for the terminal
	RUNNER_REGISTRATION_CLAIM_TAG = "rid"
)

// gin context key of the JWT claims, set by the admin and runner middleware
const (
	CLAIMS_CONTEXT_KEY = "claims"
)

// defaults shipped in config-controller.yaml; the terminal refuses to start with them
const (
	DEFAULT_BACKEND_PASSWORD  = "0000"
	DEFAULT_JWT_SECRET        = "free5gc-it-system"
	DEFAULT_RUNNER_JWT_SECRET = "free5gc-it-runner"
)

// runner status
const (
	RUNNER_STATUS_OFFLINE = "offline"
	RUNNER_STATUS_IDLE    = "idle"
	RUNNER_STATUS_RUNNING = "running"
)

// basic test case
const (
	TESTCASE_PREPARE_FREE5GC = "prepare_free5gc"
	TESTCASE_FETCH_PRS       = "fetch_prs"
	TESTCASE_MAKE_NF         = "make_nf"
	TESTCASE_CLEANUP         = "cleanup"
)

// discord notification
const (
	MAX_THREAD_PR_SUMMARY_LENGTH = 40
)
