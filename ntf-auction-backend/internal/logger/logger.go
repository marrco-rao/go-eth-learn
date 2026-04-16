package logger

import (
	"context"

	"ntf-auction-backend/internal/model"
	"ntf-auction-backend/internal/repository"
)

type DBLogger struct {
	logs repository.LogRepository
}

func NewDBLogger(logs repository.LogRepository) *DBLogger {
	return &DBLogger{logs: logs}
}

func (l *DBLogger) Info(ctx context.Context, message, method, path string, statusCode int) {
	_ = l.logs.Create(ctx, model.Log{
		Level:      "INFO",
		Message:    message,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
	})
}

func (l *DBLogger) Error(ctx context.Context, message, method, path string, statusCode int) {
	_ = l.logs.Create(ctx, model.Log{
		Level:      "ERROR",
		Message:    message,
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
	})
}
