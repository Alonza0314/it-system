package context

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/Alonza0314/it-system/controller/backend/constant"
	"github.com/Alonza0314/it-system/controller/backend/internal/notify"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
)

type taskIdGenerator struct {
	dbContext *bboltDbContext
}

func newTaskIdGenerator(dbCtx *bboltDbContext) *taskIdGenerator {
	return &taskIdGenerator{
		dbContext: dbCtx,
	}
}

func (gen *taskIdGenerator) uint64ToBytes(id uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)

	return b
}

func (gen *taskIdGenerator) bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func (gen *taskIdGenerator) assignId() (uint64, error) {
	exist, err := gen.dbContext.Exists([]byte(constant.BUCKET_TASK_ID), []byte("currentId"))
	if err != nil {
		return 0, err
	}

	if !exist {
		if err := gen.dbContext.Save([]byte(constant.BUCKET_TASK_ID), []byte("currentId"), gen.uint64ToBytes(1)); err != nil {
			return 0, err
		}
		return 1, nil
	}

	currentIdBytes, err := gen.dbContext.Load([]byte(constant.BUCKET_TASK_ID), []byte("currentId"))
	if err != nil {
		return 0, err
	}

	assignedId := gen.bytesToUint64(currentIdBytes) + 1
	if err := gen.dbContext.Save([]byte(constant.BUCKET_TASK_ID), []byte("currentId"), gen.uint64ToBytes(assignedId)); err != nil {
		return 0, err
	}

	return assignedId, nil
}

type pipeline struct {
	name   string
	status string
}

func (p *pipeline) Name() string {
	return p.name
}

func (p *pipeline) Status() string {
	return p.status
}

type nfPr struct {
	nfName string
	pr     int
}

func (p *nfPr) NFName() string {
	return p.nfName
}

func (p *nfPr) PR() int {
	return p.pr
}

type task struct {
	id         uint64
	username   string
	status     string
	createTime int64
	pipelines  []pipeline
	nfPrList   []nfPr
}

func newTask(id uint64, username string, createTime int64, pipelines []pipeline, nfPrList []nfPr) *task {

	return &task{
		id:         id,
		username:   username,
		status:     constant.TASK_STATUS_PENDING,
		createTime: createTime,
		pipelines:  pipelines,
		nfPrList:   nfPrList,
	}
}

func (t *task) ID() uint64 {
	return t.id
}

func (t *task) Username() string {
	return t.username
}

func (t *task) Status() string {
	return t.status
}

func (t *task) CreateTime() int64 {
	return t.createTime
}

func (t *task) Tests() []string {
	tests := make([]string, len(t.pipelines))
	for i, pipeline := range t.pipelines {
		tests[i] = pipeline.name
	}

	return tests
}

func (t *task) Pipelines() []pipeline {
	return t.pipelines
}

func (t *task) NFPrList() []nfPr {
	return t.nfPrList
}

func (t *task) copy() task {
	pipelineCopy := make([]pipeline, len(t.pipelines))
	copy(pipelineCopy, t.pipelines)

	nfPrListCopy := make([]nfPr, len(t.nfPrList))
	copy(nfPrListCopy, t.nfPrList)

	return task{
		id:         t.id,
		username:   t.username,
		status:     t.status,
		createTime: t.createTime,
		pipelines:  pipelineCopy,
		nfPrList:   nfPrListCopy,
	}
}

func (t *task) getLogDir(logPath string) string {
	return fmt.Sprintf("%s/%d", logPath, t.id)
}

func (t *task) toDto() TaskDto {
	pipelines := make([]PipelineDto, len(t.pipelines))
	for i, p := range t.pipelines {
		pipelines[i] = PipelineDto{
			Name:   p.name,
			Status: p.status,
		}
	}

	nfPrList := make([]NfPrDto, len(t.nfPrList))
	for i, np := range t.nfPrList {
		nfPrList[i] = NfPrDto{
			NfName: np.nfName,
			Pr:     np.pr,
		}
	}

	return TaskDto{
		Id:         t.id,
		Username:   t.username,
		Status:     t.status,
		CreateTime: t.createTime,
		Pipelines:  pipelines,
		NfPrList:   nfPrList,
	}
}

