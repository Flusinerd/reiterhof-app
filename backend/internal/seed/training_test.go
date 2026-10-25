package seed_test

import (
	"context"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/training"
)

func TestSeedTrainingFocuses(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewSeeded(t)

	focuses := func(horse string) string {
		var out string
		err := pool.QueryRow(ctx, `SELECT string_agg(t.f->>'key', ',' ORDER BY t.ord) FROM training_profiles p,
			jsonb_array_elements(p.focuses) WITH ORDINALITY AS t(f, ord) WHERE p.horse_id = $1`, horse).Scan(&out)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if got := focuses(seed.HorseLuna); got != "losgelassenheit,geraderichten" {
		t.Errorf("Luna's focuses = %q", got)
	}
	if got := focuses(seed.HorseFanta); got != "kondition" {
		t.Errorf("Fanta's focuses = %q", got)
	}

	// The blocks start on the Monday of the week of seeding.
	var monday bool
	err := pool.QueryRow(ctx, `SELECT extract(isodow FROM cycle_start) = 1 AND cycle_start <= current_date AND cycle_start > current_date - 7
		FROM training_profiles WHERE horse_id = $1`, seed.HorseLuna).Scan(&monday)
	if err != nil || !monday {
		t.Errorf("Luna's cycle_start is the Monday of this week: %v (%v)", monday, err)
	}

	var keys []string
	err = pool.QueryRow(ctx, `SELECT keys FROM rider_focuses WHERE horse_id = $1 AND user_id = $2`, seed.HorseLuna, seed.UserMia).Scan(&keys)
	if err != nil || len(keys) != 1 || keys[0] != "gelassenheit" {
		t.Errorf("Mia's focus = %v (%v)", keys, err)
	}

	// Every session but the walker's has a focus, and the catalog knows it.
	var withFocus, without int
	err = pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE focus_key IS NOT NULL),
		count(*) FILTER (WHERE focus_key IS NULL AND activity <> 'walker') FROM sessions`).Scan(&withFocus, &without)
	if err != nil {
		t.Fatal(err)
	}
	if withFocus != 16 || without != 0 {
		t.Errorf("sessions with a focus = %d, without (not walker) = %d, want 16 and 0", withFocus, without)
	}
	rows, err := pool.Query(ctx, `SELECT DISTINCT focus_key FROM sessions WHERE focus_key IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			t.Fatal(err)
		}
		if !training.ValidFocusKey(key) {
			t.Errorf("session focus %q is not in the catalog", key)
		}
	}

	// The two planned units: a focus and a unit plan with an exercise of the library in every part,
	// the main one also in exercise_id.
	var planned int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM week_slots w WHERE focus_key IS NOT NULL
		AND exercise_id::text = unit_plan->'main'->>'exercise_id'
		AND (SELECT count(*) FROM exercises e WHERE e.stable_id IS NULL AND e.id IN (
		       (unit_plan->'warmup'->>'exercise_id')::uuid, (unit_plan->'main'->>'exercise_id')::uuid,
		       (unit_plan->'cooldown'->>'exercise_id')::uuid)) = 3`).Scan(&planned)
	if err != nil || planned != 2 {
		t.Errorf("slots with a unit plan = %d (%v)", planned, err)
	}

	// The equipment of the demo stable: nothing is shared.
	var hall, shared int
	err = pool.QueryRow(ctx, `SELECT jsonb_array_length(training_equipment->'hall'->'equipment'),
		(SELECT count(*) FROM jsonb_each(s.training_equipment) WHERE (value->>'shared')::boolean) FROM stables s WHERE id = $1`, seed.StableB).Scan(&hall, &shared)
	if err != nil || hall != 2 || shared != 0 {
		t.Errorf("hall equipment = %d, shared locations = %d (%v)", hall, shared, err)
	}

	// No goal of its own: the horse is trained towards the class above its level.
	var goal *string
	if err := pool.QueryRow(ctx, `SELECT goal_level FROM training_profiles WHERE horse_id = $1`, seed.HorseLuna).Scan(&goal); err != nil || goal != nil {
		t.Errorf("Luna's goal = %v (%v)", goal, err)
	}
}

func TestSeedKeepsLocalTrainingEdits(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.NewSeeded(t)
	if _, err := pool.Exec(ctx, `UPDATE training_profiles SET focuses = '[{"key":"takt"}]' WHERE horse_id = $1`, seed.HorseLuna); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE stables SET training_equipment = '{"hall":{"equipment":[],"shared":true}}' WHERE id = $1`, seed.StableB); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var key string
	var shared bool
	err := pool.QueryRow(ctx, `SELECT p.focuses->0->>'key', (s.training_equipment->'hall'->>'shared')::boolean
		FROM training_profiles p, stables s WHERE p.horse_id = $1 AND s.id = $2`, seed.HorseLuna, seed.StableB).Scan(&key, &shared)
	if err != nil {
		t.Fatal(err)
	}
	if key != "takt" || !shared {
		t.Errorf("a second run overwrote local edits: focus %q, shared %v", key, shared)
	}
}
