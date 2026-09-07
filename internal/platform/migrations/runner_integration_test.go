package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationsApplyToCleanDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()

	cleanupTestSchema(t, ctx, db)

	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	runner := New(db, filepath.Join(root, "migrations"))
	if err := runner.Up(ctx); err != nil {
		t.Fatalf("apply clean migrations: %v", err)
	}

	var userColumns int
	err = db.QueryRow(ctx, "SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='users'").Scan(&userColumns)
	if err != nil {
		t.Fatalf("inspect users table: %v", err)
	}
	if userColumns == 0 {
		t.Fatal("users table was not created")
	}

	var migrationCount int
	err = db.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&migrationCount)
	if err != nil {
		t.Fatalf("count applied migrations: %v", err)
	}
	if migrationCount != 5 {
		t.Fatalf("expected 5 applied migrations, got %d", migrationCount)
	}
}

func cleanupTestSchema(t *testing.T, ctx context.Context, db *pgxpool.Pool) {
	t.Helper()

	_, err := db.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	if err != nil {
		t.Fatalf("reset test schema: %v", err)
	}
}
