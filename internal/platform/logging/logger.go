package logging

import (
	"log/slog"
	"os"
	"strings"
)

func New(env, level string) *slog.Logger {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug": l = slog.LevelDebug
	case "warn": l = slog.LevelWarn
	case "error": l = slog.LevelError
	default: l = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: l}
	if env == "development" { return slog.New(slog.NewTextHandler(os.Stdout, opts)) }
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
