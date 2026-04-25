package context_test

import "backend/internal/context"

var ctx *context.ItContext

func init() {
	ctx = context.NewItContext(DB_PATH)
}
