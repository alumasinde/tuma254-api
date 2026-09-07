package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv string
	HTTPAddr string
	LogLevel string
	DatabaseURL string
	JWTSecret string
	OTPHashSecret string
	AccessTTL time.Duration
	RefreshTTL time.Duration
	OTPTTL time.Duration
	OTPResendCooldown time.Duration
	OTPResendWindow time.Duration
	OTPMaxResends int
	OTPMaxAttempts int
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		AppEnv: value("APP_ENV", "development"),
		HTTPAddr: value("HTTP_ADDR", ":8080"),
		LogLevel: value("LOG_LEVEL", "info"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret: strings.TrimSpace(os.Getenv("JWT_ACCESS_SECRET")),
		OTPHashSecret: strings.TrimSpace(os.Getenv("OTP_HASH_SECRET")),
	}
	if cfg.DatabaseURL == "" { return Config{}, errors.New("DATABASE_URL is required") }
	if len(cfg.JWTSecret) < 32 { return Config{}, errors.New("JWT_ACCESS_SECRET must be at least 32 bytes") }
	if len(cfg.OTPHashSecret) < 32 { return Config{}, errors.New("OTP_HASH_SECRET must be at least 32 bytes") }
	var err error
	if cfg.AccessTTL, err = duration("JWT_ACCESS_TTL", 15*time.Minute); err != nil { return Config{}, err }
	if cfg.RefreshTTL, err = duration("JWT_REFRESH_TTL", 30*24*time.Hour); err != nil { return Config{}, err }
	if cfg.OTPTTL, err = duration("OTP_TTL", 10*time.Minute); err != nil { return Config{}, err }
	if cfg.OTPResendCooldown, err = duration("OTP_RESEND_COOLDOWN", 60*time.Second); err != nil { return Config{}, err }
	if cfg.OTPResendWindow, err = duration("OTP_RESEND_WINDOW", time.Hour); err != nil { return Config{}, err }
	if cfg.OTPMaxResends, err = positiveInt("OTP_MAX_RESENDS", 5); err != nil { return Config{}, err }
	if cfg.OTPMaxAttempts, err = positiveInt("OTP_MAX_ATTEMPTS", 5); err != nil { return Config{}, err }
	if cfg.AppEnv != "development" && cfg.AppEnv != "test" && cfg.AppEnv != "production" { return Config{}, fmt.Errorf("invalid APP_ENV") }
	return cfg, nil
}

func value(key, fallback string) string { if v := strings.TrimSpace(os.Getenv(key)); v != "" { return v }; return fallback }
func duration(key string, fallback time.Duration) (time.Duration, error) { v:=strings.TrimSpace(os.Getenv(key)); if v=="" { return fallback,nil }; d,err:=time.ParseDuration(v); if err!=nil||d<=0 { return 0,fmt.Errorf("invalid %s",key) }; return d,nil }
func positiveInt(key string, fallback int) (int,error) { v:=strings.TrimSpace(os.Getenv(key)); if v=="" { return fallback,nil }; n,err:=strconv.Atoi(v); if err!=nil||n<=0 { return 0,fmt.Errorf("invalid %s",key) }; return n,nil }