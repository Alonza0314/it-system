package processor

import (
	"backend/internal/context"
	"backend/logger"
	"time"
)

type Processor struct {
	username string
	password string

	jwtSecret    string
	jwtExpiresIn time.Duration

	runnerJwtSecret    string
	runnerJwtExpiresIn time.Duration

	itContext *context.ItContext

	*logger.BackendLogger
}

func NewProcessor(username, password, dbPath, logPath, jwtSecret, runnerJwtSecret string, maxHistoryLength int, jwtExpiresIn, runnerJwtExpiresIn, runnerCheckTimeInterval time.Duration, logger *logger.BackendLogger) *Processor {
	return &Processor{
		username: username,
		password: password,

		jwtSecret:    jwtSecret,
		jwtExpiresIn: jwtExpiresIn,

		runnerJwtSecret:    runnerJwtSecret,
		runnerJwtExpiresIn: runnerJwtExpiresIn,

		itContext: context.NewItContext(dbPath, logPath, maxHistoryLength, runnerCheckTimeInterval),

		BackendLogger: logger,
	}
}

func ReleaseProcessor(p *Processor) error {
	return context.ReleaseItContext(p.itContext)
}
