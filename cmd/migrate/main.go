package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database"
	"github.com/alumasinde/tuma254-api/internal/platform/logger"
	"github.com/alumasinde/tuma254-api/internal/platform/migrate"
)

func main() {
	if len(os.Args) != 2 { fmt.Fprintln(os.Stderr, "usage: migrate [up|down|status]"); os.Exit(2) }
	cfg, err := config.Load()
	if err != nil { panic(err) }
	log := logger.New(cfg.AppEnv, cfg.LogLevel)
	slog.SetDefault(log)
	db, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil { log.Error("database connection failed", "error", err); os.Exit(1) }
	defer db.Close()

	runner := migrate.New(db, cfg.MigrationsDir)
	switch os.Args[1] {
	case "up":
		err = runner.Up(context.Background())
	case "status":
		err = runner.Status(context.Background(), os.Stdout)
	case "down":
		err = runner.Down(context.Background())
	default:
		fmt.Fprintln(os.Stderr, "usage: migrate [up|down|status]")
		os.Exit(2)
	}
	if err != nil { log.Error("migration command failed", "error", err); os.Exit(1) }
}
