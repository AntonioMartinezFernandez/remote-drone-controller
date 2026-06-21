package logger

import (
	"context"
	"log/slog"
)

type NullLogger struct{}

func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

func (nl *NullLogger) Error(_ context.Context, _ string, _ ...slog.Attr) {}
func (nl *NullLogger) Warn(_ context.Context, _ string, _ ...slog.Attr)  {}
func (nl *NullLogger) Info(_ context.Context, _ string, _ ...slog.Attr)  {}
func (nl *NullLogger) Debug(_ context.Context, _ string, _ ...slog.Attr) {}
