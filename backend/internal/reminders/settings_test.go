package reminders_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/blankets"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/reminders"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

type notifications struct {
	Items       []reminders.KindInfo `json:"items"`
	PushConsent bool                 `json:"push_consent"`
}

func (n notifications) get(kind string) reminders.KindInfo {
	for _, it := range n.Items {
		if it.Kind == kind {
			return it
		}
	}
	return reminders.KindInfo{}
}

func TestNotificationSettings(t *testing.T) {
	e := newEnv(t)
	var n notifications
	e.do(seed.UserJan, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)

	// Every push kind is listed once, with German texts.
	var kinds []string
	for _, it := range n.Items {
		kinds = append(kinds, it.Kind)
		if it.Label == "" || it.Description == "" {
			t.Errorf("kind %s without label or description", it.Kind)
		}
		// Defaults follow the notifier: on, except for the opt-in kind.
		if it.Enabled != !push.OptIn(it.Kind) || it.OptIn != push.OptIn(it.Kind) {
			t.Errorf("default of %s = enabled %v, opt_in %v", it.Kind, it.Enabled, it.OptIn)
		}
	}
	want := push.Kinds()
	slices.Sort(kinds)
	slices.Sort(want)
	if !slices.Equal(kinds, want) {
		t.Errorf("kinds = %v, want %v", kinds, want)
	}
	if n.PushConsent {
		t.Error("push_consent is true without a consent")
	}

	// Opt-out kind: switch off and on.
	var it reminders.KindInfo
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/last_person", m{"enabled": false}).status(200).into(&it)
	if it.Kind != "last_person" || it.Enabled {
		t.Errorf("put response = %+v", it)
	}
	e.do(seed.UserJan, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)
	if n.get("last_person").Enabled || !n.get("helper").Enabled {
		t.Errorf("after switching off last_person: %+v", n.Items)
	}
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/last_person", m{"enabled": true}).status(200)
	e.do(seed.UserJan, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)
	if !n.get("last_person").Enabled {
		t.Error("last_person did not come back on")
	}

	// Opt-in kind: off without a row, on with one, shared with /requests/notify-settings.
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/new_request", m{"enabled": true}).status(200).into(&it)
	if !it.Enabled || !it.OptIn {
		t.Errorf("new_request = %+v", it)
	}
	var legacy struct {
		NewRequest bool `json:"new_request"`
	}
	e.do(seed.UserJan, "GET", "/api/v1/requests/notify-settings", nil).status(200).into(&legacy)
	if !legacy.NewRequest {
		t.Error("requests/notify-settings does not see the switch")
	}
	e.do(seed.UserJan, "PATCH", "/api/v1/requests/notify-settings", m{"new_request": false}).status(200)
	e.do(seed.UserJan, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)
	if n.get("new_request").Enabled {
		t.Error("switch off through requests/notify-settings is not visible")
	}
	// Other users are not affected.
	e.do(seed.UserAnna, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)
	if n.get("new_request").Enabled || !n.get("last_person").Enabled {
		t.Errorf("Anna's settings changed: %+v", n.Items)
	}

	// The notifier honors it.
	e.device(seed.UserJan)
	notifier := push.NewNotifier(e.pool, e.fake, nil)
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/helper", m{"enabled": false}).status(200)
	if err := notifier.NotifyUsers(t.Context(), seed.StableB, []string{seed.UserJan}, push.KindHelper, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) != 0 {
		t.Error("push sent although the kind is switched off")
	}

	// Consent state is reported.
	e.do(seed.UserJan, "GET", "/api/v1/settings/notifications", nil).status(200).into(&n)
	if !n.PushConsent {
		t.Error("push_consent false with a consent")
	}

	// Errors.
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/telepathy", m{"enabled": true}).errCode(404, "not_found")
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/helper", m{}).errCode(400, "validation_failed")
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/helper", m{"enabled": true, "x": 1}).status(400)
	e.do("", "GET", "/api/v1/settings/notifications", nil).errCode(401, "unauthorized")
	e.do("", "PUT", "/api/v1/settings/notifications/helper", m{"enabled": true}).errCode(401, "unauthorized")
}

