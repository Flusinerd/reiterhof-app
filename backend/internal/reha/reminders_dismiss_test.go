package reha_test

import (
	"net/http"
	"testing"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// checkupRow stores the vet checkup push of a plan for a user and returns the plan's id.
func (e *env) checkupRow(user, planID string) {
	e.t.Helper()
	e.exec(`INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
		VALUES ($1, $2, 'reha_checkup', 'Reha-Kontrolle: Fanta', 'Kontrolle am 10.04.2026.', $3, 'reha_plans', $4::uuid, $3)`,
		seed.StableB, user, e.now, planID)
}

func (e *env) listedCheckups(planID string) int {
	e.t.Helper()
	return e.count(`SELECT count(*) FROM reminders WHERE source_table = 'reha_plans' AND source_id = $1::uuid AND dismissed_at IS NULL`, planID)
}

func TestStoredCheckupRemindersFollowThePlan(t *testing.T) {
	e := newEnv(t)
	plan := str(obj(e.createPlan(seed.UserAnna, nil)["plan"])["id"])
	base := "/api/v1/reha-plans/" + plan
	e.checkupRow(seed.UserAnna, plan)
	e.checkupRow(seed.UserLea, plan) // the rider
	if n := e.listedCheckups(plan); n != 2 {
		t.Fatalf("rows before = %d, want 2", n)
	}

	// Editing something else leaves the push alone; the same checkup date sent again is no change.
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"abort_criteria": "neu", "checkup_date": "2026-04-10"}, http.StatusOK)
	if n := e.listedCheckups(plan); n != 2 {
		t.Fatalf("rows after an edit without a new date = %d, want 2", n)
	}
	// A new checkup date makes the push wrong (it names the old date): dismissed for everybody.
	e.call(seed.UserAnna, "PATCH", base, map[string]any{"checkup_date": "2026-04-17"}, http.StatusOK)
	if n := e.listedCheckups(plan); n != 0 {
		t.Fatalf("rows after moving the checkup = %d, want 0", n)
	}

	// Ending the plan dismisses what is left.
	e.exec(`UPDATE reminders SET dismissed_at = NULL WHERE source_id = $1::uuid`, plan)
	e.call(seed.UserAnna, "POST", base+"/end", nil, http.StatusOK)
	if n := e.listedCheckups(plan); n != 0 {
		t.Fatalf("rows after ending = %d, want 0", n)
	}
}

func TestANewPlanDismissesTheCheckupRemindersOfTheOldOne(t *testing.T) {
	e := newEnv(t)
	first := str(obj(e.createPlan(seed.UserAnna, nil)["plan"])["id"])
	e.checkupRow(seed.UserAnna, first)
	second := str(obj(e.createPlan(seed.UserAnna, map[string]any{"diagnosis": "Hufabszess", "start_date": "2026-03-25"})["plan"])["id"])
	e.checkupRow(seed.UserAnna, second)
	if n := e.listedCheckups(first); n != 0 {
		t.Fatalf("old plan rows = %d, want 0 (the plan ended)", n)
	}
	if n := e.listedCheckups(second); n != 1 {
		t.Fatalf("new plan rows = %d, want 1", n)
	}
}
