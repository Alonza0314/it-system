package context

import (
	"backend/model"
	"sync"
)

type testcase struct {
	name string
	link string
}

func (tc *testcase) Name() string {
	return tc.name
}

func (tc *testcase) Link() string {
	return tc.link
}

type testcaseContext struct {
	rwLock sync.RWMutex

	testcases []testcase
}

func newTestcaseContext() *testcaseContext {
	tCtx := &testcaseContext{
		rwLock: sync.RWMutex{},

		testcases: make([]testcase, 0),
	}

	// TODO: Get testcases from db for initialization
	return tCtx
}

func (tCtx *testcaseContext) ConvertToTestcase(modelTestcases []model.Testcase) []testcase {
	testcases := make([]testcase, 0, len(modelTestcases))
	for _, mtc := range modelTestcases {
		testcases = append(testcases, testcase{
			name: mtc.Name,
			link: mtc.Link,
		})
	}

	return testcases
}

func (tCtx *testcaseContext) getTestcases() ([]testcase, error) {
	tCtx.rwLock.RLock()
	defer tCtx.rwLock.RUnlock()

	return tCtx.testcases, nil
}

func (tCtx *testcaseContext) addTestcases(testcases []testcase) error {
	tCtx.rwLock.Lock()
	defer tCtx.rwLock.Unlock()

	tCtx.testcases = append(tCtx.testcases, testcases...)

	// TODO: Add testcases to db
	return nil

}

func (tCtx *testcaseContext) deleteTestcases(testcases []testcase) error {
	tCtx.rwLock.Lock()
	defer tCtx.rwLock.Unlock()

	for _, tcToDelete := range testcases {
		for i, tc := range tCtx.testcases {
			if tc.name == tcToDelete.name {
				tCtx.testcases = append(tCtx.testcases[:i], tCtx.testcases[i+1:]...)
				break
			}
		}
	}
	// TODO: Delete testcases from db
	return nil
}
