package internal

import (
	"github.com/Alonza0314/it-system/runner/config"
	"github.com/Alonza0314/it-system/runner/logger"
)

type runner struct{}

func NewRunner(config *config.Config, logger *logger.RunnerLogger) *runner {
	return &runner{}
}

func (r *runner) Start() {}

func (r *runner) Stop() {}
