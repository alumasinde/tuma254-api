package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv, HTTPAddr, LogLevel, DatabaseURL, RedisURL, MigrationsDir string
}

func Load() (Config, error) {
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	c := Config{
		AppEnv:        get("APP_ENV", "development"),
		HTTPAddr:      get("HTTP_ADDR", ":8080"),
		LogLevel:      get("LOG_LEVEL", "info"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		MigrationsDir: get("MIGRATIONS_DIR", "./migrations"),
	}

	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return c, fmt.Errorf("REDIS_URL is required")
	}

	return c, nil
}

func loadDotEnv() error {
	err := godotenv.Load(".env")
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("load .env: %w", err)
}

func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
