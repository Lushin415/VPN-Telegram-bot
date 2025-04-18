package logger

import (
	"go.uber.org/zap"
)

var log, _ = zap.NewProduction()

func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

func LogAdminAction(adminID int64, action, params string) {
	log.Info("admin_action", zap.Int64("admin_id", adminID), zap.String("action", action), zap.String("params", params))
}
