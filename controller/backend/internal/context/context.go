package context

import (
	"time"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/model"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
)

type ItContext struct {
	githubContext  *githubContext
	bboltDbContext *bboltDbContext
	taskContext    *taskContext
	runnerContext  *runnerContext
}

func NewItContext(dbPath, logPath string, maxHistoryLength int, runnerCheckTimeInterval time.Duration, discordEnabled bool, discordWebhookURL string, dcrLog loggergoModel.LoggerInterface) *ItContext {
	dbContext := newBboltDbContext(dbPath)

	return &ItContext{
		githubContext:  newGithubContext(),
		bboltDbContext: dbContext,
		taskContext:    newTaskContext(logPath, maxHistoryLength, dbContext, discordEnabled, discordWebhookURL, dcrLog),
		runnerContext:  newRunnerContext(dbContext, runnerCheckTimeInterval),
	}
}

func ReleaseItContext(ctx *ItContext) error {
	if err := releaseTaskContext(ctx.taskContext); err != nil {
		return err
	}

	if err := releaseBboltDbContext(ctx.bboltDbContext); err != nil {
		return err
	}

	if err := releaseRunnerContext(ctx.runnerContext); err != nil {
		return err
	}

	return nil
}

func (ctx *ItContext) GetPrList(nf string) ([]pr, error) {
	return ctx.githubContext.getPrList(nf)
}

func (ctx *ItContext) SaveToDb(bucket, key, value string) error {
	return ctx.bboltDbContext.Save([]byte(bucket), []byte(key), []byte(value))
}

func (ctx *ItContext) LoadFromDb(bucket, key string) (string, error) {
	value, err := ctx.bboltDbContext.Load([]byte(bucket), []byte(key))
	if err != nil {
		return "", err
	}

	return string(value), nil
}

func (ctx *ItContext) LoadAllFromDb(bucket string) (map[string]string, error) {
	rawResult, err := ctx.bboltDbContext.LoadAll([]byte(bucket))
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for k, v := range rawResult {
		result[k] = string(v)
	}

	return result, nil
}

func (ctx *ItContext) UpdateDb(bucket, key, value string) error {
	return ctx.bboltDbContext.Update([]byte(bucket), []byte(key), []byte(value))
}

func (ctx *ItContext) RemoveFromDb(bucket, key string) error {
	return ctx.bboltDbContext.Remove([]byte(bucket), []byte(key))
}

func (ctx *ItContext) RemoveAllFromDb(bucket string) error {
	return ctx.bboltDbContext.RemoveAll([]byte(bucket))
}

func (ctx *ItContext) ExistsInDb(bucket, key string) (bool, error) {
	return ctx.bboltDbContext.Exists([]byte(bucket), []byte(key))
}

func (ctx *ItContext) GetPendingTasks() []model.TaskSimple {
	return convertTaskToResponseTask(ctx.taskContext.getPendingQueue())
}

func (ctx *ItContext) GetOngoingTasks() []model.TaskSimple {
	return convertTaskToResponseTask(ctx.taskContext.getOngoingQueue())
}

func (ctx *ItContext) GetHistoryTasks() []model.TaskSimple {
	return convertTaskToResponseTask(ctx.taskContext.getHistoryQueue())
}

func (ctx *ItContext) GetTask(id uint64) (*task, error) {
	return ctx.taskContext.getTaskById(id)
}

func (ctx *ItContext) CreateTask(username string, createTime int64, tests []string, nfPrList []model.NfPr) error {
	return ctx.taskContext.createTask(username, createTime, convertTestsToPipelines(tests), convertNfPrListToNfPr(nfPrList))
}

func (ctx *ItContext) GetFirstPendingTaskAndMoveToOngoing() (*task, error) {
	return ctx.taskContext.getFirstPendingTaskAndMoveToOngoing()
}

func (ctx *ItContext) CancelTask(id uint64) error {
	return ctx.taskContext.cancelTask(id)
}

func (ctx *ItContext) TtestOutputEnd(id uint64) error {
	return ctx.taskContext.moveOngoingTaskToHistory(id)
}

func (ctx *ItContext) TtestOutputTransfer(id uint64, testName string, success bool, status string, log *string) error {
	return ctx.taskContext.writeLogToFile(id, testName, success, status, log)
}

func (ctx *ItContext) DeleteHistory() error {
	return ctx.taskContext.deleteHistory()
}

func (ctx *ItContext) GetTestLog(id uint64, testName string) (string, error) {
	return ctx.taskContext.getTestLog(id, testName)
}

func (ctx *ItContext) RunnerExists(name string) bool {
	return ctx.runnerContext.runnerExists(name)
}

func (ctx *ItContext) RegisterRunner(name, ip string) error {
	return ctx.runnerContext.registerRunner(name, ip)
}

func (ctx *ItContext) DeleteRunner(name string) error {
	return ctx.runnerContext.deleteRunner(name)
}

func (ctx *ItContext) GetRunners() []model.Runner {
	runners := ctx.runnerContext.getRunners()

	responseRunners := make([]model.Runner, len(runners))
	for i, r := range runners {
		responseRunners[i] = model.Runner{
			Name:        r.name,
			IP:          r.ip,
			Status:      r.status,
			OnGoingTask: r.onGoingtask,
		}
	}

	return responseRunners
}

func (ctx *ItContext) HeartbeatWithoutTask(name string) error {
	return ctx.runnerContext.heartbeatWithoutTask(name)
}

func (ctx *ItContext) HeartbeatWithTask(name string, taskId uint64) error {
	return ctx.runnerContext.heartbeatWithTask(name, taskId)
}

func convertTaskToResponseTask(tasks []task) []model.TaskSimple {
	simpleTasks := make([]model.TaskSimple, len(tasks))
	for i, t := range tasks {
		simpleTasks[i] = model.TaskSimple{
			Id:         t.ID(),
			Username:   t.Username(),
			CreateTime: t.CreateTime(),
		}
	}

	return simpleTasks
}

func convertTestsToPipelines(tests []string) []pipeline {
	pipelines := make([]pipeline, len(tests))
	pipelines = append(pipelines, []pipeline{
		{
			name:   constant.TESTCASE_PREPARE_FREE5GC,
			status: constant.TASK_STATUS_PENDING,
		},
		{
			name:   constant.TESTCASE_FETCH_PRS,
			status: constant.TASK_STATUS_PENDING,
		},
		{
			name:   constant.TESTCASE_MAKE_NF,
			status: constant.TASK_STATUS_PENDING,
		},
	}...)

	for i, test := range tests {
		pipelines[i] = pipeline{
			name:   test,
			status: constant.TASK_STATUS_PENDING,
		}
	}

	pipelines = append(pipelines, pipeline{
		name:   constant.TESTCASE_CLEANUP,
		status: constant.TASK_STATUS_PENDING,
	})
	return pipelines
}

func convertNfPrListToNfPr(nfPrList []model.NfPr) []nfPr {
	nfPrs := make([]nfPr, len(nfPrList))
	for i, np := range nfPrList {
		nfPrs[i] = nfPr{
			nfName: np.NfName,
			pr:     np.PR,
		}
	}

	return nfPrs
}
