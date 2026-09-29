// Package dbtest creates throwaway Postgres databases for tests.
//
// Set REITERHOF_TEST_DATABASE_URL to an admin connection string (a user that
// may CREATE DATABASE), e.g. postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable.
// Without it, tests using this package are skipped.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

// EnvAdminURL is the environment variable holding the admin connection string.
const EnvAdminURL = "REITERHOF_TEST_DATABASE_URL"

// New creates a fresh, migrated, empty database and returns a pool for it.
// The database is dropped when the test ends. Skips the test if
// REITERHOF_TEST_DATABASE_URL is unset.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool := createDB(t)
	if err := db.Migrate(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("dbtest: migrate: %v", err)
	}
	return pool
}

// NewSeeded is like New but also loads the seed data (see package seed).
func NewSeeded(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool := New(t)
	if err := seed.Run(context.Background(), pool); err != nil {
		t.Fatalf("dbtest: seed: %v", err)
	}
	return pool
}

// NewUnmigrated creates a fresh empty database without applying migrations.
// Use it to test the migrator itself.
func NewUnmigrated(t testing.TB) *pgxpool.Pool {
	t.Helper()
	return createDB(t)
}

func createDB(t testing.TB) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv(EnvAdminURL)
	if adminURL == "" {
		t.Skipf("%s not set; skipping database test", EnvAdminURL)
	}
	ctx := context.Background()

	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("dbtest: random name: %v", err)
	}
	name := "reiterhof_test_" + hex.EncodeToString(suffix[:])

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatalf("dbtest: connect admin database: %v", err)
	}
	defer admin.Close(ctx)
	// Names are generated above, so quoting with pgx.Identifier is just belt and braces.
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		t.Fatalf("dbtest: create database: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("dbtest: parse %s: %v", EnvAdminURL, err)
	}
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatalf("dbtest: open pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, adminURL)
		if err != nil {
			t.Errorf("dbtest: connect for cleanup: %v", err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Errorf("dbtest: drop database %s: %v", name, err)
		}
	})
	return pool
}
