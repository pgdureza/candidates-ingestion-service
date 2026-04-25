package logger

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/candidate-ingestion/service/internal/domain/service"
)

// Ensure it implements the interface
var _ service.Logger = (*logrusAdapter)(nil)

type logrusAdapter struct {
	entry *logrus.Entry
}

func New(level string) *logrusAdapter {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)
	return &logrusAdapter{
		entry: logrus.NewEntry(l),
	}
}

func (l *logrusAdapter) Info(args ...interface{})                  { l.entry.Info(args...) }
func (l *logrusAdapter) Infof(format string, args ...interface{})  { l.entry.Infof(format, args...) }
func (l *logrusAdapter) Warn(args ...interface{})                  { l.entry.Warn(args...) }
func (l *logrusAdapter) Warnf(format string, args ...interface{})  { l.entry.Warnf(format, args...) }
func (l *logrusAdapter) Error(args ...interface{})                 { l.entry.Error(args...) }
func (l *logrusAdapter) Errorf(format string, args ...interface{}) { l.entry.Errorf(format, args...) }
func (l *logrusAdapter) Fatal(args ...interface{})                 { l.entry.Fatal(args...) }
func (l *logrusAdapter) Fatalf(format string, args ...interface{}) { l.entry.Fatalf(format, args...) }
func (l *logrusAdapter) Debug(args ...interface{})                 { l.entry.Debug(args...) }
func (l *logrusAdapter) Debugf(format string, args ...interface{}) { l.entry.Debugf(format, args...) }

func (l *logrusAdapter) WithField(key string, value any) service.Logger {
	return &logrusAdapter{entry: l.entry.WithField(key, value)}
}

func (l *logrusAdapter) WithError(err error) service.Logger {
	return &logrusAdapter{entry: l.entry.WithError(err)}
}

func (l *logrusAdapter) WithContext(ctx context.Context) service.Logger {
	return &logrusAdapter{entry: l.entry.WithContext(ctx)}
}

func (l *logrusAdapter) WithFields(fields map[string]any) service.Logger {
	return &logrusAdapter{entry: l.entry.WithFields(logrus.Fields(fields))}
}
