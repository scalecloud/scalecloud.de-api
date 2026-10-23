package main

import (
	"encoding/json"
	"fmt"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

type sentryZapCore struct {
	logger sentry.Logger
	level  zapcore.LevelEnabler
	fields []zapcore.Field
}

func newSentryZapCore(logger sentry.Logger, level zapcore.LevelEnabler) zapcore.Core {
	return &sentryZapCore{logger: logger, level: level}
}

func (core *sentryZapCore) Enabled(level zapcore.Level) bool {
	return core.level.Enabled(level)
}

func (core *sentryZapCore) With(fields []zapcore.Field) zapcore.Core {
	withFields := &sentryZapCore{
		logger: core.logger,
		level:  core.level,
		fields: append([]zapcore.Field(nil), core.fields...),
	}
	withFields.fields = append(withFields.fields, fields...)
	return withFields
}

func (core *sentryZapCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}

func (core *sentryZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	logEntry := core.logger.Info()
	switch entry.Level {
	case zapcore.DebugLevel:
		logEntry = core.logger.Debug()
	case zapcore.WarnLevel:
		logEntry = core.logger.Warn()
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel:
		logEntry = core.logger.Error()
	case zapcore.FatalLevel:
		logEntry = core.logger.LFatal()
	}

	if entry.LoggerName != "" {
		logEntry = logEntry.String("logger", entry.LoggerName)
	}
	if entry.Caller.Defined {
		logEntry = logEntry.String("caller", entry.Caller.TrimmedPath())
	}

	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range core.fields {
		field.AddTo(encoder)
	}
	for _, field := range fields {
		field.AddTo(encoder)
	}
	for key, value := range encoder.Fields {
		logEntry = addSentryAttribute(logEntry, key, value)
	}
	logEntry.Emit(entry.Message)
	return nil
}

func (core *sentryZapCore) Sync() error {
	return nil
}

func addSentryAttribute(logEntry sentry.LogEntry, key string, value any) sentry.LogEntry {
	switch value := value.(type) {
	case string:
		return logEntry.String(key, value)
	case bool:
		return logEntry.Bool(key, value)
	case int:
		return logEntry.Int(key, value)
	case int8:
		return logEntry.Int(key, int(value))
	case int16:
		return logEntry.Int(key, int(value))
	case int32:
		return logEntry.Int(key, int(value))
	case int64:
		return logEntry.Int64(key, value)
	case float32:
		return logEntry.Float64(key, float64(value))
	case float64:
		return logEntry.Float64(key, value)
	default:
		encoded, err := json.Marshal(value)
		if err == nil {
			return logEntry.String(key, string(encoded))
		}
		return logEntry.String(key, fmt.Sprint(value))
	}
}
