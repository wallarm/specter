package testutil

import (
	"go.uber.org/zap"
)

func ReplaceGlobalLogger() *zap.Logger {
	log := NewLogger()
	zap.ReplaceGlobals(log)
	zap.RedirectStdLog(log)
	return log
}

func NewLogger() *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	conf.OutputPaths = []string{"stdout"}
	log, err := conf.Build(zap.AddCaller(), zap.AddStacktrace(zap.PanicLevel))
	if err != nil {
		panic(err)
	}
	return log
}