func TestReminderTimeAdminOnly(t *testing.T) {
	e := newEnv(t)
	var got struct {
		ReminderTime string `json:"reminder_time"`
		CanEdit      bool   `json:"can_edit"`
	}
	e.do(seed.UserAnna, "GET", "/api/v1/stables/reminder-time", nil).status(200).into(&got)
	if got.ReminderTime != "20:30" || got.CanEdit {
		t.Errorf("member view = %+v", got)
	}
	e.do(seed.UserJan, "GET", "/api/v1/stables/reminder-time", nil).status(200).into(&got)
	if !got.CanEdit {
		t.Errorf("admin view = %+v", got)
	}

	// Non-admins are refused and nothing changes.
	e.do(seed.UserAnna, "PUT", "/api/v1/stables/reminder-time", m{"reminder_time": "19:45"}).errCode(403, "forbidden")
	e.do("", "PUT", "/api/v1/stables/reminder-time", m{"reminder_time": "19:45"}).errCode(401, "unauthorized")
	if s := e.scan(`SELECT to_char(reminder_time, 'HH24:MI') FROM stables WHERE id = $1`, seed.StableB); s != "20:30" {
		t.Fatalf("reminder time = %s after refused calls", s)
	}

	for _, bad := range []string{"", "8:30", "19:5", "25:00", "20:60", "15:59", "22:01", "23:00", "20:30:00", "abends"} {
		e.do(seed.UserJan, "PUT", "/api/v1/stables/reminder-time", m{"reminder_time": bad}).errCode(400, "validation_failed")
	}
	e.do(seed.UserJan, "PUT", "/api/v1/stables/reminder-time", m{}).errCode(400, "validation_failed")

	for _, good := range []string{"16:00", "22:00", "19:45"} {
		e.do(seed.UserJan, "PUT", "/api/v1/stables/reminder-time", m{"reminder_time": good}).status(200).into(&got)
		if got.ReminderTime != good || !got.CanEdit {
			t.Errorf("put %s = %+v", good, got)
		}
	}
	e.do(seed.UserAnna, "GET", "/api/v1/stables/reminder-time", nil).status(200).into(&got)
	if got.ReminderTime != "19:45" {
		t.Errorf("stored = %s", got.ReminderTime)
	}
	// The blanket screen and the reminder center show the new time.
	if out, _ := e.list(seed.UserAnna, ""); out.BlanketCheck.Time != "19:45" ||
		!out.BlanketCheck.DueAt.Equal(berlinAt(2026, time.September, 30, 19, 45)) {
		t.Errorf("blanket check = %+v", out.BlanketCheck)
	}
}

// The last-person job reads stables.reminder_time on every run, so a change applies at once.
func TestLastPersonJobUsesNewReminderTime(t *testing.T) {
	e := newEnv(t)
	deps := httpapi.Deps{Pool: e.pool, Now: e.clock, Notify: push.NewNotifier(e.pool, e.fake, nil)}
	svc := blankets.NewService(deps)
	for _, u := range []string{seed.UserJan, seed.UserAnna, seed.UserJonas, seed.UserTom, seed.UserSarah, seed.UserKai, seed.UserMia, seed.UserLea} {
		e.device(u)
	}

	e.at(berlinAt(2026, time.September, 30, 19, 45))
	if err := svc.RunLastPerson(t.Context()); err != nil {
		t.Fatal(err)
	}
	if n := len(e.fake.Sent()); n != 0 {
		t.Fatalf("%d pushes at 19:45 with the default reminder time 20:30", n)
	}

	e.do(seed.UserJan, "PUT", "/api/v1/stables/reminder-time", m{"reminder_time": "19:45"}).status(200)
	if err := svc.RunLastPerson(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(e.fake.Sent()) == 0 {
		t.Fatal("no last_person push at 19:45 after moving the reminder time to 19:45")
	}
	for _, msg := range e.fake.Sent() {
		if msg.Data["kind"] != push.KindLastPerson {
			t.Errorf("unexpected push %+v", msg)
		}
	}
}
