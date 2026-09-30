package reminders_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

func TestListStoredRowsAreGroupedAndFiltered(t *testing.T) {
	e := newEnv(t)
	item := e.scan(`INSERT INTO health_items (stable_id, horse_id, kind, label, due_date)
		VALUES ($1, $2, 'farrier', 'Hufschmied', '2026-10-07') RETURNING id::text`, seed.StableB, seed.HorseLuna)
	requestID := seed.HorseFanta // any uuid: reminders.source_id has no foreign key

	insert := func(user, kind, title, table, source string, due time.Time, sent *time.Time, dismissed *time.Time) string {
		return e.scan(`INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at, dismissed_at)
			VALUES ($1, $2, $3, $4, 'Text', $5, $6, $7::uuid, $8, $9) RETURNING id::text`,
			seed.StableB, user, kind, title, due, table, source, sent, dismissed)
	}
	at := func(d, hh, mm int) time.Time { return berlinAt(2026, time.September, d, hh, mm) }
	sent := func(t time.Time) *time.Time { return &t }

	health := insert(seed.UserJan, "health_due", "Fällig: Luna", "health_items", item, at(30, 8, 0), sent(at(30, 8, 1)), nil)
	night := insert(seed.UserJan, "last_person", "Decken heute Abend", "blanket_night", seed.StableB, at(30, 12, 0), sent(at(30, 18, 0)), nil)
	future := insert(seed.UserJan, "helper", "Erinnerung: Bewegen", "requests", requestID, berlinAt(2026, time.October, 2, 18, 0), nil, nil)
	insert(seed.UserJan, "helper", "Erledigt und weggewischt", "requests", seed.HorseBalu, at(30, 9, 0), sent(at(30, 9, 0)), sent(at(30, 10, 0)))
	insert(seed.UserJan, "helper", "Von gestern", "requests", seed.HorseCookie, at(29, 9, 0), sent(at(29, 9, 0)), nil)
	insert(seed.UserJan, "helper", "Zu weit weg", "requests", seed.HorseMerlin, berlinAt(2026, time.October, 8, 9, 0), nil, nil)
	insert(seed.UserAnna, "helper", "Nur für Anna", "requests", seed.HorsePepe, at(30, 9, 0), sent(at(30, 9, 0)), nil)

	out, groups := e.list(seed.UserJan, "")
	if out.Range != "week" || out.Today != "2026-09-30" || out.Timezone != "Europe/Berlin" {
		t.Errorf("response header = %+v", out)
	}
	if len(out.Groups) != 2 || out.Groups[0].Label != "Heute" || out.Groups[1].Label != "Diese Woche" {
		t.Fatalf("groups = %+v", out.Groups)
	}
	if got, want := titles(groups["today"]), []string{"Fällig: Luna", "Decken heute Abend"}; !slices.Equal(got, want) {
		t.Errorf("today = %v, want %v", got, want)
	}
	if got, want := titles(groups["week"]), []string{"Erinnerung: Bewegen"}; !slices.Equal(got, want) {
		t.Errorf("week = %v, want %v", got, want)
	}

	// Screens follow the deep link convention of the features (data.screen of the pushes).
	if it, _ := find(groups["today"], "Fällig: Luna"); it.ID != health || it.Screen != "/horses/"+seed.HorseLuna+"/health" ||
		!it.Dismissible || it.Computed || it.SentAt == nil || it.Body != "Text" {
		t.Errorf("health item = %+v", it)
	}
	if it, _ := find(groups["today"], "Decken heute Abend"); it.ID != night || it.Screen != "/blankets" {
		t.Errorf("blanket item = %+v", it)
	}
	if it, _ := find(groups["week"], "Erinnerung: Bewegen"); it.ID != future || it.Screen != "/requests/"+requestID || it.SentAt != nil {
		t.Errorf("request item = %+v", it)
	}

	// range=today returns the "Heute" group only.
	out, groups = e.list(seed.UserJan, "?range=today")
	if len(out.Groups) != 1 || out.Groups[0].Key != "today" || len(groups["today"]) != 2 {
		t.Errorf("range=today groups = %+v", out.Groups)
	}
	e.do(seed.UserJan, "GET", "/api/v1/reminders?range=month", nil).errCode(400, "validation_failed")
	e.do("", "GET", "/api/v1/reminders", nil).errCode(401, "unauthorized")
}

