package blankets_test

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/presence"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func (e *env) runLastPerson(at time.Time) {
	e.t.Helper()
	e.at(at)
	if err := e.svc.RunLastPerson(context.Background()); err != nil {
		e.t.Fatal(err)
	}
}

func sorted(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

func TestLastPersonNobodyPresentNotifiesOwners(t *testing.T) {
	e := newEnv(t)
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 30))

	// Owners of the open horses: Jan (Luna), Anna (Fanta, Nala), Jonas (Balu), Tom, Sarah, Kai.
	want := sorted([]string{seed.UserJan, seed.UserAnna, seed.UserJonas, seed.UserTom, seed.UserSarah, seed.UserKai})
	if got := sorted(recipients(e.fake)); !reflect.DeepEqual(got, want) {
		t.Fatalf("recipients = %v, want %v", got, want)
	}
	anna := sentTo(e.fake, seed.UserAnna)
	if len(anna) != 1 || !strings.Contains(anna[0].Body, "Fanta") || !strings.Contains(anna[0].Body, "Nala") || strings.Contains(anna[0].Body, "Luna") {
		t.Errorf("Anna's push = %+v", anna)
	}
	if anna[0].Data["kind"] != push.KindLastPerson || anna[0].Data["screen"] != "/blankets" {
		t.Errorf("push data = %v", anna[0].Data)
	}
	// Riders are not owners: Mia gets nothing.
	if len(sentTo(e.fake, seed.UserMia)) != 0 {
		t.Error("rider was notified")
	}

	// Idempotent: the same slot and later slots send nothing more.
	before := len(e.fake.Sent())
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 30))
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 31))
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 45))
	if n := len(e.fake.Sent()); n != before {
		t.Errorf("repeated runs sent %d more pushes", n-before)
	}
	// The next night reminds again.
	e.runLastPerson(berlinAt(2026, 10, 1, 20, 30))
	if n := len(e.fake.Sent()); n != 2*before {
		t.Errorf("next night: %d pushes, want %d", n, 2*before)
	}
}

func TestLastPersonExactlyOnePresent(t *testing.T) {
	e := newEnv(t)
	e.setState(seed.HorseLuna, "2026-09-30", "covered", berlinAt(2026, 9, 30, 19, 0))
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))

	e.runLastPerson(berlinAt(2026, 9, 30, 20, 30))
	msgs := e.fake.Sent()
	if len(msgs) != 1 || msgs[0].To != "tok-"+seed.UserTom {
		t.Fatalf("pushes = %+v", msgs)
	}
	// Six horses are open; Luna is done and not listed; long lists are shortened.
	if strings.Contains(msgs[0].Body, "Luna") || !strings.Contains(msgs[0].Body, "Balu, Cookie, Fanta, Merlin, Nala und 1 weitere") {
		t.Errorf("body = %q", msgs[0].Body)
	}
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 45))
	if len(e.fake.Sent()) != 1 {
		t.Errorf("repeat sent %d pushes", len(e.fake.Sent()))
	}
}

func TestLastPersonRespectsTimeWindowAndSlots(t *testing.T) {
	e := newEnv(t)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	for _, at := range []time.Time{
		berlinAt(2026, 9, 30, 20, 29), // before the reminder time
		berlinAt(2026, 9, 30, 20, 37), // between two slots
		berlinAt(2026, 9, 30, 22, 0),  // window is over
		berlinAt(2026, 9, 30, 23, 15),
		berlinAt(2026, 10, 1, 7, 30), // morning of the same night
	} {
		e.runLastPerson(at)
		if n := len(e.fake.Sent()); n != 0 {
			t.Fatalf("push at %s", at)
		}
	}
	// 21:45 is the last slot (20:30 + 5 x 15 min).
	e.runLastPerson(berlinAt(2026, 9, 30, 21, 46))
	if len(e.fake.Sent()) != 1 {
		t.Errorf("no push in the 21:45 slot")
	}
}

