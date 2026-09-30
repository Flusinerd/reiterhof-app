package reha_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reha"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

type jobEnv struct {
	*env
	fake *push.Fake
	job  *reha.Reminders
	clk  time.Time
}

// newJobEnv registers one push token per user and starts a plan for Fanta with the checkup on
// 2026-04-10. The job runs on a fake clock that the test moves.
func newJobEnv(t *testing.T) *jobEnv {
	t.Helper()
	e := newEnv(t)
	ctx := context.Background()
	for _, u := range []string{seed.UserAnna, seed.UserLea, seed.UserSarah, seed.UserJan} {
		if err := push.RegisterToken(ctx, e.pool, seed.StableB, u, "tok-"+u[len(u)-3:], push.PlatformAndroid); err != nil {
			t.Fatal(err)
		}
	}
	e.createPlan(seed.UserAnna, nil)
	j := &jobEnv{env: e, fake: &push.Fake{}}
	j.job = &reha.Reminders{Pool: e.pool, Notify: push.NewNotifier(e.pool, j.fake, nil), Now: func() time.Time { return j.clk }}
	return j
}

// run moves the fake clock (Berlin wall-clock time), runs the job once and returns the recipients.
func (j *jobEnv) run(y int, m time.Month, d, hh, mm int) []string {
	j.t.Helper()
	j.clk = time.Date(y, m, d, hh, mm, 0, 0, berlin)
	j.fake.Reset()
	if err := j.job.Run(context.Background()); err != nil {
		j.t.Fatal(err)
	}
	var to []string
	for _, m := range j.fake.Sent() {
		to = append(to, m.To)
	}
	return to
}

func sameSet(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			found = found || g == w
		}
		if !found {
			return false
		}
	}
	return true
}

func TestCheckupReminders(t *testing.T) {
	j := newJobEnv(t)
	anna, lea := "tok-"+seed.UserAnna[len(seed.UserAnna)-3:], "tok-"+seed.UserLea[len(seed.UserLea)-3:]

	// Three days before: nothing. Two days before, but before 08:00: nothing yet.
	if got := j.run(2026, 4, 7, 12, 0); len(got) != 0 {
		t.Fatalf("3 days before: %v", got)
	}
	if got := j.run(2026, 4, 8, 7, 59); len(got) != 0 {
		t.Fatalf("before 08:00: %v", got)
	}
	// Two days before at 08:00: owner and rider, nobody else (Sarah and the admin are no riders).
	got := j.run(2026, 4, 8, 8, 0)
	if !sameSet(got, anna, lea) {
		t.Fatalf("2 days before: %v, want owner and rider", got)
	}
	m := j.fake.Sent()[0]
	if m.Data["kind"] != push.KindRehaCheckup || m.Data["route"] != "/horses/"+fanta+"/reha" || m.Data["screen"] != "/horses/"+fanta+"/reha" ||
		!strings.Contains(m.Title, "Fanta") || !strings.Contains(m.Body, "in 2 Tagen") || !strings.Contains(m.Body, "10.04.2026") || !strings.Contains(m.Body, "Dr. Berger") {
		t.Errorf("message = %+v", m)
	}
	// Idempotent: running again (also much later that day) sends nothing.
	if got := j.run(2026, 4, 8, 8, 15); len(got) != 0 {
		t.Fatalf("second run: %v", got)
	}
	if got := j.run(2026, 4, 8, 20, 0); len(got) != 0 {
		t.Fatalf("evening run: %v", got)
	}
	if n := j.count(`SELECT count(*) FROM reminders WHERE source_table = 'reha_plans' AND kind = 'reha_checkup'`); n != 2 {
		t.Errorf("reminder rows = %d, want 2", n)
	}
	// One day before: not an offset.
	if got := j.run(2026, 4, 9, 9, 0); len(got) != 0 {
		t.Fatalf("1 day before: %v", got)
	}
	// The morning of the checkup: once.
	if got := j.run(2026, 4, 10, 8, 0); !sameSet(got, anna, lea) || !strings.Contains(j.fake.Sent()[0].Body, "heute") {
		t.Fatalf("checkup day: %v %+v", got, j.fake.Sent())
	}
	if got := j.run(2026, 4, 10, 8, 5); len(got) != 0 {
		t.Fatalf("checkup day again: %v", got)
	}
	// The day after: over.
	if got := j.run(2026, 4, 11, 9, 0); len(got) != 0 {
		t.Fatalf("after: %v", got)
	}

	// A moved checkup date reminds again.
	planID := j.planID()
	j.call(seed.UserAnna, "PATCH", "/api/v1/reha-plans/"+planID, map[string]any{"checkup_date": "2026-04-20"}, 200)
	if got := j.run(2026, 4, 18, 8, 0); !sameSet(got, anna, lea) {
		t.Fatalf("moved checkup: %v", got)
	}

	// Ending the plan stops the reminders.
	j.call(seed.UserAnna, "PATCH", "/api/v1/reha-plans/"+planID, map[string]any{"checkup_date": "2026-04-30"}, 200)
	j.call(seed.UserAnna, "POST", "/api/v1/reha-plans/"+planID+"/end", nil, 200)
	if got := j.run(2026, 4, 28, 9, 0); len(got) != 0 {
		t.Fatalf("ended plan: %v", got)
	}
}

