package context

import "backend/model"

type ItContext struct {
	testcaseContext *testcaseContext
	githubContext   *githubContext
	bboltDbContext  *bboltDbContext
}

func NewItContext(dbPath string) *ItContext {
	return &ItContext{
		testcaseContext: newTestcaseContext(),
		githubContext:   newGithubContext(),
		bboltDbContext:  newBboltDbContext(dbPath),
	}
}

func (ctx *ItContext) ConvertToTestcase(modelTestcases []model.Testcase) []testcase {
	return ctx.testcaseContext.ConvertToTestcase(modelTestcases)
}

func (ctx *ItContext) GetTestcases() ([]testcase, error) {
	return ctx.testcaseContext.getTestcases()
}

func (ctx *ItContext) AddTestcases(testcases []testcase) error {
	return ctx.testcaseContext.addTestcases(testcases)
}

func (ctx *ItContext) DeleteTestcases(testcases []testcase) error {
	return ctx.testcaseContext.deleteTestcases(testcases)
}

func (ctx *ItContext) GetPrList(nf string) ([]pr, error) {
	return ctx.githubContext.getPrList(nf)
}

func (ctx *ItContext) SaveToDb(bucket, key, value []byte) error {
	return ctx.bboltDbContext.Save(bucket, key, value)
}

func (ctx *ItContext) LoadFromDb(bucket, key []byte) ([]byte, error) {
	return ctx.bboltDbContext.Load(bucket, key)
}

func (ctx *ItContext) UpdateDb(bucket, key, value []byte) error {
	return ctx.bboltDbContext.Update(bucket, key, value)
}

func (ctx *ItContext) RemoveFromDb(bucket, key []byte) error {
	return ctx.bboltDbContext.Remove(bucket, key)
}
