package context_test

import (
	"backend/internal/context"
	"os"
	"testing"
	"time"
)

var ctx *context.ItContext

func TestMain(m *testing.M) {
	ctx = context.NewItContext(DB_PATH, LOG_PATH, 20, 30*time.Second)

	code := m.Run()

	if err := context.ReleaseItContext(ctx); err != nil {
		panic("Failed to release ItContext: " + err.Error())
	}

	os.Exit(code)
}
