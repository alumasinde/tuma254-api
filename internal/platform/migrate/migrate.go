package migrate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Runner struct { db *pgxpool.Pool; dir string }
func New(db *pgxpool.Pool, dir string) *Runner { return &Runner{db: db, dir: dir} }

type fileMigration struct { version string; path string; name string }

func (r *Runner) ensure(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, name TEXT NOT NULL, checksum TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	return err
}

func (r *Runner) files() ([]fileMigration, error) {
	var out []fileMigration
	err := filepath.WalkDir(r.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() || !strings.HasSuffix(path, ".up.sql") { return nil }
		base := strings.TrimSuffix(filepath.Base(path), ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 { return fmt.Errorf("migration must begin with globally unique version: %s", path) }
		out = append(out, fileMigration{version: parts[0], name: parts[1], path: path})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, err
}

func (r *Runner) Up(ctx context.Context) error {
	if err := r.ensure(ctx); err != nil { return err }
	files, err := r.files(); if err != nil { return err }
	for _, m := range files {
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, m.version).Scan(&exists); err != nil { return err }
		if exists { continue }
		b, err := os.ReadFile(m.path); if err != nil { return err }
		tx, err := r.db.Begin(ctx); if err != nil { return err }
		if _, err = tx.Exec(ctx, string(b)); err == nil {
			sum := fmt.Sprintf("%x", sha256.Sum256(b))
			_, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version,name,checksum) VALUES($1,$2,$3)`, m.version, m.name, sum)
		}
		if err != nil { _ = tx.Rollback(ctx); return fmt.Errorf("apply %s: %w", m.path, err) }
		if err := tx.Commit(ctx); err != nil { return err }
	}
	return nil
}

func (r *Runner) Down(ctx context.Context) error {
	if err := r.ensure(ctx); err != nil { return err }
	var version string
	err := r.db.QueryRow(ctx, `SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&version)
	if err != nil { return fmt.Errorf("find last migration: %w", err) }
	var up string
	if err := filepath.WalkDir(r.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".up.sql") { return err }
		base := strings.TrimSuffix(filepath.Base(path), ".up.sql"); parts := strings.SplitN(base, "_", 2)
		if len(parts) == 2 && parts[0] == version { up = path }
		return nil
	}); err != nil { return err }
	if up == "" { return fmt.Errorf("migration file for version %s not found", version) }
	down := strings.TrimSuffix(up, ".up.sql") + ".down.sql"
	b, err := os.ReadFile(down); if err != nil { return fmt.Errorf("read down migration: %w", err) }
	tx, err := r.db.Begin(ctx); if err != nil { return err }
	if _, err = tx.Exec(ctx, string(b)); err == nil { _, err = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version=$1`, version) }
	if err != nil { _ = tx.Rollback(ctx); return err }
	return tx.Commit(ctx)
}

func (r *Runner) Status(ctx context.Context, w io.Writer) error {
	if err := r.ensure(ctx); err != nil { return err }
	rows, err := r.db.Query(ctx, `SELECT version,name,applied_at FROM schema_migrations ORDER BY version`)
	if err != nil { return err }; defer rows.Close()
	for rows.Next() { var v,n string; var t any; if err := rows.Scan(&v,&n,&t); err != nil{return err}; fmt.Fprintf(w, "%s %s %v\n", v,n,t) }
	return rows.Err()
}
