// Package migrate applies a service's embedded SQL migrations at startup.
//
// A service owns its schema, so it also owns applying it. This keeps the setup
// to "run Postgres, run the service" with no migration tool to install.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

const ledger = `CREATE TABLE IF NOT EXISTS schema_migrations (
	filename    text PRIMARY KEY,
	applied_at  timestamptz NOT NULL DEFAULT now()
)`

// Run applies every .sql file in dir that has not been applied yet, in
// filename order. Each file runs inside a transaction together with its ledger
// entry, so a failing migration leaves no partial schema behind.
func Run(ctx context.Context, pool *pgxpool.Pool, files fs.FS, dir string) error {
	if _, err := pool.Exec(ctx, ledger); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return fmt.Errorf("read migrations %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && path.Ext(e.Name()) == ".sql" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := isApplied(ctx, pool, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := fs.ReadFile(files, path.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := apply(ctx, pool, name, string(body)); err != nil {
			return err
		}
		log.Printf("migrate: applied %s", name)
	}
	return nil
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, name string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE filename = $1)`, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return exists, nil
}

func apply(ctx context.Context, pool *pgxpool.Pool, name, body string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", name, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, body); err != nil {
		return fmt.Errorf("apply %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record %s: %w", name, err)
	}
	return tx.Commit(ctx)
}
