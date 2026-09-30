package reminders_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reminders"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func (e *env) planSlot(day, horse, activity, user string) {
	e.t.Helper()
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, $3::date, $4, $5, 'planned')`,
		seed.StableB, horse, day, user, activity)
}

func TestTrainingPlanEveningPush(t *testing.T) {
	e := newEnv(t)
	e.device(seed.UserJan)
	e.device(seed.UserAnna)
	e.device(seed.UserSarah)
	e.planSlot("2026-10-01", seed.HorseLuna, "hack", seed.UserJan)
	e.planSlot("2026-10-01", seed.HorseFanta, "hall", seed.UserJan)
	e.planSlot("2026-10-01", seed.HorseBalu, "lunge", seed.UserAnna)
	e.planSlot("2026-10-02", seed.HorseNala, "hall", seed.UserSarah) // not tomorrow
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, status) VALUES ($1, $2, '2026-10-01', $3, 'rest')`,
		seed.StableB, seed.HorseCookie, seed.UserSarah) // rest day: no plan
	e.do(seed.UserAnna, "PUT", "/api/v1/settings/notifications/training_plan", m{"enabled": false}).status(200)

	// Before 19:00 nothing happens.
	e.at(berlinAt(2026, time.September, 30, 18, 45))
	if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 0 || e.count(`SELECT count(*) FROM reminders`) != 0 {
		t.Fatal("training plan sent before 19:00")
	}

	e.at(berlinAt(2026, time.September, 30, 19, 5))
	if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
		t.Fatal(err)
	}
	sent := e.fake.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent %d pushes, want 1 (Jan; Anna switched the kind off, Sarah has no plan): %+v", len(sent), sent)
	}
	msg := sent[0]
	if msg.To != "tok-"+seed.UserJan || msg.Title != "Training morgen" || msg.Body != "Fanta: Halle, Luna: Ausritt" ||
		msg.Data["screen"] != "/training/week" || msg.Data["kind"] != push.KindTrainingPlan {
		t.Errorf("push = %+v", msg)
	}
	if n := e.count(`SELECT count(*) FROM reminders WHERE kind = 'training_plan' AND source_table = 'training_plan' AND sent_at IS NOT NULL`); n != 1 {
		t.Errorf("%d claim rows, want 1", n)
	}

	// Later runs of the same evening (also from a second process) send nothing again.
	e.at(berlinAt(2026, time.September, 30, 19, 20))
	e.fake.Reset()
	for range 2 {
		if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(e.fake.Sent()) != 0 {
		t.Errorf("repeated: %+v", e.fake.Sent())
	}

	// The center lists the sent reminder once (as the stored row, not additionally computed).
	_, groups := e.list(seed.UserJan, "")
	if len(groups["today"]) != 1 || groups["today"][0].Computed || groups["today"][0].Screen != "/training/week" {
		t.Errorf("today = %+v", groups["today"])
	}

	// After 22:00 a missed evening is not sent late.
	e.exec(`DELETE FROM reminders`)
	e.at(berlinAt(2026, time.September, 30, 22, 0))
	if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 0 {
		t.Error("sent at 22:00")
	}

	// The next evening plans the day after tomorrow.
	e.at(berlinAt(2026, time.October, 1, 19, 0))
	if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := e.fake.Sent(); len(got) != 1 || got[0].To != "tok-"+seed.UserSarah || got[0].Title != "Training morgen" {
		t.Errorf("next evening: %+v", got)
	}
}

type failingNotifier struct{}

func (failingNotifier) NotifyUsers(context.Context, string, []string, string, string, string, map[string]any) error {
	return errors.New("expo is down")
}

func TestTrainingPlanFailedPushReleasesTheClaim(t *testing.T) {
	e := newEnv(t)
	e.device(seed.UserJan)
	e.planSlot("2026-10-01", seed.HorseLuna, "hack", seed.UserJan)
	e.at(berlinAt(2026, time.September, 30, 19, 0))

	failing := &reminders.Service{Pool: e.pool, Notify: failingNotifier{}, Now: e.clock}
	if err := failing.RunTrainingPlan(context.Background()); err == nil {
		t.Fatal("expected the push error")
	}
	if n := e.count(`SELECT count(*) FROM reminders`); n != 0 {
		t.Fatalf("%d rows left after a failed push", n)
	}
	// The next run (working Expo) retries.
	if err := e.svc.RunTrainingPlan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 1 {
		t.Errorf("retry sent %d pushes", len(e.fake.Sent()))
	}
}

func TestTrainingPlanJobRegistered(t *testing.T) {
	e := newEnv(t)
	jobs := e.svc.Jobs()
	if len(jobs) != 1 || jobs[0].Name != "training-plan-reminders" || jobs[0].Run == nil || jobs[0].Schedule == nil {
		t.Errorf("jobs = %+v", jobs)
	}
}
