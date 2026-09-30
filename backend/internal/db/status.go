package db

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationState is one migration file and whether it has been applied.
type MigrationState struct {
	Version   int        `json:"version"`
	Name      string     `json:"name"`
	Applied   bool       `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// Status lists the migrations in fsys with their state, in version order. Migrations
// recorded in the database but missing from fsys (a newer binary ran before) are returned
// as applied entries with the recorded name, so the caller can show the mismatch. It
// never changes the database; without a schema_migrations table nothing counts as applied.
func Status(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS) ([]MigrationState, error) {
	files, err := listMigrations(fsys)
	if err != nil {
		return nil, err
	}
	type record struct {
		name string
		at   time.Time
	}
	recorded := map[int]record{}
	rows, err := pool.Query(ctx, "SELECT version, name, applied_at FROM schema_migrations")
	if err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" { // 42P01 = undefined_table
			return nil, fmt.Errorf("read schema_migrations: %w", err)
		}
	} else {
		defer rows.Close()
		for rows.Next() {
			var v int
			var r record
			if err := rows.Scan(&v, &r.name, &r.at); err != nil {
				return nil, fmt.Errorf("scan schema_migrations: %w", err)
			}
			recorded[v] = r
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read schema_migrations: %w", err)
		}
	}

	var out []MigrationState
	known := map[int]bool{}
	for _, m := range files {
		known[m.version] = true
		st := MigrationState{Version: m.version, Name: m.name}
		if r, ok := recorded[m.version]; ok {
			at := r.at
			st.Applied, st.AppliedAt = true, &at
		}
		out = append(out, st)
	}
	for v, r := range recorded {
		if !known[v] {
			at := r.at
			out = append(out, MigrationState{Version: v, Name: r.name, Applied: true, AppliedAt: &at})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}
