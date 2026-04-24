package context

import "backend/model"

type ItContext struct {
	testcaseContext *testcaseContext
}

func NewItContext() *ItContext {
	return &ItContext{
		testcaseContext: newTestcaseContext(),
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