func (t *task) revertDto(dto TaskDto) {
	t.id = dto.Id
	t.username = dto.Username
	t.status = dto.Status
	t.createTime = dto.CreateTime
	t.pipelines = make([]pipeline, len(dto.Pipelines))
	for i, p := range dto.Pipelines {
		t.pipelines[i] = pipeline{
			name:   p.Name,
			status: p.Status,
		}
	}
	t.nfPrList = make([]nfPr, len(dto.NfPrList))
	for i, np := range dto.NfPrList {
		t.nfPrList[i] = nfPr{
			nfName: np.NfName,
			pr:     np.Pr,
		}
	}
}

type taskQueue []*task

func newTaskQueue() taskQueue {
	return make([]*task, 0)
}

func (q *taskQueue) copy() []task {
	tasks := make([]task, len(*q))
	for i, t := range *q {
		pipeline := make([]pipeline, len(t.pipelines))
		copy(pipeline, t.pipelines)

		tasks[i] = task{
			id:         t.id,
			username:   t.username,
			status:     t.status,
			createTime: t.createTime,
			pipelines:  pipeline,
		}
	}

	return tasks
}

func (q *taskQueue) Push(t *task) {
	*q = append(*q, t)
}

func (q *taskQueue) Pop() *task {
	if len(*q) == 0 {
		return nil
	}

	t := (*q)[0]
	*q = (*q)[1:]

	return t
}

func (q *taskQueue) RemoveById(id uint64) {
	for i, t := range *q {
		if t.id == id {
			*q = append((*q)[:i], (*q)[i+1:]...)
			return
		}
	}
}

func (q *taskQueue) FindById(id uint64) *task {
	for _, t := range *q {
		if t.id == id {
			return t
		}
	}

	return nil
}

type taskContext struct {
	pendingQueue taskQueue
	ongoingQueue taskQueue
	historyQueue taskQueue

	pendingQueueLock sync.RWMutex
	ongoingQueueLock sync.RWMutex
	historyQueueLock sync.RWMutex

	maxHistoryLength int

	taskIdGenerator *taskIdGenerator

	dbContext *bboltDbContext

	logPath string

	discordEnabled    bool
	discordWebhookURL string
	discordLogger     loggergoModel.LoggerInterface
}

