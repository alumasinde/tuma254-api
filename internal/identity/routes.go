package identity

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/handlers"
	"github.com/alumasinde/tuma254-api/internal/identity/repositories"
	"github.com/alumasinde/tuma254-api/internal/identity/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	JWTSecret string
	OTPHashSecret string
	AccessTTL time.Duration
	RefreshTTL time.Duration
	OTPTTL time.Duration
	OTPResendCooldown time.Duration
	OTPResendWindow time.Duration
	OTPMaxResends int
	OTPMaxAttempts int
	SMSProvider string
	SMSWebhookURL string
	SMSWebhookToken string
}

func buildSMSSender(cfg Config, log *slog.Logger) (services.SMSSender, error) {
	if cfg.SMSProvider == "webhook" { return services.NewWebhookSMSSender(cfg.SMSWebhookURL, cfg.SMSWebhookToken) }
	return services.NewLoggingSMSSender(log), nil
}

func BuildService(db *pgxpool.Pool, cfg Config, log *slog.Logger) (*services.Service, error) {
	policy := services.OTPPolicy{
		TTL: cfg.OTPTTL,
		ResendCooldown: cfg.OTPResendCooldown,
		ResendWindow: cfg.OTPResendWindow,
		MaxResends: cfg.OTPMaxResends,
		MaxAttempts: cfg.OTPMaxAttempts,
	}
	sender, err := buildSMSSender(cfg, log)
	if err != nil { return err }
	svc := services.New(repositories.New(db), sender, cfg.JWTSecret, cfg.OTPHashSecret, cfg.AccessTTL, cfg.RefreshTTL, policy)
	return svc, nil
}


func RegisterRoutes(mux *http.ServeMux, db *pgxpool.Pool, cfg Config, log *slog.Logger) error {
	svc, err := BuildService(db, cfg, log)
	if err != nil { return err }
	h := handlers.New(svc)
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/phone/verify", h.VerifyPhone)
	mux.HandleFunc("POST /api/v1/auth/phone/resend", h.ResendPhoneVerification)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.Handle("GET /api/v1/me", h.RequireAuth(http.HandlerFunc(h.Me)))
	return nil
}
