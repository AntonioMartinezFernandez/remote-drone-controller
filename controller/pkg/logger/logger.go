package logger

import (
	"context"
	"log/slog"
	"os"
)

type Logger interface {
	Error(ctx context.Context, message string, items ...slog.Attr)
	Warn(ctx context.Context, message string, items ...slog.Attr)
	Info(ctx context.Context, message string, items ...slog.Attr)
	Debug(ctx context.Context, message string, items ...slog.Attr)
}

type logger struct {
	sLogger *slog.Logger
}

func NewLogger(level string) *logger {
	opts := &slog.HandlerOptions{Level: levelFromStringLevel(level)}

	return &logger{
		sLogger: slog.New(NewPrettyLogHandler(opts).WithGroup("data")),
	}
}

func NewJsonLogger(level string) *logger {
	jsonHandler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: levelFromStringLevel(level),
		},
	)

	return &logger{
		sLogger: slog.New(jsonHandler),
	}
}

func (l *logger) Error(ctx context.Context, message string, items ...slog.Attr) {
	l.sLogger.LogAttrs(ctx, slog.LevelError, message, items...)
}

func (l *logger) Warn(ctx context.Context, message string, items ...slog.Attr) {
	l.sLogger.LogAttrs(ctx, slog.LevelWarn, message, items...)
}

func (l *logger) Info(ctx context.Context, message string, items ...slog.Attr) {
	l.sLogger.LogAttrs(ctx, slog.LevelInfo, message, items...)
}

func (l *logger) Debug(ctx context.Context, message string, items ...slog.Attr) {
	l.sLogger.LogAttrs(ctx, slog.LevelDebug, message, items...)
}

func levelFromStringLevel(lvl string) slog.Level {
	var logLevel slog.Level
	switch level := lvl; level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelWarn
	}

	return logLevel
}