func newTaskContext(logPath string, maxHistoryLength int, dbCtx *bboltDbContext, discordEnabled bool, discordWebhookURL string, discordLogger loggergoModel.LoggerInterface) *taskContext {
	tCtx := &taskContext{
		pendingQueue: newTaskQueue(),
		ongoingQueue: newTaskQueue(),
		historyQueue: newTaskQueue(),

		pendingQueueLock: sync.RWMutex{},
		ongoingQueueLock: sync.RWMutex{},
		historyQueueLock: sync.RWMutex{},

		maxHistoryLength: maxHistoryLength,

		taskIdGenerator: newTaskIdGenerator(dbCtx),

		dbContext: dbCtx,

		logPath: logPath,

		discordEnabled:    discordEnabled,
		discordWebhookURL: discordWebhookURL,
		discordLogger:     discordLogger,
	}

	if err := os.MkdirAll(logPath, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	historyTasksRaw, err := dbCtx.LoadAll([]byte(constant.BUCKET_HISTORY))
	if err != nil {
		panic(fmt.Sprintf("Failed to load history tasks from DB: %v", err))
	}

	for _, raw := range historyTasksRaw {
		var dto TaskDto
		if err := json.Unmarshal(raw, &dto); err != nil {
			panic(fmt.Sprintf("Failed to unmarshal history task with key %s: %v\n", raw, err))
		}
		var t task
		t.revertDto(dto)
		tCtx.historyQueue.Push(&t)
	}

	sort.Slice(tCtx.historyQueue, func(i, j int) bool {
		return tCtx.historyQueue[i].id < tCtx.historyQueue[j].id
	})

	return tCtx
}

func releaseTaskContext(ctx *taskContext) error {
	ctx.historyQueueLock.Lock()
	defer ctx.historyQueueLock.Unlock()

	if err := ctx.dbContext.RemoveAll([]byte(constant.BUCKET_HISTORY)); err != nil {
		return fmt.Errorf("failed to clear history tasks from DB: %v", err)
	}

	for _, t := range ctx.historyQueue {
		raw, err := json.Marshal(t.toDto())
		if err != nil {
			return fmt.Errorf("failed to marshal history task: %v", err)
		}
		if err := ctx.dbContext.Save([]byte(constant.BUCKET_HISTORY), []byte(fmt.Sprintf("%d", t.ID())), raw); err != nil {
			return fmt.Errorf("failed to save history task to DB: %v", err)
		}
	}

	return nil
}

func (ctx *taskContext) getPendingQueue() []task {
	ctx.pendingQueueLock.RLock()
	defer ctx.pendingQueueLock.RUnlock()

	return ctx.pendingQueue.copy()
}

func (ctx *taskContext) getOngoingQueue() []task {
	ctx.ongoingQueueLock.RLock()
	defer ctx.ongoingQueueLock.RUnlock()

	return ctx.ongoingQueue.copy()
}

func (ctx *taskContext) getHistoryQueue() []task {
	ctx.historyQueueLock.RLock()
	defer ctx.historyQueueLock.RUnlock()

	return ctx.historyQueue.copy()
}

func (ctx *taskContext) getTaskById(id uint64) (*task, error) {
	ctx.pendingQueueLock.RLock()
	defer ctx.pendingQueueLock.RUnlock()

	for _, t := range ctx.pendingQueue {
		if t.ID() == id {
			copy := t.copy()
			return &copy, nil
		}
	}

	ctx.ongoingQueueLock.RLock()
	defer ctx.ongoingQueueLock.RUnlock()

	for _, t := range ctx.ongoingQueue {
		if t.ID() == id {
			copy := t.copy()
			return &copy, nil
		}
	}

	ctx.historyQueueLock.RLock()
	defer ctx.historyQueueLock.RUnlock()

	for _, t := range ctx.historyQueue {
		if t.ID() == id {
			copy := t.copy()
			return &copy, nil
		}
	}

	return nil, fmt.Errorf("task with id %d not found", id)
}

func (ctx *taskContext) createTask(username string, createTime int64, tests []pipeline, nfPrList []nfPr) error {
	id, err := ctx.taskIdGenerator.assignId()
	if err != nil {
		return err
	}

	task := newTask(id, username, createTime, tests, nfPrList)

	ctx.pendingQueueLock.Lock()
	defer ctx.pendingQueueLock.Unlock()

	ctx.pendingQueue.Push(task)

	return nil
}

func (ctx *taskContext) getFirstPendingTaskAndMoveToOngoing() (*task, error) {
	ctx.pendingQueueLock.Lock()
	defer ctx.pendingQueueLock.Unlock()

	task := ctx.pendingQueue.Pop()
	if task == nil {
		return nil, nil
	}

	task.status = constant.TASK_STATUS_RUNNING

	ctx.ongoingQueueLock.Lock()
	defer ctx.ongoingQueueLock.Unlock()

	ctx.ongoingQueue.Push(task)

	return task, nil
}

func (ctx *taskContext) cancelTask(id uint64) error {
	ctx.pendingQueueLock.Lock()
	defer ctx.pendingQueueLock.Unlock()

	ctx.pendingQueue.RemoveById(id)

	return nil
}

func (ctx *taskContext) moveOngoingTaskToHistory(id uint64) error {
	task, err := ctx.getTaskById(id)
	if err != nil {
		return err
	}

	ctx.ongoingQueueLock.Lock()
	defer ctx.ongoingQueueLock.Unlock()

	ctx.ongoingQueue.RemoveById(id)
	task.setFinalStatus()

	return ctx.pushHistory(task)
}

func (t *task) setFinalStatus() {
	t.status = constant.TASK_STATUS_SUCCESS

	for _, p := range t.pipelines {
		if p.status == constant.TASK_STATUS_FAILED {
			t.status = constant.TASK_STATUS_FAILED
			return
		}
		if p.status == constant.TASK_STATUS_TIMEOUT {
			t.status = constant.TASK_STATUS_TIMEOUT
			continue
		}
		if p.status != constant.TASK_STATUS_SUCCESS {
			t.status = constant.TASK_STATUS_FAILED
			return
		}
	}
}

func (ctx *taskContext) pushHistory(task *task) error {
	ctx.historyQueueLock.Lock()
	defer ctx.historyQueueLock.Unlock()

	if len(ctx.historyQueue) >= ctx.maxHistoryLength {
		rTask := ctx.historyQueue.Pop()
		if rTask != nil {
			if err := os.RemoveAll(rTask.getLogDir(ctx.logPath)); err != nil {
				return fmt.Errorf("failed to remove log directory for task %d: %v", rTask.ID(), err)
			}
		}
	}

	ctx.historyQueue.Push(task)
	sort.Slice(ctx.historyQueue, func(i, j int) bool {
		return ctx.historyQueue[i].id < ctx.historyQueue[j].id
	})

	if ctx.discordEnabled && ctx.discordWebhookURL != "" {
		pipelines := make([]notify.PipelineResult, len(task.pipelines))
		for i, p := range task.pipelines {
			pipelines[i] = notify.PipelineResult{Name: p.name, Status: p.status}
		}
		go func(taskID uint64, username, status string) {
			userDiscordId, err := ctx.dbContext.Load([]byte(constant.BUCKET_DISCORD_ID), []byte(username))
			if err != nil {
				ctx.discordLogger.Warnf("failed to load discord user ID for username %s: %v", username, err)
				userDiscordId = []byte(username)
			}

			if err := notify.SendTaskNotification(ctx.discordWebhookURL, taskID, username, string(userDiscordId), status, pipelines); err != nil {
				if ctx.discordLogger != nil {
					ctx.discordLogger.Errorf("failed to send discord notification for task %d: %v", taskID, err)
				}
				ctx.discordLogger.Debugf("send discord notification for task: %d", taskID)
			}
		}(task.id, task.username, task.status)
	}

	return nil
}

func (ctx *taskContext) writeLogToFile(id uint64, testName string, success bool, status string, log *string) error {
	ctx.ongoingQueueLock.Lock()
	defer ctx.ongoingQueueLock.Unlock()

	task := ctx.ongoingQueue.FindById(id)
	if task != nil {
		pipelineStatus := normalizePipelineStatus(success, status)
		for i, t := range task.pipelines {
			if t.name == testName {
				task.pipelines[i].status = pipelineStatus
				break
			}
		}
	}

	if task == nil {
		return fmt.Errorf("task with id %d not found", id)
	}

	logDir := task.getLogDir(ctx.logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory for task %d: %v", id, err)
	}

	logFilePath := fmt.Sprintf("%s/%s.log", logDir, testName)
	f, err := os.Create(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to create log file for task %d: %v", id, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(fmt.Sprintf("Failed to close log file for task %d: %v", id, err))
		}
	}()

	if _, err := f.WriteString(*log); err != nil {
		return fmt.Errorf("failed to write log to file for task %d: %v", id, err)
	}

	return nil
}

