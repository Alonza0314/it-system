package logger

import (
	"backend/constant"

	loggergo "github.com/Alonza0314/logger-go/v2"
	loggergoModel "github.com/Alonza0314/logger-go/v2/model"
	loggergoUtil "github.com/Alonza0314/logger-go/v2/util"
)

type BackendLogger struct {
	*loggergo.Logger

	CfgLog  loggergoModel.LoggerInterface
	AccLog  loggergoModel.LoggerInterface
	BckLog  loggergoModel.LoggerInterface
	ProcLog loggergoModel.LoggerInterface
}

func NewBackendLogger(level loggergoUtil.LogLevelString, filePath string, debugMode bool) *BackendLogger {
	logger := loggergo.NewLogger(filePath, debugMode)
	logger.SetLevel(level)

	return &BackendLogger{
		Logger: logger,

		CfgLog:  logger.WithTags(constant.CFG_LOG),
		AccLog:  logger.WithTags(constant.ACC_LOG),
		BckLog:  logger.WithTags(constant.BCK_LOG),
		ProcLog: logger.WithTags(constant.PROC_LOG),
	}
}
