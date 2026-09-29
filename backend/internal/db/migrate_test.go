package db_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

func TestMigrateAppliesEmbeddedMigrationsOnce(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewUnmigrated(t)

	for range 2 { // the second run must be a no-op
		if err := db.Migrate(ctx, pool, migrations.FS); err != nil {
			t.Fatalf("Migrate: %v", err)
		}
	}

	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	files, _ := migrations.FS.ReadDir(".")
	if n != len(files) {
		t.Errorf("schema_migrations rows = %d, want %d", n, len(files))
	}
	for _, table := range []string{"stables", "users", "horses", "requests", "sessions", "reha_days"} {
		var ok bool
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&ok); err != nil || !ok {
			t.Errorf("table %s missing (err %v)", table, err)
		}
	}
}

func TestMigrateRollsBackFailedMigration(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewUnmigrated(t)
	fsys := fstest.MapFS{
		"0001_ok.up.sql":  {Data: []byte("CREATE TABLE a (id int);")},
		"0002_bad.up.sql": {Data: []byte("CREATE TABLE b (id int); SELECT nope;")},
	}

	err := db.Migrate(ctx, pool, fsys)
	if err == nil || !strings.Contains(err.Error(), "0002_bad.up.sql") {
		t.Fatalf("err = %v, want failure naming 0002_bad.up.sql", err)
	}
	var aExists, bExists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass('a') IS NOT NULL, to_regclass('b') IS NOT NULL").Scan(&aExists, &bExists); err != nil {
		t.Fatal(err)
	}
	if !aExists || bExists {
		t.Errorf("a exists = %v (want true), b exists = %v (want false)", aExists, bExists)
	}
}

func TestMigrateConcurrent(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewUnmigrated(t)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- db.Migrate(ctx, pool, migrations.FS)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent Migrate: %v", err)
		}
	}
}

func TestMigrateRejectsBadNames(t *testing.T) {
	// Validation happens before any database access, so no pool is needed.
	for name, fsys := range map[string]fstest.MapFS{
		"bad name":  {"1_short.up.sql": {Data: []byte("SELECT 1")}},
		"duplicate": {"0001_a.up.sql": {Data: []byte("SELECT 1")}, "0001_b.up.sql": {Data: []byte("SELECT 1")}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := db.Migrate(context.Background(), nil, fsys); err == nil {
				t.Error("want error")
			}
		})
	}
}
