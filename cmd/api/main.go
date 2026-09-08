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

	"github.com/alumasinde/tuma254-api/internal/identity"
	identityhandlers "github.com/alumasinde/tuma254-api/internal/identity/handlers"
	identityrepo "github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database/postgres"
	"github.com/alumasinde/tuma254-api/internal/platform/logging"
	"github.com/alumasinde/tuma254-api/internal/platform/httpx"
	"github.com/alumasinde/tuma254-api/internal/riders"
	"github.com/alumasinde/tuma254-api/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	log := logging.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)

	startCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := postgres.Open(startCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	identityConfig := identity.Config{
		JWTSecret: cfg.JWTSecret, OTPHashSecret: cfg.OTPHashSecret,
		AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL, OTPTTL: cfg.OTPTTL,
		OTPResendCooldown: cfg.OTPResendCooldown, OTPResendWindow: cfg.OTPResendWindow,
		OTPMaxResends: cfg.OTPMaxResends, OTPMaxAttempts: cfg.OTPMaxAttempts,
		LoginAttemptWindow: cfg.LoginAttemptWindow, LoginMaxAttempts: cfg.LoginMaxAttempts,
		SMSProvider: cfg.SMSProvider, SMSWebhookURL: cfg.SMSWebhookURL, SMSWebhookToken: cfg.SMSWebhookToken,
	}
	identityRepository := identityrepo.New(db)
	identityService, err := identity.BuildServiceWithRepository(identityRepository, identityConfig, log)
	if err != nil {
		log.Error("identity setup failed", "error", err)
		os.Exit(1)
	}
	identityHandler := identityhandlers.New(identityService)

	mux.HandleFunc("POST /api/v1/auth/register", identityHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/phone/verify", identityHandler.VerifyPhone)
	mux.HandleFunc("POST /api/v1/auth/phone/resend", identityHandler.ResendPhoneVerification)
	mux.HandleFunc("POST /api/v1/auth/login", identityHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", identityHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", identityHandler.Logout)
	mux.Handle("GET /api/v1/me", identityHandler.RequireAuth(http.HandlerFunc(identityHandler.Me)))

	users.RegisterRoutes(mux, db, identityHandler, identityRepository)
	riders.RegisterRoutes(mux, db, identityHandler, identityRepository, identityRepository)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = r
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	handler := httpx.WithRequestID(httpx.RequestLogger(log, mux))
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stop:
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) { log.Error("server failed", "error", err); os.Exit(1) }
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil { log.Error("shutdown failed", "error", err) }
}
