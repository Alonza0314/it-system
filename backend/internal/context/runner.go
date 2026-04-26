package context

import (
	"backend/constant"
	cctx "context"
	"sync"
	"time"
)

type runnerWithoutLock struct {
	name           string
	ip             string
	status         string
	onGoingtask    uint64
	lastActiveTime int64
}

type runner struct {
	runnerWithoutLock
	rwLock sync.RWMutex
}

func newRunner(name, ip string) *runner {
	return &runner{
		runnerWithoutLock: runnerWithoutLock{
			name:           name,
			ip:             ip,
			status:         constant.RUNNER_STATUS_OFFLINE,
			onGoingtask:    0,
			lastActiveTime: 0,
		},
		rwLock: sync.RWMutex{},
	}
}

func (r *runner) checkStatus(interval time.Duration) string {
	r.rwLock.Lock()
	defer r.rwLock.Unlock()

	if r.status != constant.RUNNER_STATUS_IDLE && time.Now().Unix()-r.lastActiveTime > int64(interval.Seconds()) {
		r.status, r.onGoingtask = constant.RUNNER_STATUS_OFFLINE, 0
	}

	return r.status
}

func (r *runner) copy() runnerWithoutLock {
	r.rwLock.RLock()
	defer r.rwLock.RUnlock()

	return runnerWithoutLock{
		name:        r.name,
		ip:          r.ip,
		status:      r.status,
		onGoingtask: r.onGoingtask,
	}
}

type runnerContext struct {
	runners map[string]*runner
	rwLock  sync.RWMutex

	tctx           cctx.Context
	tctxCancelFunc cctx.CancelFunc

	dbContext *bboltDbContext
}

func newRunnerContext(dbContext *bboltDbContext, runnerCheckTimeInterval time.Duration) *runnerContext {
	r := &runnerContext{
		runners: make(map[string]*runner),
		rwLock:  sync.RWMutex{},

		tctx:           nil,
		tctxCancelFunc: nil,

		dbContext: dbContext,
	}

	runnerMap, err := dbContext.LoadAll([]byte(constant.BUCKET_RUNNER))
	if err != nil {
		panic("Failed to load runner data from DB: " + err.Error())
	}

	for name, ipBytes := range runnerMap {
		r.runners[name] = newRunner(name, string(ipBytes))
	}

	r.tctx, r.tctxCancelFunc = cctx.WithCancel(cctx.Background())
	r.checkRunnerStatus(runnerCheckTimeInterval, r.tctx)

	return r
}

func releaseRunnerContext(ctx *runnerContext) error {
	if ctx.tctxCancelFunc != nil {
		ctx.tctxCancelFunc()
	}

	return nil
}

func (ctx *runnerContext) checkRunnerStatus(interval time.Duration, tctx cctx.Context) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-tctx.Done():
				return
			case <-ticker.C:
				go func() {
					for _, runner := range ctx.runners {
						runner.checkStatus(interval)
					}
				}()
			}
		}
	}()
}

func (ctx *runnerContext) runnerExists(name string) bool {
	ctx.rwLock.RLock()
	defer ctx.rwLock.RUnlock()

	if _, exists := ctx.runners[name]; exists {
		return true
	}

	return false
}
func (ctx *runnerContext) registerRunner(name, ip string) error {
	if err := ctx.dbContext.Save([]byte(constant.BUCKET_RUNNER), []byte(name), []byte(ip)); err != nil {
		return err
	}

	ctx.rwLock.Lock()
	defer ctx.rwLock.Unlock()

	ctx.runners[name] = newRunner(name, ip)

	return nil
}

func (ctx *runnerContext) deleteRunner(name string) error {
	if err := ctx.dbContext.Remove([]byte(constant.BUCKET_RUNNER), []byte(name)); err != nil {
		return err
	}

	ctx.rwLock.Lock()
	defer ctx.rwLock.Unlock()

	delete(ctx.runners, name)

	return nil
}

func (ctx *runnerContext) getRunners() []runnerWithoutLock {
	runnerList := make([]runnerWithoutLock, 0)

	ctx.rwLock.RLock()
	defer ctx.rwLock.RUnlock()

	for _, runner := range ctx.runners {
		runnerList = append(runnerList, runner.copy())
	}

	return runnerList
}
