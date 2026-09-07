package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	HTTPAddr     string
	LogLevel     string
	DatabaseURL  string
	JWTSecret    string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		AppEnv:      value("APP_ENV", "development"),
		HTTPAddr:    value("HTTP_ADDR", ":8080"),
		LogLevel:    value("LOG_LEVEL", "info"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:   strings.TrimSpace(os.Getenv("JWT_ACCESS_SECRET")),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_ACCESS_SECRET must be at least 32 bytes")
	}
	var err error
	if cfg.AccessTTL, err = duration("JWT_ACCESS_TTL", 15*time.Minute); err != nil { return Config{}, err }
	if cfg.RefreshTTL, err = duration("JWT_REFRESH_TTL", 30*24*time.Hour); err != nil { return Config{}, err }
	if cfg.AppEnv != "development" && cfg.AppEnv != "test" && cfg.AppEnv != "production" {
		return Config{}, fmt.Errorf("invalid APP_ENV")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" { return v }
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" { return fallback, nil }
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 { return 0, fmt.Errorf("invalid %s", key) }
	return d, nil
}
