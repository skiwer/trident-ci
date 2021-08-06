package log

import (
	"go.uber.org/zap"
	"strings"
)

var logger *zap.Logger

func GetLogger() *zap.Logger {
	return logger
}

func InitLogger(env string) (err error) {
	env = strings.ToLower(env)

	if strings.Contains(env, "dev") || strings.Contains(env, "test") {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	return
}
