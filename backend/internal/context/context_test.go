package context_test

import (
	"backend/internal/context"
	"os"
	"testing"
)

var ctx *context.ItContext

func TestMain(m *testing.M) {
	ctx = context.NewItContext(DB_PATH)

	code := m.Run()

	if err := context.ReleaseItContext(ctx); err != nil {
		panic("Failed to release ItContext: " + err.Error())
	}

	os.Exit(code)
}
