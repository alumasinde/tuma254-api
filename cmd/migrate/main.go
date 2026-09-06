package main

import (
	"context"
	"log"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database/mongodb"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := mongodb.Connect(context.Background(), cfg.MongoDBURI, cfg.MongoDBDatabase)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close(context.Background())

	if err := migrations.Run(context.Background(), db.Database(), migrations.All()...); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations complete")
}
