package migrations

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockID int64 = 25420260907

type Runner struct {
	db  *pgxpool.Pool
	dir string
}

func New(db *pgxpool.Pool, dir string) *Runner {
	return &Runner{db: db, dir: dir}
}

func (r *Runner) Up(ctx context.Context) error {
	if err := r.acquireMigrationLock(ctx); err != nil {
		return err
	}
	defer r.releaseMigrationLock(context.Background())

	if err := r.ensureMigrationTable(ctx); err != nil {
		return err
	}

	files, err := r.findMigrationFiles()
	if err != nil {
		return err
	}

	seenVersions := make(map[string]string, len(files))
	for _, path := range files {
		version, err := migrationVersion(path)
		if err != nil {
			return err
		}
		if previous, exists := seenVersions[version]; exists {
			return fmt.Errorf("duplicate migration version %q: %s and %s", version, previous, path)
		}
		seenVersions[version] = path

		if err := r.applyFile(ctx, version, path); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) acquireMigrationLock(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID)
	if err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	return nil
}

func (r *Runner) releaseMigrationLock(ctx context.Context) {
	_, _ = r.db.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockID)
}

func (r *Runner) ensureMigrationTable(ctx context.Context) error {
	_, err := r.db.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())")
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func (r *Runner) findMigrationFiles() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(r.dir, "*.up.sql"))
	if err != nil {
		return nil, fmt.Errorf("find migrations: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func migrationVersion(path string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(path), ".up.sql")
	version, _, found := strings.Cut(base, "_")
	if !found || version == "" {
		return "", fmt.Errorf("invalid migration filename %q: expected <version>_<name>.up.sql", path)
	}
	return version, nil
}

func (r *Runner) applyFile(ctx context.Context, version, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}

	checksum := fmt.Sprintf("%x", sha256.Sum256(content))
	applied, err := r.migrationApplied(ctx, version, checksum)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", path, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("execute migration %s: %w", path, err)
	}

	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations(version, checksum) VALUES($1, $2)", version, checksum); err != nil {
		return fmt.Errorf("record migration %s: %w", path, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", path, err)
	}

	return nil
}

func (r *Runner) migrationApplied(ctx context.Context, version, checksum string) (bool, error) {
	var existingChecksum string
	err := r.db.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", version).Scan(&existingChecksum)

	switch {
	case err == nil:
		if existingChecksum != checksum {
			return false, fmt.Errorf("migration checksum mismatch for version %s; create a new migration instead of editing an applied migration", version)
		}
		return true, nil
	case errors.Is(err, pgx.ErrNoRows):
		return false, nil
	default:
		return false, fmt.Errorf("check migration %s: %w", version, err)
	}
}
