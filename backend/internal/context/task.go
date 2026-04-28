package context

import (
	"backend/constant"
	"encoding/binary"
	"fmt"
	"sync"
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

type taskContext struct {
	pendingQueue taskQueue
	ongoingQueue taskQueue

	pendingQueueLock sync.RWMutex
	ongoingQueueLock sync.RWMutex

	taskIdGenerator *taskIdGenerator
}

func newTaskContext(dbCtx *bboltDbContext) *taskContext {
	tCtx := &taskContext{
		pendingQueue: newTaskQueue(),
		ongoingQueue: newTaskQueue(),

		taskIdGenerator: newTaskIdGenerator(dbCtx),
	}

	return tCtx
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

func (ctx *taskContext) getTaskById(id uint64) (*task, error) {
	ctx.pendingQueueLock.RLock()
	defer ctx.pendingQueueLock.RUnlock()

	for _, t := range ctx.pendingQueue {
		if t.ID() == id {
			copy := t.copy()
			return &copy, nil
		}
	}

	for _, t := range ctx.ongoingQueue {
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
