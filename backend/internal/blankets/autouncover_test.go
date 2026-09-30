package blankets_test

import (
	"context"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func (e *env) runAutoUncover(at time.Time) {
	e.t.Helper()
	e.at(at)
	if err := e.svc.RunAutoUncover(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

// enableAutoUncover switches the automatic uncovering on for the seed stable (Mon-Fri).
func (e *env) enableAutoUncover(clock string) {
	e.t.Helper()
	e.exec(`UPDATE stables SET auto_uncover_time = $2::time WHERE id = $1`, seed.StableB, clock)
}

// stateRows counts the state rows of a horse and day, and how many of them are automatic.
func (e *env) stateRows(horse, day string) (total, automatic int) {
	e.t.Helper()
	err := e.pool.QueryRow(context.Background(), `
		SELECT count(*), count(*) FILTER (WHERE automatic) FROM blanket_states WHERE horse_id = $1 AND day = $2::date`,
		horse, day).Scan(&total, &automatic)
	if err != nil {
		e.t.Fatal(err)
	}
	return total, automatic
}

func TestAutoUncoverWeekday(t *testing.T) {
	e := newEnv(t)
	e.enableAutoUncover("12:30")
	night := "2026-09-30" // Wednesday night, uncovered on Thursday 2026-10-01
	evening := berlinAt(2026, 9, 30, 19, 0)
	e.setState(seed.HorseLuna, night, "covered", evening)
	e.setState(seed.HorseFanta, night, "covered", evening)
	e.setState(seed.HorseFanta, night, "uncovered", berlinAt(2026, 10, 1, 8, 0)) // already uncovered by hand
	e.setState(seed.HorseBalu, night, "checked", evening)
	// Cookie has no state at all. Merlin was covered, uncovered and covered again.
	e.setState(seed.HorseMerlin, night, "covered", evening)
	e.setState(seed.HorseMerlin, night, "uncovered", evening.Add(time.Hour))
	e.setState(seed.HorseMerlin, night, "covered", evening.Add(2*time.Hour))

	// Before 12:30 nothing happens.
	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 29))
	if total, auto := e.stateRows(seed.HorseLuna, night); total != 1 || auto != 0 {
		t.Fatalf("before 12:30: %d rows, %d automatic", total, auto)
	}

	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 30))
	for _, h := range []string{seed.HorseLuna, seed.HorseMerlin} {
		if total, auto := e.stateRows(h, night); total != map[string]int{seed.HorseLuna: 2, seed.HorseMerlin: 4}[h] || auto != 1 {
			t.Errorf("horse %s: %d rows, %d automatic", h, total, auto)
		}
	}
	var action string
	var by, with *string
	var automatic bool
	var changedAt time.Time
	err := e.pool.QueryRow(context.Background(), `
		SELECT action, changed_by::text, covered_with::text, automatic, changed_at FROM blanket_states
		WHERE horse_id = $1 AND day = $2::date ORDER BY changed_at DESC, created_at DESC LIMIT 1`, seed.HorseLuna, night).
		Scan(&action, &by, &with, &automatic, &changedAt)
	if err != nil {
		t.Fatal(err)
	}
	if action != blankets.ActionUncovered || by != nil || with != nil || !automatic || !changedAt.Equal(berlinAt(2026, 10, 1, 12, 30)) {
		t.Errorf("automatic state = %s by=%v with=%v automatic=%v at=%s", action, by, with, automatic, changedAt)
	}
	// Uncovered, checked and stateless horses are untouched.
	for h, want := range map[string]int{seed.HorseFanta: 2, seed.HorseBalu: 1, seed.HorseCookie: 0} {
		if total, auto := e.stateRows(h, night); total != want || auto != 0 {
			t.Errorf("horse %s: %d rows, %d automatic, want %d rows", h, total, auto, want)
		}
	}
	// Nothing was written for the coming night.
	for _, h := range e.allHorses() {
		if total, _ := e.stateRows(h, "2026-10-01"); total != 0 {
			t.Errorf("horse %s has %d states for the coming night", h, total)
		}
	}

	// Running again (also later in the window) inserts nothing.
	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 30))
	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 31))
	e.runAutoUncover(berlinAt(2026, 10, 1, 14, 59))
	if total, auto := e.stateRows(seed.HorseLuna, night); total != 2 || auto != 1 {
		t.Errorf("after reruns: %d rows, %d automatic", total, auto)
	}

	// The overview of the coming night is not affected: nobody is done.
	e.at(berlinAt(2026, 10, 1, 13, 0))
	today := e.today(seed.UserJan)
	if today.Day != "2026-10-01" || today.Progress.Done != 0 {
		t.Errorf("today = day %s, done %d", today.Day, today.Progress.Done)
	}
	for _, h := range today.Horses {
		if h.State != nil || h.Done {
			t.Errorf("horse %s has state %+v in the coming night", h.Horse.Name, h.State)
		}
	}

	// The history shows the automatic state without author.
	var hist struct {
		States []struct {
			stateJSON
			Automatic bool `json:"automatic"`
		} `json:"states"`
	}
	e.do(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna+"/blanket-states", nil).status(t, 200).into(t, &hist)
	if len(hist.States) != 2 || !hist.States[0].Automatic || hist.States[0].Action != "uncovered" ||
		hist.States[0].ChangedBy != nil || hist.States[1].Automatic {
		t.Errorf("history = %+v", hist.States)
	}
}

