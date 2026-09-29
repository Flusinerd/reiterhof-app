package seed_test

import (
	"context"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestRunIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.New(t)

	for range 2 {
		if err := seed.Run(ctx, pool); err != nil {
			t.Fatalf("Run: %v", err)
		}
	}

	counts := map[string]int{
		"stables": 1, "users": 8, "horses": 7, "horse_riders": 2,
		"blankets": 4, "blanket_rules": 7, "training_profiles": 3,
	}
	for table, want := range counts {
		var got int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
}

func TestSeedData(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewSeeded(t)

	var owner, email string
	err := pool.QueryRow(ctx, `SELECT o.name, o.email FROM horses h JOIN users o ON o.id = h.owner_id WHERE h.id = $1`, seed.HorseLuna).Scan(&owner, &email)
	if err != nil {
		t.Fatal(err)
	}
	if owner != "Jan" || email != "jan@example.org" {
		t.Errorf("Luna owner = %s <%s>", owner, email)
	}

	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM training_profiles WHERE horse_id = $1", seed.HorseFanta).Scan(&status); err != nil || status != "reha" {
		t.Errorf("Fanta status = %q, err %v", status, err)
	}

	var rules int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM blanket_rules WHERE horse_id = $1", seed.HorseLuna).Scan(&rules); err != nil || rules != 5 {
		t.Errorf("Luna rules = %d, err %v", rules, err)
	}
}
