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

func (ctx *ItContext) ExistsInDb(bucket, key string) (bool, error) {
	return ctx.bboltDbContext.Exists([]byte(bucket), []byte(key))
}