func TestAutoUncoverDoesNotRun(t *testing.T) {
	night := "2026-09-30"
	evening := berlinAt(2026, 9, 30, 19, 0)
	cases := []struct {
		name  string
		setup func(e *env)
		at    time.Time
		day   string
	}{
		{"from 15:00 on", func(e *env) { e.enableAutoUncover("12:30") }, berlinAt(2026, 10, 1, 15, 0), night},
		{"disabled (NULL)", func(e *env) {}, berlinAt(2026, 10, 1, 12, 30), night},
		{"weekend not in days", func(e *env) { e.enableAutoUncover("12:30") }, berlinAt(2026, 10, 3, 12, 30), "2026-10-02"},
		{"sunday not in days", func(e *env) { e.enableAutoUncover("12:30") }, berlinAt(2026, 10, 4, 13, 0), "2026-10-03"},
		{"weekday removed from days", func(e *env) {
			e.enableAutoUncover("12:30")
			e.exec(`UPDATE stables SET auto_uncover_days = '{6,7}'`)
		}, berlinAt(2026, 10, 1, 12, 30), night},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			c.setup(e)
			e.setState(seed.HorseLuna, c.day, "covered", evening)
			e.runAutoUncover(c.at)
			if total, auto := e.stateRows(seed.HorseLuna, c.day); total != 1 || auto != 0 {
				t.Errorf("%d rows, %d automatic", total, auto)
			}
		})
	}
}

func TestAutoUncoverConfiguredDays(t *testing.T) {
	e := newEnv(t)
	e.enableAutoUncover("11:00")
	e.exec(`UPDATE stables SET auto_uncover_days = '{6,7}'`)
	e.setState(seed.HorseLuna, "2026-10-02", "covered", berlinAt(2026, 10, 2, 19, 0))
	e.runAutoUncover(berlinAt(2026, 10, 3, 11, 0)) // Saturday
	if total, auto := e.stateRows(seed.HorseLuna, "2026-10-02"); total != 2 || auto != 1 {
		t.Errorf("Saturday: %d rows, %d automatic", total, auto)
	}
}

func TestAutoUncoverClosesRequests(t *testing.T) {
	e := newEnv(t)
	e.enableAutoUncover("12:30")
	var out struct{ ID string }
	e.do(seed.UserJan, "POST", "/api/v1/requests", m{"type": "blanket", "horse_id": seed.HorseLuna, "date": "2026-09-30", "payload": m{}}).status(t, 201).into(t, &out)
	e.setState(seed.HorseLuna, "2026-09-30", "covered", berlinAt(2026, 9, 30, 19, 0))
	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 30))
	var st, fb string
	if err := e.pool.QueryRow(context.Background(), `SELECT status, COALESCE(payload->>'feedback', '') FROM requests WHERE id = $1`, out.ID).Scan(&st, &fb); err != nil {
		t.Fatal(err)
	}
	if st != "done" || fb != "Automatisch abgedeckt" {
		t.Errorf("request: %s %q", st, fb)
	}
}

func TestAutoUncoverKeepsManualTapsWorking(t *testing.T) {
	e := newEnv(t)
	e.enableAutoUncover("12:30")
	luna := "/api/v1/horses/" + seed.HorseLuna + "/blanket-state"
	e.do(seed.UserMia, "POST", luna, m{"action": "covered"}).status(t, 200)
	e.runAutoUncover(berlinAt(2026, 10, 1, 12, 30))
	// A manual "uncovered" afterwards is the same action as the newest state, so it is a
	// no-op and returns the automatic row.
	e.at(berlinAt(2026, 10, 1, 12, 40))
	var res struct {
		State struct {
			ID, Day   string
			Automatic bool
		}
	}
	e.do(seed.UserMia, "POST", luna, m{"action": "uncovered"}).status(t, 200).into(t, &res)
	if res.State.Day != "2026-09-30" || !res.State.Automatic {
		t.Errorf("state = %+v", res.State)
	}
	if total, _ := e.stateRows(seed.HorseLuna, "2026-09-30"); total != 2 {
		t.Errorf("rows = %d, want 2", total)
	}
}

func TestAutoUncoverSchema(t *testing.T) {
	e := newEnv(t)
	var days []int16
	var clock *time.Time
	if err := e.pool.QueryRow(context.Background(), `SELECT auto_uncover_days, auto_uncover_time FROM stables WHERE id = $1`, seed.StableB).Scan(&days, &clock); err != nil {
		t.Fatal(err)
	}
	if len(days) != 5 || days[0] != 1 || days[4] != 5 || clock != nil {
		t.Errorf("defaults: days %v, time %v", days, clock)
	}
	for _, bad := range []string{`'{0}'`, `'{8}'`, `'{1,9}'`} {
		if _, err := e.pool.Exec(context.Background(), `UPDATE stables SET auto_uncover_days = `+bad); err == nil {
			t.Errorf("auto_uncover_days = %s was accepted", bad)
		}
	}
}
