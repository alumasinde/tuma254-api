package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database"
	"github.com/alumasinde/tuma254-api/internal/platform/health"
	"github.com/alumasinde/tuma254-api/internal/platform/logger"
	"github.com/alumasinde/tuma254-api/internal/platform/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil { panic(err) }

	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil { log.Error("database connection failed", "error", err); os.Exit(1) }
	defer db.Close()
	log.Info("postgresql connected")

	rdb, err := redis.Open(ctx, cfg.RedisURL)
	if err != nil { log.Error("redis connection failed", "error", err); os.Exit(1) }
	defer rdb.Close()
	log.Info("redis connected")

	h := health.New(db, rdb)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Handle)
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"tuma254-api","version":"v1"}`))
	})

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: logger.RequestID(logger.AccessLog(log, mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil { log.Error("shutdown failed", "error", err) }
	log.Info("tuma254 api stopped")
}
