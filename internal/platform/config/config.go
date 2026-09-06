package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	AppEnv          string
	HTTPAddr        string
	MongoDBURI      string
	MongoDBDatabase string
	LogLevel        string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:          value("APP_ENV", "development"),
		HTTPAddr:        value("HTTP_ADDR", ":8080"),
		MongoDBURI:      strings.TrimSpace(os.Getenv("MONGODB_URI")),
		MongoDBDatabase: strings.TrimSpace(os.Getenv("MONGODB_DATABASE")),
		LogLevel:        value("LOG_LEVEL", "info"),
	}

	if cfg.MongoDBURI == "" {
		return Config{}, errors.New("MONGODB_URI is required")
	}
	if cfg.MongoDBDatabase == "" {
		return Config{}, errors.New("MONGODB_DATABASE is required")
	}
	return cfg, nil
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
