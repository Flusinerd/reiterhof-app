package db_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/Flusinerd/reiterhof-app/backend/internal/db"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
)

func TestStatus(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewUnmigrated(t)
	fsys := fstest.MapFS{
		"0001_a.up.sql": {Data: []byte("CREATE TABLE a (id int);")},
		"0002_b.up.sql": {Data: []byte("CREATE TABLE b (id int);")},
	}

	// Before the first run there is no schema_migrations table: nothing is applied.
	st, err := db.Status(ctx, pool, fsys)
	if err != nil || len(st) != 2 || st[0].Applied || st[1].Applied {
		t.Fatalf("Status before Migrate = %+v, %v", st, err)
	}

	if err := db.Migrate(ctx, pool, fstest.MapFS{"0001_a.up.sql": fsys["0001_a.up.sql"]}); err != nil {
		t.Fatal(err)
	}
	st, err = db.Status(ctx, pool, fsys)
	if err != nil || len(st) != 2 || !st[0].Applied || st[0].AppliedAt == nil || st[1].Applied {
		t.Fatalf("Status after partial Migrate = %+v, %v", st, err)
	}

	// A recorded migration whose file is gone (older binary) still shows up.
	st, err = db.Status(ctx, pool, fstest.MapFS{"0002_b.up.sql": fsys["0002_b.up.sql"]})
	if err != nil || len(st) != 2 || st[0].Version != 1 || !st[0].Applied || st[1].Applied {
		t.Fatalf("Status with missing file = %+v, %v", st, err)
	}
}
