package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv string
	HTTPAddr string
	MongoDBURI string
	MongoDBDatabase string
	LogLevel string
	JWTSigningKey string
	AccessTokenTTL time.Duration
	RefreshTokenTTL time.Duration
}

func Load() (Config, error) {
	accessMinutes, err := intValue("ACCESS_TOKEN_TTL_MINUTES", 15)
	if err != nil { return Config{}, err }
	refreshDays, err := intValue("REFRESH_TOKEN_TTL_DAYS", 30)
	if err != nil { return Config{}, err }

	cfg := Config{
		AppEnv: value("APP_ENV", "development"),
		HTTPAddr: value("HTTP_ADDR", ":8080"),
		MongoDBURI: strings.TrimSpace(os.Getenv("MONGODB_URI")),
		MongoDBDatabase: strings.TrimSpace(os.Getenv("MONGODB_DATABASE")),
		LogLevel: value("LOG_LEVEL", "info"),
		JWTSigningKey: strings.TrimSpace(os.Getenv("JWT_SIGNING_KEY")),
		AccessTokenTTL: time.Duration(accessMinutes) * time.Minute,
		RefreshTokenTTL: time.Duration(refreshDays) * 24 * time.Hour,
	}
	if cfg.MongoDBURI == "" { return Config{}, errors.New("MONGODB_URI is required") }
	if cfg.MongoDBDatabase == "" { return Config{}, errors.New("MONGODB_DATABASE is required") }
	if len(cfg.JWTSigningKey) < 32 { return Config{}, errors.New("JWT_SIGNING_KEY must be at least 32 characters") }
	return cfg, nil
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" { return v }
	return fallback
}

func intValue(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" { return fallback, nil }
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 { return 0, fmt.Errorf("%s must be a positive integer", key) }
	return v, nil
}
