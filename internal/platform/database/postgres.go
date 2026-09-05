package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil { return nil, fmt.Errorf("parse database url: %w", err) }
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute
	cfg.HealthCheckPeriod = time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil { return nil, fmt.Errorf("open database: %w", err) }
	if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("ping database: %w", err) }
	return pool, nil
}
