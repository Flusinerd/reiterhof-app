package stables_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/stables"
)

func TestGroundCondition(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()

	g, err := stables.GetGroundCondition(ctx, pool, seed.StableB)
	if err != nil {
		t.Fatal(err)
	}
	if g.Condition != "" && !stables.ValidCondition(g.Condition) {
		t.Fatalf("unexpected seeded condition %q", g.Condition)
	}

	now := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	if err := stables.SetGroundCondition(ctx, pool, seed.StableB, stables.GroundMuddy, now); err != nil {
		t.Fatal(err)
	}
	g, err = stables.GetGroundCondition(ctx, pool, seed.StableB)
	if err != nil {
		t.Fatal(err)
	}
	if g.Condition != stables.GroundMuddy || g.UpdatedAt == nil || !g.UpdatedAt.Equal(now) {
		t.Errorf("got %+v", g)
	}

	if err := stables.SetGroundCondition(ctx, pool, seed.StableB, "icy", now); !errors.Is(err, stables.ErrInvalidCondition) {
		t.Errorf("invalid condition: err = %v", err)
	}
	unknown := "00000000-0000-4000-8000-0000000009ff"
	if err := stables.SetGroundCondition(ctx, pool, unknown, stables.GroundDry, now); !errors.Is(err, stables.ErrNotFound) {
		t.Errorf("unknown stable set: err = %v", err)
	}
	if _, err := stables.GetGroundCondition(ctx, pool, unknown); !errors.Is(err, stables.ErrNotFound) {
		t.Errorf("unknown stable get: err = %v", err)
	}
}

func TestListLocated(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	list, err := stables.ListLocated(context.Background(), pool)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != seed.StableB || list[0].Lat != 51.66 || list[0].Timezone != "Europe/Berlin" {
		t.Errorf("got %+v", list)
	}
}