func normalizePipelineStatus(success bool, status string) string {
	// Prefer the explicit status when it is one of the known values. This lets
	// the runner report timeout separately from failed, without trusting unknown
	// request strings and storing invalid pipeline states.
	switch strings.ToLower(status) {
	case constant.TASK_STATUS_SUCCESS:
		return constant.TASK_STATUS_SUCCESS
	case constant.TASK_STATUS_FAILED:
		return constant.TASK_STATUS_FAILED
	case constant.TASK_STATUS_TIMEOUT:
		return constant.TASK_STATUS_TIMEOUT
	}

	// Fall back to the legacy success flag for older runners or requests that do
	// not provide a valid status field.
	if success {
		return constant.TASK_STATUS_SUCCESS
	}
	return constant.TASK_STATUS_FAILED
}

func (ctx *taskContext) deleteHistory() error {
	ctx.historyQueueLock.Lock()
	defer ctx.historyQueueLock.Unlock()

	for _, t := range ctx.historyQueue {
		if err := os.RemoveAll(t.getLogDir(ctx.logPath)); err != nil {
			return fmt.Errorf("failed to remove log directory for task %d: %v", t.ID(), err)
		}
	}

	ctx.historyQueue = newTaskQueue()

	return nil
}

func (ctx *taskContext) getTestLog(id uint64, testName string) (string, error) {
	logFilePath := fmt.Sprintf("%s/%d/%s.log", ctx.logPath, id, testName)

	logs, err := os.ReadFile(logFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file for task %d and test %s: %v", id, testName, err)
	}

	return string(logs), nil
}
