package main

import (
	"context"
	"encoding/json"
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
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := redis.Open(startupCtx, cfg.RedisURL)
	if err != nil {
		log.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	h := health.New(db, rdb)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.Live)
	mux.HandleFunc("GET /ready", h.Ready)
	mux.HandleFunc("GET /api/v1", apiInfo)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           logger.RequestID(logger.AccessLog(log, mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", cfg.HTTPAddr, "environment", cfg.AppEnv)
		serverErr <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", "error", err)
	}
	log.Info("tuma254 api stopped")
}

func apiInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"service": "tuma254-api",
		"version": "v1",
	})
}
