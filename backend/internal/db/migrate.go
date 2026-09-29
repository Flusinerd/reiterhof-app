package db

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationLockKey is the Postgres advisory lock that serializes migrators.
const migrationLockKey int64 = 0x7265697465726f66 // "reiterof"

var migrationName = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.up\.sql$`)

type migration struct {
	version int
	name    string // file name, also stored in schema_migrations
}

// Migrate applies all pending NNNN_description.up.sql files from fsys (in the
// root directory) in version order. Each file runs in its own transaction and
// is recorded in schema_migrations. Concurrent callers are serialized by an
// advisory lock, so it is safe to run at every API start.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) error {
	migrations, err := listMigrations(fsys)
	if err != nil {
		return err
	}

	// The advisory lock is bound to a session, so hold one connection throughout.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationLockKey); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		// Use a fresh context: ctx may already be cancelled.
		_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", migrationLockKey)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    integer PRIMARY KEY,
		name       text NOT NULL,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[int]bool{}
	rows, err := conn.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read schema_migrations: %w", err)
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		sql, err := fs.ReadFile(fsys, m.name)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.name, err)
		}
		if err := apply(ctx, conn, m, string(sql)); err != nil {
			return err
		}
	}
	return nil
}

// apply runs one migration and records it, atomically.
func apply(ctx context.Context, conn *pgxpool.Conn, m migration, sql string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s: %w", m.name, err)
	}
	// Rollback after a successful Commit is a no-op.
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, sql); err != nil {
		return fmt.Errorf("apply %s: %w", m.name, err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.version, m.name); err != nil {
		return fmt.Errorf("record %s: %w", m.name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s: %w", m.name, err)
	}
	return nil
}

// listMigrations returns the migration files sorted by version and rejects
// malformed names and duplicate versions.
func listMigrations(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var out []migration
	seen := map[int]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrationName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migration %q: name must match NNNN_description.up.sql", e.Name())
		}
		version, _ := strconv.Atoi(m[1])
		if prev, ok := seen[version]; ok {
			return nil, fmt.Errorf("migrations %q and %q share version %04d", prev, e.Name(), version)
		}
		seen[version] = e.Name()
		out = append(out, migration{version: version, name: e.Name()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}
