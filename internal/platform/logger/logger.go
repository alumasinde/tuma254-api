package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
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

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" { id = fmtID() }
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

type requestIDKey struct{}
func RequestIDFromContext(ctx context.Context) string { v, _ := ctx.Value(requestIDKey{}).(string); return v }
func fmtID() string { return strings.ReplaceAll(time.Now().UTC().Format("20060102T150405.000000000Z"), ".", "") }

func AccessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info("http request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds(), "request_id", RequestIDFromContext(r.Context()))
	})
}
