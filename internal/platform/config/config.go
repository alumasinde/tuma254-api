package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv, HTTPAddr, LogLevel, DatabaseURL, RedisURL, MigrationsDir string
	JWTAccessSecret, JWTRefreshSecret string
	JWTAccessTTL, JWTRefreshTTL time.Duration
}

func Load() (Config, error) {
	if err := loadDotEnv(); err != nil { return Config{}, err }
	c := Config{
		AppEnv: get("APP_ENV", "development"), HTTPAddr: get("HTTP_ADDR", ":8080"), LogLevel: get("LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("DATABASE_URL"), RedisURL: os.Getenv("REDIS_URL"), MigrationsDir: get("MIGRATIONS_DIR", "./migrations"),
		JWTAccessSecret: os.Getenv("JWT_ACCESS_SECRET"), JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
	}
	var err error
	if c.JWTAccessTTL, err = time.ParseDuration(get("JWT_ACCESS_TTL", "15m")); err != nil { return c, fmt.Errorf("JWT_ACCESS_TTL: %w", err) }
	if c.JWTRefreshTTL, err = time.ParseDuration(get("JWT_REFRESH_TTL", "720h")); err != nil { return c, fmt.Errorf("JWT_REFRESH_TTL: %w", err) }
	for k,v := range map[string]string{"DATABASE_URL":c.DatabaseURL,"REDIS_URL":c.RedisURL,"JWT_ACCESS_SECRET":c.JWTAccessSecret,"JWT_REFRESH_SECRET":c.JWTRefreshSecret} { if v=="" { return c, fmt.Errorf("%s is required", k) } }
	return c,nil
}
func loadDotEnv() error { err:=godotenv.Load(".env"); if err==nil||errors.Is(err,os.ErrNotExist){return nil}; return fmt.Errorf("load .env: %w",err) }
func get(k,d string) string { if v:=os.Getenv(k);v!="" {return v}; return d }