func (j *jobEnv) planID() string {
	j.t.Helper()
	var id string
	if err := j.pool.QueryRow(context.Background(), `SELECT id::text FROM reha_plans WHERE active`).Scan(&id); err != nil {
		j.t.Fatal(err)
	}
	return id
}

// A failed push releases the claim, so the next run tries again; a user who switched the kind off
// gets nothing (handled by the notifier).
func TestCheckupReminderRetriesAfterFailure(t *testing.T) {
	j := newJobEnv(t)
	j.fake.Err = errors.New("expo down")
	j.clk = time.Date(2026, 4, 10, 9, 0, 0, 0, berlin)
	if err := j.job.Run(context.Background()); err == nil {
		t.Fatal("expected the push error")
	}
	if n := j.count(`SELECT count(*) FROM reminders WHERE source_table = 'reha_plans'`); n != 0 {
		t.Fatalf("claims must be released after a failed push, have %d", n)
	}
	j.fake.Err = nil
	if got := j.run(2026, 4, 10, 9, 15); len(got) != 2 {
		t.Fatalf("retry: %v", got)
	}

	// Opt-out via reminder_settings.
	j2 := newJobEnv(t)
	j2.exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'reha_checkup', false)`, seed.StableB, seed.UserLea)
	if got := j2.run(2026, 4, 10, 8, 0); len(got) != 1 {
		t.Fatalf("Lea opted out: %v", got)
	}
}

// The 08:00 rule uses the stable's wall clock, also on the day of the DST change.
func TestCheckupReminderAcrossDST(t *testing.T) {
	j := newJobEnv(t)
	j.call(seed.UserAnna, "PATCH", "/api/v1/reha-plans/"+j.planID(), map[string]any{"checkup_date": "2026-03-31"}, 200)
	// 2026-03-29 is the spring DST change in Berlin; 07:59 CEST is 05:59 UTC.
	if got := j.run(2026, 3, 29, 7, 59); len(got) != 0 {
		t.Fatalf("before 08:00 on the DST day: %v", got)
	}
	if got := j.run(2026, 3, 29, 8, 0); len(got) != 2 {
		t.Fatalf("08:00 on the DST day: %v", got)
	}
	if got := j.run(2026, 3, 29, 23, 0); len(got) != 0 {
		t.Fatalf("again: %v", got)
	}
	// A UTC clock reading is converted: 2026-03-31 05:59 UTC is 07:59 CEST, 06:00 UTC is 08:00.
	j.clk = time.Date(2026, 3, 31, 5, 59, 0, 0, time.UTC)
	j.fake.Reset()
	_ = j.job.Run(context.Background())
	if len(j.fake.Sent()) != 0 {
		t.Fatalf("05:59 UTC: %v", j.fake.Sent())
	}
	j.clk = time.Date(2026, 3, 31, 6, 0, 0, 0, time.UTC)
	_ = j.job.Run(context.Background())
	if len(j.fake.Sent()) != 2 {
		t.Fatalf("06:00 UTC: %v", j.fake.Sent())
	}
}
