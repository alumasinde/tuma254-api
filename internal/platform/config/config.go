package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv, HTTPAddr, LogLevel, DatabaseURL, RedisURL, MigrationsDir string
}

func Load() (Config, error) {
	c := Config{
		AppEnv: get("APP_ENV", "development"),
		HTTPAddr: get("HTTP_ADDR", ":8080"),
		LogLevel: get("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL: os.Getenv("REDIS_URL"),
		MigrationsDir: get("MIGRATIONS_DIR", "./migrations"),
	}
	if c.DatabaseURL == "" { return c, fmt.Errorf("DATABASE_URL is required") }
	if c.RedisURL == "" { return c, fmt.Errorf("REDIS_URL is required") }
	return c, nil
}
func get(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
