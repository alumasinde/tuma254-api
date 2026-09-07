package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alumasinde/tuma254-api/internal/database/migrations"
	"github.com/alumasinde/tuma254-api/testkit"
)

func TestMigrations(t *testing.T) {
	uri := os.Getenv("MONGODB_TEST_URI")
	if uri == "" {
		t.Skip("MONGODB_TEST_URI not configured")
	}

	db := testkit.MongoDatabase(t, uri, "tuma254_migration_test")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := migrations.Run(ctx, db, migrations.All()...); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	if err := migrations.Run(ctx, db, migrations.All()...); err != nil {
		t.Fatalf("migrations should be idempotent: %v", err)
	}
}
