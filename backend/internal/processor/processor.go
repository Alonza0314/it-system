package processor

import (
	"backend/logger"
	"time"
)

type Processor struct {
	username string
	password string

	jwtSecret    string
	jwtExpiresIn time.Duration

	*logger.BackendLogger
}

func NewProcessor(username, password string, jwtSecret string, jwtExpiresIn time.Duration, logger *logger.BackendLogger) *Processor {
	return &Processor{
		username: username,
		password: password,

		jwtSecret:    jwtSecret,
		jwtExpiresIn: jwtExpiresIn,

		BackendLogger: logger,
	}
}