// insertHealth adds a health item and returns its id.
func (e *env) insertHealth(horse, kind, label, due, daily string) string {
	e.t.Helper()
	return e.scan(`INSERT INTO health_items (stable_id, horse_id, kind, label, due_date, daily_time)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::date, NULLIF($6, '')::time) RETURNING id::text`,
		seed.StableB, horse, kind, label, due, daily)
}

func TestListComputedItems(t *testing.T) {
	e := newEnv(t)

	// A horse Jan neither owns nor rides: nothing of it may show up for him.
	stranger := e.scan(`SELECT id::text FROM horses h WHERE stable_id = $1 AND owner_id <> $2
		AND NOT EXISTS (SELECT 1 FROM horse_riders r WHERE r.horse_id = h.id AND r.user_id = $2) ORDER BY name LIMIT 1`,
		seed.StableB, seed.UserJan)
	e.insertHealth(stranger, "farrier", "Fremder Hufschmied", "2026-10-01", "")
	e.insertHealth(stranger, "medication", "Fremdes Medikament", "", "07:00")

	e.insertHealth(seed.HorseLuna, "medication", "Equimax", "", "08:00")
	e.insertHealth(seed.HorseLuna, "medication", "Kur vorbei", "2026-09-20", "09:00")
	e.insertHealth(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-03", "")
	e.insertHealth(seed.HorseLuna, "vaccination", "Impfung", "2026-09-28", "")
	e.insertHealth(seed.HorseLuna, "dentist", "Zahnarzt", "2026-10-20", "")

	e.exec(`INSERT INTO reha_plans (stable_id, horse_id, diagnosis, vet, start_date, checkup_date, active)
		VALUES ($1, $2, 'Sehne', 'Dr. Weber', '2026-09-20', '2026-10-02', true)`, seed.StableB, seed.HorseLuna)
	e.exec(`INSERT INTO reha_plans (stable_id, horse_id, diagnosis, start_date, checkup_date, active)
		VALUES ($1, $2, 'Alt', '2026-01-01', '2026-10-02', false)`, seed.StableB, seed.HorseFanta)

	req := func(status, date, timeFrom string, helper string) string {
		id := e.scan(`INSERT INTO requests (stable_id, type, horse_id, created_by, date, time_from, location, status)
			VALUES ($1, 'exercise', $2, $3, $4::date, NULLIF($5, '')::time, 'Reitplatz', $6) RETURNING id::text`,
			seed.StableB, seed.HorseLuna, seed.UserAnna, date, timeFrom, status)
		e.exec(`INSERT INTO request_assignees (stable_id, request_id, user_id) VALUES ($1, $2, $3)`, seed.StableB, id, helper)
		return id
	}
	helping := req("assigned", "2026-10-01", "14:00", seed.UserJan)
	req("cancelled", "2026-10-01", "15:00", seed.UserJan)
	req("assigned", "2026-10-09", "15:00", seed.UserJan)
	req("assigned", "2026-10-01", "16:00", seed.UserSarah)

	slot := func(day, horse, activity, status, user string) {
		e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, $3::date, $4, $5, $6)`,
			seed.StableB, horse, day, user, activity, status)
	}
	slot("2026-10-01", seed.HorseLuna, "hack", "planned", seed.UserJan)
	slot("2026-10-03", seed.HorseLuna, "lunge", "planned", seed.UserJan)
	slot("2026-10-03", seed.HorseFanta, "hall", "planned", seed.UserJan)
	slot("2026-10-02", seed.HorseLuna, "hall", "done", seed.UserJan)
	slot("2026-10-01", seed.HorseFanta, "hall", "planned", seed.UserAnna)

	e.exec(`INSERT INTO blanket_states (stable_id, horse_id, day, action) VALUES ($1, $2, '2026-09-30', 'covered'), ($1, $3, '2026-09-30', 'checked'), ($1, $4, '2026-09-29', 'checked')`,
		seed.StableB, seed.HorseLuna, seed.HorseFanta, seed.HorseBalu)

	out, groups := e.list(seed.UserJan, "")

	wantToday := []string{"Fällig: Luna", "Medikament: Luna", "Training morgen"}
	if got := titles(groups["today"]); !slices.Equal(got, wantToday) {
		t.Errorf("today = %v, want %v", got, wantToday)
	}
	wantWeek := []string{"Bewegen: Luna", "Reha-Kontrolle: Luna", "Training am Samstag", "Fällig: Luna"}
	if got := titles(groups["week"]); !slices.Equal(got, wantWeek) {
		t.Errorf("week = %v, want %v", got, wantWeek)
	}

	today := groups["today"]
	if it := today[0]; it.Kind != "health_due" || it.Body != "Impfung ist seit 2 Tagen überfällig (28.09.2026)." ||
		!it.AllDay || !it.Computed || it.Dismissible || it.Screen != "/horses/"+seed.HorseLuna+"/health" {
		t.Errorf("overdue item = %+v", it)
	}
	if it := today[1]; it.Kind != "medication" || it.Body != "Equimax, 08:00 Uhr." || it.AllDay ||
		!it.DueAt.Equal(berlinAt(2026, time.September, 30, 8, 0)) {
		t.Errorf("medication item = %+v", it)
	}
	if it := today[2]; it.Kind != "training_plan" || it.Body != "Luna: Ausritt" || it.Screen != "/training/week" ||
		!it.DueAt.Equal(berlinAt(2026, time.September, 30, 19, 0)) {
		t.Errorf("training item = %+v", it)
	}
	week := groups["week"]
	if it := week[0]; it.Kind != "helper" || it.Body != "Du hilfst mit · Reitplatz" || it.Screen != "/requests/"+helping ||
		!it.DueAt.Equal(berlinAt(2026, time.October, 1, 14, 0)) || it.AllDay {
		t.Errorf("request item = %+v", it)
	}
	if it := week[1]; it.Kind != "reha_checkup" || it.Body != "Kontrolltermin bei Dr. Weber am 02.10.2026." ||
		it.Screen != "/horses/"+seed.HorseLuna+"/reha" || !it.AllDay {
		t.Errorf("reha item = %+v", it)
	}
	if it := week[2]; it.Body != "Fanta: Halle, Luna: Longe" {
		t.Errorf("training item of Saturday = %+v", it)
	}
	if it := week[3]; it.Body != "Hufschmied ist in 3 Tagen fällig (03.10.2026)." {
		t.Errorf("health item = %+v", it)
	}

	// Tonight's blanket check: 2 of 7 horses have a state for 2026-09-30, reminder at 20:30.
	bc := out.BlanketCheck
	if bc.Day != "2026-09-30" || bc.Time != "20:30" || bc.Done != 2 || bc.Total != 7 || bc.State != "upcoming" ||
		bc.Screen != "/blankets" || !bc.DueAt.Equal(berlinAt(2026, time.September, 30, 20, 30)) {
		t.Errorf("blanket check = %+v", bc)
	}
	e.at(berlinAt(2026, time.September, 30, 20, 45))
	if out, _ := e.list(seed.UserJan, ""); out.BlanketCheck.State != "due" {
		t.Errorf("state at 20:45 = %q, want due", out.BlanketCheck.State)
	}
	e.exec(`INSERT INTO blanket_states (stable_id, horse_id, day, action)
		SELECT stable_id, id, '2026-09-30', 'checked' FROM horses h WHERE NOT EXISTS
		(SELECT 1 FROM blanket_states s WHERE s.horse_id = h.id AND s.day = '2026-09-30')`)
	if out, _ := e.list(seed.UserJan, ""); out.BlanketCheck.State != "done" || out.BlanketCheck.Done != 7 {
		t.Errorf("all done: %+v", out.BlanketCheck)
	}

	// range=today: only what falls on today (the plan for tomorrow is due this evening).
	e.at(berlinAt(2026, time.September, 30, 19, 0))
	_, groups = e.list(seed.UserJan, "?range=today")
	if got := titles(groups["today"]); !slices.Equal(got, wantToday) || len(groups["week"]) != 0 {
		t.Errorf("range=today = %v / week %v", got, titles(groups["week"]))
	}

	// Somebody who neither owns nor rides Luna, helps, or has slots sees none of this.
	outsider := e.scan(`SELECT id::text FROM users u WHERE u.stable_id = $1 AND u.id NOT IN ($2, $3, $4)
		AND NOT EXISTS (SELECT 1 FROM horse_riders r WHERE r.horse_id = $5 AND r.user_id = u.id) ORDER BY name LIMIT 1`,
		seed.StableB, seed.UserJan, seed.UserAnna, seed.UserSarah, seed.HorseLuna)
	_, groups = e.list(outsider, "")
	for _, key := range []string{"today", "week"} {
		for _, it := range groups[key] {
			if it.Title == "Fällig: Luna" || it.Title == "Medikament: Luna" || it.Title == "Bewegen: Luna" || it.Title == "Training morgen" {
				t.Errorf("outsider sees %+v", it)
			}
		}
	}
}

func TestListDoesNotDuplicateSentReminders(t *testing.T) {
	e := newEnv(t)
	item := e.insertHealth(seed.HorseLuna, "medication", "Equimax", "", "08:00")
	// The medication push of 08:00 already created its row, so the computed item stays away.
	e.exec(`INSERT INTO reminders (stable_id, user_id, kind, title, body, due_at, source_table, source_id, sent_at)
		VALUES ($1, $2, 'medication', 'Medikament: Luna', 'Equimax, 08:00 Uhr.', $3, 'health_items', $4, $3)`,
		seed.StableB, seed.UserJan, berlinAt(2026, time.September, 30, 8, 0), item)

	_, groups := e.list(seed.UserJan, "")
	if got := titles(groups["today"]); !slices.Equal(got, []string{"Medikament: Luna"}) {
		t.Fatalf("today = %v", got)
	}
	if it := groups["today"][0]; it.Computed || !it.Dismissible {
		t.Errorf("expected the stored row, got %+v", it)
	}
}

func TestListTrainingPlanIsOptional(t *testing.T) {
	e := newEnv(t)
	e.exec(`INSERT INTO week_slots (stable_id, horse_id, day, user_id, activity, status) VALUES ($1, $2, '2026-10-01', $3, 'hack', 'planned')`,
		seed.StableB, seed.HorseLuna, seed.UserJan)
	if _, groups := e.list(seed.UserJan, ""); len(groups["today"]) != 1 {
		t.Fatalf("today = %v", titles(groups["today"]))
	}
	e.do(seed.UserJan, "PUT", "/api/v1/settings/notifications/training_plan", m{"enabled": false}).status(200)
	if _, groups := e.list(seed.UserJan, ""); len(groups["today"]) != 0 {
		t.Errorf("training plan listed although switched off: %v", titles(groups["today"]))
	}
}

func TestDismiss(t *testing.T) {
	e := newEnv(t)
	own := e.scan(`INSERT INTO reminders (stable_id, user_id, kind, title, due_at, sent_at) VALUES ($1, $2, 'helper', 'Meins', $3, $3) RETURNING id::text`,
		seed.StableB, seed.UserJan, berlinAt(2026, time.September, 30, 9, 0))
	other := e.scan(`INSERT INTO reminders (stable_id, user_id, kind, title, due_at, sent_at) VALUES ($1, $2, 'helper', 'Annas', $3, $3) RETURNING id::text`,
		seed.StableB, seed.UserAnna, berlinAt(2026, time.September, 30, 9, 0))

	e.do(seed.UserJan, "POST", "/api/v1/reminders/"+other+"/dismiss", nil).errCode(404, "not_found")
	e.do(seed.UserJan, "POST", "/api/v1/reminders/not-a-uuid/dismiss", nil).errCode(404, "not_found")
	e.do(seed.UserJan, "POST", "/api/v1/reminders/"+seed.HorseLuna+"/dismiss", nil).errCode(404, "not_found")
	e.do("", "POST", "/api/v1/reminders/"+own+"/dismiss", nil).errCode(401, "unauthorized")
	if n := e.count(`SELECT count(*) FROM reminders WHERE dismissed_at IS NOT NULL`); n != 0 {
		t.Fatalf("%d reminders dismissed by refused calls", n)
	}

	e.do(seed.UserJan, "POST", "/api/v1/reminders/"+own+"/dismiss", nil).status(204)
	first := e.scan(`SELECT dismissed_at::text FROM reminders WHERE id = $1`, own)
	e.at(e.clock().Add(time.Hour))
	e.do(seed.UserJan, "POST", "/api/v1/reminders/"+own+"/dismiss", nil).status(204) // idempotent
	if again := e.scan(`SELECT dismissed_at::text FROM reminders WHERE id = $1`, own); again != first {
		t.Errorf("dismissed_at changed from %s to %s", first, again)
	}
	if _, groups := e.list(seed.UserJan, ""); len(groups["today"]) != 0 {
		t.Errorf("dismissed reminder still listed: %v", titles(groups["today"]))
	}
	if _, groups := e.list(seed.UserAnna, ""); len(groups["today"]) != 1 {
		t.Errorf("Anna's reminder was affected: %v", titles(groups["today"]))
	}
}
