package requests_test

import (
	"context"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// listedHelperReminders counts the stored helper reminder rows of a request that the reminder
// center would still list (not dismissed).
func (e *env) listedHelperReminders(requestID string) int {
	e.t.Helper()
	var n int
	err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reminders
		WHERE source_table = 'requests' AND source_id = $1 AND kind = 'helper' AND dismissed_at IS NULL`, requestID).Scan(&n)
	if err != nil {
		e.t.Fatal(err)
	}
	return n
}

// A reminder push about a request that is done or cancelled is moot: its row leaves the reminder
// center (withdrawing already deleted it).
func TestDoneAndCancelledRequestsDismissTheirHelperReminders(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	send := func(id string) {
		t.Helper()
		e.setNow(time.Date(2026, 10, 1, 18, 5, 0, 0, berlin)) // the default reminder: the day before at 18:00
		if err := e.svc.SendHelperReminders(ctx); err != nil {
			t.Fatal(err)
		}
		if n := e.listedHelperReminders(id); n == 0 {
			t.Fatalf("no reminder row for %s", id)
		}
	}

	done := e.create(seed.UserAnna, showHelperBody(2)) // Fri 2026-10-02
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+done.ID+"/accept", nil).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/requests/"+done.ID+"/accept", nil).status(t, 200)
	send(done.ID)
	e.do(seed.UserJan, "POST", "/api/v1/requests/"+done.ID+"/done", nil).status(t, 200)
	if n := e.listedHelperReminders(done.ID); n != 0 {
		t.Errorf("rows after done = %d, want 0 (for both helpers)", n)
	}

	e.setNow(time.Date(2026, 9, 30, 10, 0, 0, 0, berlin))
	cancelled := e.create(seed.UserAnna, m{"type": "other", "date": "2026-10-02", "helpers_needed": 1})
	e.do(seed.UserLea, "POST", "/api/v1/requests/"+cancelled.ID+"/accept", nil).status(t, 200)
	send(cancelled.ID)
	e.do(seed.UserAnna, "POST", "/api/v1/requests/"+cancelled.ID+"/cancel", nil).status(t, 200)
	if n := e.listedHelperReminders(cancelled.ID); n != 0 {
		t.Errorf("rows after cancel = %d, want 0", n)
	}
}
