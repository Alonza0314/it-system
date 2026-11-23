package processor

import "github.com/Alonza0314/it-system/controller/backend/model"

// CreateTaskUnchecked stores a task without the checks of SubmitTask, like a
// task that was submitted before a check was added.
func (p *Processor) CreateTaskUnchecked(username string, createTime int64, tests []model.Test, nfPrList []model.NfPr) error {
	return p.itContext.CreateTask(username, createTime, tests, nfPrList)
}
