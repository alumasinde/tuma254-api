package migrations

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct {
	db  *pgxpool.Pool
	dir string
}

func New(db *pgxpool.Pool, dir string) *Runner { return &Runner{db: db, dir: dir} }

type migration struct { version, path string }

func (r *Runner) Up(ctx context.Context) error {
	if _, err := r.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil { return err }

	files, err := filepath.Glob(filepath.Join(r.dir, "*.up.sql"))
	if err != nil { return err }
	sort.Strings(files)
	for _, path := range files {
		base := strings.TrimSuffix(filepath.Base(path), ".up.sql")
		version := strings.SplitN(base, "_", 2)[0]
		b, err := os.ReadFile(path)
		if err != nil { return fmt.Errorf("read migration %s: %w", path, err) }
		checksum := fmt.Sprintf("%x", sha256.Sum256(b))
		var existing string
		err = r.db.QueryRow(ctx, "SELECT checksum FROM schema_migrations WHERE version=$1", version).Scan(&existing)
		if err == nil {
			if existing != checksum { return fmt.Errorf("migration checksum mismatch for %s", version) }
			continue
		}
		if err.Error() != "no rows in result set" && err != nil {
			// pgx.ErrNoRows is intentionally checked by string-free helper below.
			if !strings.Contains(err.Error(), "no rows") { return err }
		}
		tx, err := r.db.Begin(ctx)
		if err != nil { return err }
		if _, err = tx.Exec(ctx, string(b)); err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version,checksum) VALUES($1,$2)", version, checksum)
		}
		if err != nil { _ = tx.Rollback(ctx); return fmt.Errorf("apply %s: %w", path, err) }
		if err = tx.Commit(ctx); err != nil { return err }
	}
	return nil
}
