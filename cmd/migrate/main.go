package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database/postgres"
	"github.com/alumasinde/tuma254-api/internal/platform/migrations"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "up" {
		log.Fatal("usage: migrate up")
	}
	cfg, err := config.Load()
	if err != nil { log.Fatal(err) }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	if err := migrations.New(db, "./migrations").Up(ctx); err != nil { log.Fatal(err) }
}