func TestLastPersonSeveralPresentRechecksUntilOnlyOneIsLeft(t *testing.T) {
	e := newEnv(t)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e.present(seed.UserKai, berlinAt(2026, 9, 30, 18, 30))

	for _, at := range []time.Time{berlinAt(2026, 9, 30, 20, 30), berlinAt(2026, 9, 30, 20, 45), berlinAt(2026, 9, 30, 21, 0)} {
		e.runLastPerson(at)
	}
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed while two people are present: %+v", e.fake.Sent())
	}
	// Kai leaves at 21:20, so the 21:30 check finds Tom alone.
	e.leave(seed.UserKai, berlinAt(2026, 9, 30, 21, 20))
	e.runLastPerson(berlinAt(2026, 9, 30, 21, 30))
	msgs := e.fake.Sent()
	if len(msgs) != 1 || msgs[0].To != "tok-"+seed.UserTom {
		t.Fatalf("pushes = %+v", msgs)
	}
}

func TestLastPersonNoReminderAfterTenWhenTwoWereStillThere(t *testing.T) {
	e := newEnv(t)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e.present(seed.UserKai, berlinAt(2026, 9, 30, 18, 30))
	for at := berlinAt(2026, 9, 30, 20, 30); at.Before(berlinAt(2026, 9, 30, 21, 50)); at = at.Add(15 * time.Minute) {
		e.runLastPerson(at)
	}
	e.leave(seed.UserKai, berlinAt(2026, 9, 30, 21, 50))
	e.runLastPerson(berlinAt(2026, 9, 30, 22, 0))
	e.runLastPerson(berlinAt(2026, 9, 30, 22, 15))
	if len(e.fake.Sent()) != 0 {
		t.Errorf("pushed after 22:00: %+v", e.fake.Sent())
	}
}

