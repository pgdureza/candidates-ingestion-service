package service

import "context"

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	WithContext(ctx context.Context) Logger
	WithField(key string, value any) Logger
	WithFields(fields map[string]any) Logger
	WithError(err error) Logger
}
