package processor

import (
	"backend/internal/context"
	"backend/logger"
	"backend/model"
	"time"
)

type Processor struct {
	username string
	password string

	jwtSecret    string
	jwtExpiresIn time.Duration

	itContext *context.ItContext

	tmpTenants []model.Tenant

	*logger.BackendLogger
}

func NewProcessor(username, password, dbPath, jwtSecret string, jwtExpiresIn time.Duration, logger *logger.BackendLogger) *Processor {
	return &Processor{
		username: username,
		password: password,

		jwtSecret:    jwtSecret,
		jwtExpiresIn: jwtExpiresIn,

		itContext: context.NewItContext(dbPath),

		tmpTenants: make([]model.Tenant, 0),

		BackendLogger: logger,
	}
}
