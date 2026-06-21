package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
)

const (
	reset = "\033[0m"

	cyan        = 36
	lightGray   = 37
	darkGray    = 90
	lightRed    = 91
	lightYellow = 93
	white       = 97

	timeFormat = "[15:04:05.000]"
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type PrettyLogHandler struct {
	h slog.Handler
	b *bytes.Buffer
	m *sync.Mutex
}

func NewPrettyLogHandler(opts *slog.HandlerOptions) *PrettyLogHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	b := &bytes.Buffer{}
	return &PrettyLogHandler{
		b: b,
		h: slog.NewJSONHandler(b, &slog.HandlerOptions{
			Level:       opts.Level,
			AddSource:   opts.AddSource,
			ReplaceAttr: suppressDefaults(opts.ReplaceAttr),
		}),
		m: &sync.Mutex{},
	}
}

func (h *PrettyLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

func (h *PrettyLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyLogHandler{h: h.h.WithAttrs(attrs), b: h.b, m: h.m}
}

func (h *PrettyLogHandler) WithGroup(name string) slog.Handler {
	return &PrettyLogHandler{h: h.h.WithGroup(name), b: h.b, m: h.m}
}

func (h *PrettyLogHandler) Handle(ctx context.Context, r slog.Record) error {

	level := r.Level.String() + ":"

	switch r.Level {
	case slog.LevelDebug:
		level = colorize(darkGray, level)
	case slog.LevelInfo:
		level = colorize(cyan, level)
	case slog.LevelWarn:
		level = colorize(lightYellow, level)
	case slog.LevelError:
		level = colorize(lightRed, level)
	}

	attrs, err := h.computeAttrs(ctx, r)
	if err != nil {
		return err
	}

	attrsAsBytes, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		return fmt.Errorf("error when marshaling attrs: %w", err)
	}

	fmt.Println(
		colorize(lightGray, r.Time.Format(timeFormat)),
		level,
		colorize(white, r.Message),
		colorize(darkGray, string(attrsAsBytes)),
	)

	return nil
}

func (h *PrettyLogHandler) computeAttrs(
	ctx context.Context,
	r slog.Record,
) (map[string]any, error) {
	h.m.Lock()
	defer func() {
		h.b.Reset()
		h.m.Unlock()
	}()
	if err := h.h.Handle(ctx, r); err != nil {
		return nil, fmt.Errorf("error when calling inner handler's Handle: %w", err)
	}

	var attrs map[string]any
	err := json.Unmarshal(h.b.Bytes(), &attrs)
	if err != nil {
		return nil, fmt.Errorf("error when unmarshaling inner handler's Handle result: %w", err)
	}
	return attrs, nil
}

func suppressDefaults(
	next func([]string, slog.Attr) slog.Attr,
) func([]string, slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey ||
			a.Key == slog.LevelKey ||
			a.Key == slog.MessageKey {
			return slog.Attr{}
		}
		if next == nil {
			return a
		}
		return next(groups, a)
	}
}
