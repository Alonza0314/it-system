package logger

import (
	"github.com/Alonza0314/it-system/runner/constant"

	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

type RunnerLogger struct {
	*loggergo.Logger

	CfgLog  loggergoModel.LoggerInterface
	RunLog  loggergoModel.LoggerInterface
	HttpLog loggergoModel.LoggerInterface
}

func NewRunnerLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) *RunnerLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return &RunnerLogger{
		Logger: logger,

		CfgLog:  logger.WithTags(constant.CFG_LOG),
		RunLog:  logger.WithTags(constant.RUN_LOG),
		HttpLog: logger.WithTags(constant.HTTP_LOG),
	}
}
