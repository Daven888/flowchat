package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global logger instance.
var Log *zap.Logger

// Init initializes the global zap logger with the given log level.
func Init(level string) error {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var l zapcore.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(l)

	logger, err := config.Build()
	if err != nil {
		return err
	}

	Log = logger
	return nil
}

// Sync flushes any buffered log entries.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