func TestLastPersonEverythingDoneAndOptOut(t *testing.T) {
	e := newEnv(t)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	for _, h := range e.allHorses() {
		e.setState(h, "2026-09-30", "checked", berlinAt(2026, 9, 30, 19, 0))
	}
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 30))
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed although everything is done: %+v", e.fake.Sent())
	}

	// A state from the previous night does not count for tonight.
	e2 := newEnv(t)
	e2.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	for _, h := range e2.allHorses() {
		e2.setState(h, "2026-09-29", "checked", berlinAt(2026, 9, 29, 19, 0))
	}
	e2.exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'last_person', false)`, seed.StableB, seed.UserTom)
	e2.runLastPerson(berlinAt(2026, 9, 30, 20, 30))
	// Tom opted out: nothing is delivered (the notifier honors reminder_settings).
	if len(e2.fake.Sent()) != 0 {
		t.Errorf("opted-out user was notified: %+v", e2.fake.Sent())
	}
	// Without the opt-out, the same situation notifies.
	e3 := newEnv(t)
	e3.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e3.runLastPerson(berlinAt(2026, 9, 30, 20, 30))
	if len(e3.fake.Sent()) != 1 {
		t.Errorf("baseline: %d pushes", len(e3.fake.Sent()))
	}
}

func TestLastPersonUsesStableReminderTime(t *testing.T) {
	e := newEnv(t)
	e.exec(`UPDATE stables SET reminder_time = '19:00' WHERE id = $1`, seed.StableB)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 30)) // 19:00 + 90 min is a slot
	if len(e.fake.Sent()) != 1 {
		t.Fatalf("pushes = %d", len(e.fake.Sent()))
	}
	td := e.today(seed.UserMia)
	if td.ReminderTime != "19:00" {
		t.Errorf("reminder time = %s", td.ReminderTime)
	}
	e2 := newEnv(t)
	e2.exec(`UPDATE stables SET reminder_time = '19:00' WHERE id = $1`, seed.StableB)
	e2.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e2.runLastPerson(berlinAt(2026, 9, 30, 19, 0))
	if len(e2.fake.Sent()) != 1 {
		t.Fatalf("no push at the reminder time")
	}
}

func TestLastPersonFailedPushIsRetried(t *testing.T) {
	e := newEnv(t)
	e.present(seed.UserTom, berlinAt(2026, 9, 30, 18, 0))
	e.fake.Err = context.DeadlineExceeded
	e.at(berlinAt(2026, 9, 30, 20, 30))
	if err := e.svc.RunLastPerson(context.Background()); err == nil {
		t.Fatal("expected the push error")
	}
	e.fake.Err = nil
	e.runLastPerson(berlinAt(2026, 9, 30, 20, 45))
	if len(e.fake.Sent()) != 1 {
		t.Errorf("retry sent %d pushes", len(e.fake.Sent()))
	}
}

func TestCheckOutHook(t *testing.T) {
	e := newEnv(t)
	presence.AfterCheckOut = func(ctx context.Context, stableID, userID string) { e.svc.OnCheckOut(ctx, stableID, userID) }
	t.Cleanup(func() { presence.AfterCheckOut = nil })

	checkIn := func(user string) { e.do(user, "POST", "/api/v1/presence/check-in", nil).status(t, 200) }
	checkOut := func(user string) { e.do(user, "POST", "/api/v1/presence/check-out", nil).status(t, 200) }

	// Two people: the first to leave is not the last one, the second is.
	e.at(berlinAt(2026, 9, 30, 20, 0))
	checkIn(seed.UserTom)
	checkIn(seed.UserKai)
	e.at(berlinAt(2026, 9, 30, 21, 10))
	checkOut(seed.UserTom)
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("Tom was not the last person: %+v", e.fake.Sent())
	}
	checkOut(seed.UserKai)
	msgs := e.fake.Sent()
	if len(msgs) != 1 || msgs[0].To != "tok-"+seed.UserKai || !strings.Contains(msgs[0].Body, "Balu") || msgs[0].Data["kind"] != push.KindLastPerson {
		t.Fatalf("pushes = %+v", msgs)
	}
	// A check-out without an open visit does nothing; the same night pushes only once.
	checkOut(seed.UserKai)
	checkIn(seed.UserKai)
	e.at(berlinAt(2026, 9, 30, 21, 40))
	checkOut(seed.UserKai)
	if len(e.fake.Sent()) != 1 {
		t.Errorf("second check-out of the night pushed again: %d", len(e.fake.Sent()))
	}
}

func TestCheckOutHookConditions(t *testing.T) {
	e := newEnv(t)
	presence.AfterCheckOut = func(ctx context.Context, stableID, userID string) { e.svc.OnCheckOut(ctx, stableID, userID) }
	t.Cleanup(func() { presence.AfterCheckOut = nil })

	// Afternoon: ordinary day, no reminder.
	e.at(berlinAt(2026, 9, 30, 14, 0))
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-in", nil).status(t, 200)
	e.at(berlinAt(2026, 9, 30, 16, 59))
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-out", nil).status(t, 200)
	// Morning after: no reminder either.
	e.at(berlinAt(2026, 10, 1, 7, 0))
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-in", nil).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-out", nil).status(t, 200)
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed outside the evening: %+v", e.fake.Sent())
	}
	// Everything done: no reminder.
	e.at(berlinAt(2026, 9, 30, 18, 0))
	for _, h := range e.allHorses() {
		e.setState(h, "2026-09-30", "checked", berlinAt(2026, 9, 30, 17, 0))
	}
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-in", nil).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/presence/check-out", nil).status(t, 200)
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed although everything is done: %+v", e.fake.Sent())
	}
	// From 17:00 the last person is reminded.
	e2 := newEnv(t)
	presence.AfterCheckOut = func(ctx context.Context, stableID, userID string) { e2.svc.OnCheckOut(ctx, stableID, userID) }
	e2.at(berlinAt(2026, 9, 30, 17, 0))
	e2.do(seed.UserTom, "POST", "/api/v1/presence/check-in", nil).status(t, 200)
	e2.do(seed.UserTom, "POST", "/api/v1/presence/check-out", nil).status(t, 200)
	if len(e2.fake.Sent()) != 1 {
		t.Errorf("no reminder at 17:00")
	}
}

func TestWeatherChangePush(t *testing.T) {
	e := newEnv(t)
	check := func(at time.Time) {
		t.Helper()
		e.at(at)
		if err := e.svc.CheckWeatherChange(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	// Forecast A: 8 °C, dry. Luna: no blanket. Balu: no blanket.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 18, 0), 8, false)
	// Nothing to compare yet, and no state yet.
	check(berlinAt(2026, 9, 30, 18, 5))
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "checked"}).status(t, 200)
	e.do(seed.UserTom, "POST", "/api/v1/horses/"+seed.HorseBalu+"/blanket-state", m{"action": "checked"}).status(t, 200)
	// Fanta has no state, so a change for her stays silent.
	e.at(berlinAt(2026, 9, 30, 19, 0))
	check(berlinAt(2026, 9, 30, 19, 10))
	if len(e.fake.Sent()) != 0 {
		t.Fatalf("pushed without change: %+v", e.fake.Sent())
	}

	// Forecast B: 3 °C with rain. Luna: 100 g (changed), Balu: none (unchanged), Fanta: 100 g but no state.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 19, 30), 3, true)
	check(berlinAt(2026, 9, 30, 19, 35))
	msgs := e.fake.Sent()
	got := sorted(recipients(e.fake))
	// Owner Jan and rider Mia of Luna.
	if want := sorted([]string{seed.UserJan, seed.UserMia}); !reflect.DeepEqual(got, want) {
		t.Fatalf("recipients = %v, want %v (%+v)", got, want, msgs)
	}
	if msgs[0].Data["kind"] != push.KindWeatherChange || msgs[0].Data["screen"] != "/horses/"+seed.HorseLuna+"/blanket-plan" ||
		!strings.Contains(msgs[0].Body, "Luna") || !strings.Contains(msgs[0].Body, "Decke 100 g") || !strings.Contains(msgs[0].Body, "keine Decke") {
		t.Errorf("push = %+v", msgs[0])
	}

	// Once per horse and night: a further change and repeated runs stay silent.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 20, 30), -2, false)
	check(berlinAt(2026, 9, 30, 20, 35))
	check(berlinAt(2026, 9, 30, 20, 40))
	if n := len(e.fake.Sent()); n != 2 {
		t.Errorf("pushes = %d, want 2", n)
	}
	// At -6 °C Balu needs his 150 g blanket: his owner gets the first push for him.
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 21, 30), -6, false)
	check(berlinAt(2026, 9, 30, 21, 35))
	if got := sorted(recipients(e.fake)); len(got) != 3 || !contains(got, seed.UserJonas) {
		t.Errorf("recipients after -6 °C = %v", got)
	}
}

func TestWeatherChangeIgnoresStatesSetBeforeAnyForecastAndOtherDays(t *testing.T) {
	e := newEnv(t)
	e.at(berlinAt(2026, 9, 30, 17, 0))
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "checked"}).status(t, 200)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 18, 0), 8, false)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 19, 0), 3, true)
	// Snapshots for tomorrow do not matter for tonight.
	e.snapshot("2026-10-01", berlinAt(2026, 9, 30, 19, 0), 20, false)
	e.snapshot("2026-10-01", berlinAt(2026, 9, 30, 20, 0), -10, false)
	e.at(berlinAt(2026, 9, 30, 20, 5))
	if err := e.svc.CheckWeatherChange(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 0 {
		t.Errorf("pushed: %+v", e.fake.Sent())
	}
}

func TestWeatherChangeOptOut(t *testing.T) {
	e := newEnv(t)
	e.exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'weather_change', false)`, seed.StableB, seed.UserMia)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 18, 0), 8, false)
	e.do(seed.UserMia, "POST", "/api/v1/horses/"+seed.HorseLuna+"/blanket-state", m{"action": "checked"}).status(t, 200)
	e.snapshot("2026-09-30", berlinAt(2026, 9, 30, 19, 30), 3, true)
	e.at(berlinAt(2026, 9, 30, 19, 35))
	if err := e.svc.CheckWeatherChange(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := recipients(e.fake); !reflect.DeepEqual(got, []string{seed.UserJan}) {
		t.Errorf("recipients = %v, want only the owner", got)
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
