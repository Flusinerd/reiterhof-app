package reminders_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// sentToday stores the push "Fällig: Luna" of today 08:00 for a health item and a user.
func (e *env) sentToday(user, item, kind, title string) {
	e.t.Helper()
	e.exec(`INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
		VALUES ($1, $2, $3, $4, 'Text', $5, 'health_items', $6::uuid, $7)`,
		seed.StableB, user, kind, title, berlinAt(2026, time.September, 30, 8, 0), item, berlinAt(2026, time.September, 30, 8, 1))
}

func TestDoneRemovesTheHealthReminderFromTheList(t *testing.T) {
	e := newEnv(t)
	overdue := e.insertHealth(seed.HorseLuna, "deworming", "Wurmkur", "2026-09-25", "")
	e.exec(`UPDATE health_items SET interval_count = 90 WHERE id = $1`, overdue)
	other := e.insertHealth(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-02", "")
	e.sentToday(seed.UserJan, overdue, "health_due", "Fällig: Luna")
	e.sentToday(seed.UserMia, overdue, "health_due", "Fällig: Luna")
	e.sentToday(seed.UserJan, other, "health_due", "Fällig: Luna")

	// Before: the push of the overdue item and the one of the farrier (the stored row also hides
	// the computed twin of the same item, so each shows up once).
	_, groups := e.list(seed.UserJan, "")
	if got := titles(groups["today"]); !slices.Equal(got, []string{"Fällig: Luna", "Fällig: Luna"}) {
		t.Fatalf("before = %v", got)
	}

	e.do(seed.UserJan, "POST", "/api/v1/health-items/"+overdue+"/done", nil).status(200)

	// After: neither the stored row nor a computed "überfällig" twin of the done item; the other
	// item is untouched. The rider's list is cleaned up too.
	_, groups = e.list(seed.UserJan, "")
	today := groups["today"]
	if len(today) != 1 || today[0].Body != "Text" {
		t.Fatalf("after done (Jan) = %+v", today)
	}
	if _, groups = e.list(seed.UserMia, ""); len(groups["today"]) != 0 {
		t.Fatalf("after done (Mia) = %+v", groups["today"])
	}

	// Undone: the item is overdue again and the push row it dismissed is back, once.
	e.do(seed.UserJan, "DELETE", "/api/v1/health-items/"+overdue+"/done", nil).status(200)
	_, groups = e.list(seed.UserJan, "")
	if got := titles(groups["today"]); !slices.Equal(got, []string{"Fällig: Luna", "Fällig: Luna"}) {
		t.Fatalf("after undo = %v", got)
	}
	if _, groups = e.list(seed.UserMia, ""); len(groups["today"]) != 1 {
		t.Fatalf("after undo (Mia) = %+v", groups["today"])
	}
}

func TestDeletedAndEditedItemsLeaveNoStaleReminder(t *testing.T) {
	e := newEnv(t)
	deleted := e.insertHealth(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-01", "")
	moved := e.insertHealth(seed.HorseLuna, "vaccination", "Impfung", "2026-10-01", "")
	e.sentToday(seed.UserJan, deleted, "health_due", "Fällig: Luna")
	e.sentToday(seed.UserJan, moved, "health_due", "Fällig: Luna")
	if _, groups := e.list(seed.UserJan, ""); len(groups["today"])+len(groups["week"]) != 2 {
		t.Fatalf("before = %+v", groups)
	}

	e.do(seed.UserJan, "DELETE", "/api/v1/health-items/"+deleted, nil).status(204)
	// Moved far out of the window: the push of "morgen (01.10.2026)" is no longer true.
	e.do(seed.UserJan, "PATCH", "/api/v1/health-items/"+moved, m{"due_date": "2026-12-01"}).status(200)
	_, groups := e.list(seed.UserJan, "")
	if n := len(groups["today"]) + len(groups["week"]); n != 0 {
		t.Fatalf("after delete and edit = %+v", groups)
	}
}

func TestEndedMedicationDisappearsFromTheList(t *testing.T) {
	e := newEnv(t)
	course := e.insertHealth(seed.HorseFanta, "medication", "Equioxx", "2026-10-06", "08:00")
	e.sentToday(seed.UserAnna, course, "medication", "Medikament: Fanta")
	_, groups := e.list(seed.UserAnna, "")
	if got := titles(groups["today"]); !slices.Equal(got, []string{"Medikament: Fanta"}) {
		t.Fatalf("before = %v", got)
	}
	e.do(seed.UserAnna, "POST", "/api/v1/health-items/"+course+"/done", nil).status(200)
	// Neither the push row nor today's computed medication item (the course ends today).
	_, groups = e.list(seed.UserAnna, "")
	if n := len(groups["today"]) + len(groups["week"]); n != 0 {
		t.Fatalf("after ending the course = %+v", groups)
	}
	// And it does not come back tomorrow.
	e.at(berlinAt(2026, time.October, 2, 7, 0))
	_, groups = e.list(seed.UserAnna, "")
	if n := len(groups["today"]) + len(groups["week"]); n != 0 {
		t.Fatalf("next day = %+v", groups)
	}
}
