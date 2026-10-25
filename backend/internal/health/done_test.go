package health_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/health"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

// berlinAt is a wall-clock time in the stable's time zone (CEST in October 2026) as UTC.
func berlinAt(day, h, m int) time.Time {
	return time.Date(2026, 10, day, h, m, 0, 0, time.FixedZone("CEST", 2*3600)).UTC()
}

func donePath(id string) string { return "/api/v1/health-items/" + id + "/done" }

// item reads one item through the list endpoint of its horse.
func (e *env) item(horse, id string) health.Item {
	e.t.Helper()
	rec := e.call(seed.UserJan, "GET", "/api/v1/horses/"+horse+"/health", nil)
	e.expect(rec, 200)
	for _, it := range decode[health.Response](e.t, rec).Items {
		if it.ID == id {
			return it
		}
	}
	e.t.Fatalf("item %s not in the list of %s", id, horse)
	return health.Item{}
}

func str(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func (e *env) errorCode(rec *httptest.ResponseRecorder, status int, code string) {
	e.t.Helper()
	e.expect(rec, status)
	var b struct{ Error struct{ Code string } }
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.Error.Code != code {
		e.t.Fatalf("error code = %q (%v), want %q: %s", b.Error.Code, err, code, rec.Body)
	}
}

func TestDoneIsIdempotentPerDay(t *testing.T) {
	e := setup(t)
	id := e.addItem(seed.HorseFanta, "farrier", "Hufschmied", "2026-10-07", 42, "") // owner Anna, admin Jan

	rec := e.call(seed.UserAnna, "POST", donePath(id), nil)
	e.expect(rec, 200)
	it := decode[health.Item](t, rec)
	if str(it.DueDate) != "2026-11-18" || str(it.LastDoneOn) != "2026-10-01" || it.LastDoneBy == nil ||
		it.LastDoneBy.ID != seed.UserAnna || it.LastDoneBy.Name == "" {
		t.Fatalf("after the first done = %+v", it)
	}

	// Taps two and three (double tap, retry, a second phone) change nothing and still answer 200.
	for i, user := range []string{seed.UserAnna, seed.UserJan} {
		rec = e.call(user, "POST", donePath(id), nil)
		e.expect(rec, 200)
		if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-11-18" || it.LastDoneBy.ID != seed.UserAnna {
			t.Fatalf("repeat %d = %+v", i, it)
		}
	}

	// 23:30 local is still the same stable-local day ...
	e.now = berlinAt(1, 23, 30)
	rec = e.call(seed.UserAnna, "POST", donePath(id), nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-11-18" {
		t.Fatalf("late the same day = %+v", it)
	}
	// ... and 00:30 local, which is still Oct 1 in UTC, is the next one: it advances again.
	e.now = berlinAt(2, 0, 30)
	rec = e.call(seed.UserAnna, "POST", donePath(id), nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-12-30" || str(it.LastDoneOn) != "2026-10-02" {
		t.Fatalf("next day = %+v", it)
	}
}

func TestConcurrentDoneAdvancesOnce(t *testing.T) {
	e := setup(t)
	id := e.addItem(seed.HorseLuna, "deworming", "Wurmkur", "2026-06-01", 90, "")
	const taps = 8
	tokens := make([]string, taps)
	for i := range tokens {
		tokens[i] = authtest.TokenAt(t, e.pool, seed.UserJan, e.now)
	}
	codes := make([]int, taps)
	var wg sync.WaitGroup
	for i := 0; i < taps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", donePath(id), nil)
			req.Header.Set("Authorization", "Bearer "+tokens[i])
			rec := httptest.NewRecorder()
			e.h.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}()
	}
	wg.Wait()
	for i, c := range codes {
		if c != 200 {
			t.Errorf("tap %d: status %d", i, c)
		}
	}
	// 2026-06-01 + 90 days = 08-30, + 90 = 11-28 is the first date after today: one advance, not eight.
	if it := e.item(seed.HorseLuna, id); str(it.DueDate) != "2026-11-28" {
		t.Fatalf("due after %d concurrent taps = %s, want 2026-11-28", taps, str(it.DueDate))
	}
}

func TestDoneChecksTheDueDateTheAppSaw(t *testing.T) {
	e := setup(t)
	id := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-07", 42, "")
	post := func(body any) *httptest.ResponseRecorder { return e.call(seed.UserJan, "POST", donePath(id), body) }

	// Somebody moved the date since the app loaded it: nothing is done.
	e.errorCode(post(map[string]any{"due_date": "2026-10-06"}), 409, "due_date_changed")
	if it := e.item(seed.HorseLuna, id); str(it.DueDate) != "2026-10-07" || it.LastDoneOn != nil {
		t.Fatalf("after a refused done = %+v", it)
	}
	e.errorCode(post(map[string]any{"due_date": ""}), 409, "due_date_changed") // the app saw no date
	e.errorCode(post(map[string]any{"due_date": "07.10.2026"}), 400, "validation_failed")
	e.errorCode(post(map[string]any{"when": "now"}), 400, "invalid_json")

	e.expect(post(map[string]any{"due_date": "2026-10-07"}), 200)
	// The second tap of the same app still carries the old date: it is a repeat, not a conflict.
	rec := post(map[string]any{"due_date": "2026-10-07"})
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-11-18" {
		t.Fatalf("repeat with the old date = %+v", it)
	}

	undated := e.addItem(seed.HorseLuna, "physio", "Physio", "", 28, "")
	e.expect(e.call(seed.UserJan, "POST", donePath(undated), map[string]any{"due_date": ""}), 200)
}

func TestUndoDone(t *testing.T) {
	e := setup(t)
	overdue := e.addItem(seed.HorseLuna, "deworming", "Wurmkur", "2026-06-01", 90, "")
	path := donePath(overdue)

	// Nothing done yet: nothing to undo, the item stays as it is.
	rec := e.call(seed.UserJan, "DELETE", path, nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-06-01" {
		t.Fatalf("undo without done = %+v", it)
	}

	e.expect(e.call(seed.UserJan, "POST", path, nil), 200)
	e.expect(e.call(seed.UserMia, "DELETE", path, nil), 403) // riders do not manage health items
	e.expect(e.call(seed.UserTom, "DELETE", path, nil), 403)
	e.expect(e.call(otherUser, "DELETE", path, nil), 404)
	e.expect(e.call(seed.UserJan, "DELETE", donePath("nope"), nil), 404)
	e.expect(e.call("", "DELETE", path, nil), 401)

	rec = e.call(seed.UserJan, "DELETE", path, nil)
	e.expect(rec, 200)
	it := decode[health.Item](t, rec)
	if str(it.DueDate) != "2026-06-01" || it.LastDoneOn != nil || it.LastDoneBy != nil || *it.DaysUntilDue != -122 {
		t.Fatalf("after the undo = %+v", it)
	}
	// A repeated undo is fine.
	e.expect(e.call(seed.UserJan, "DELETE", path, nil), 200)
	// Done again after the undo advances once.
	rec = e.call(seed.UserJan, "POST", path, nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-11-28" {
		t.Fatalf("done after undo = %+v", it)
	}

	// A done of an earlier day stays.
	e.now = berlinAt(2, 9, 0)
	e.errorCode(e.call(seed.UserJan, "DELETE", path, nil), 409, "undo_expired")
	if it := e.item(seed.HorseLuna, overdue); str(it.DueDate) != "2026-11-28" || str(it.LastDoneOn) != "2026-10-01" {
		t.Fatalf("after the refused undo = %+v", it)
	}

	// A one-off appointment gets its date back, an item that had none goes back to none.
	e.now = fixedNow
	oneOff := e.addItem(seed.HorseLuna, "vet", "Kontrolle", "2026-10-05", 0, "")
	e.expect(e.call(seed.UserJan, "POST", donePath(oneOff), nil), 200)
	if it := e.item(seed.HorseLuna, oneOff); it.DueDate != nil {
		t.Fatalf("one-off after done = %+v", it)
	}
	e.expect(e.call(seed.UserJan, "DELETE", donePath(oneOff), nil), 200)
	if it := e.item(seed.HorseLuna, oneOff); str(it.DueDate) != "2026-10-05" {
		t.Fatalf("one-off after undo = %+v", it)
	}
	undated := e.addItem(seed.HorseLuna, "physio", "Physio", "", 28, "")
	e.expect(e.call(seed.UserJan, "POST", donePath(undated), nil), 200)
	if it := e.item(seed.HorseLuna, undated); str(it.DueDate) != "2026-10-29" {
		t.Fatalf("undated after done = %+v", it)
	}
	e.expect(e.call(seed.UserJan, "DELETE", donePath(undated), nil), 200)
	if it := e.item(seed.HorseLuna, undated); it.DueDate != nil || it.LastDoneOn != nil {
		t.Fatalf("undated after undo = %+v", it)
	}
}

func TestDoneEndsAMedicationCourse(t *testing.T) {
	e := setup(t)
	course := e.addItem(seed.HorseFanta, "medication", "Equioxx", "2026-10-06", 0, "08:00") // ends in 5 days
	openEnded := e.addItem(seed.HorseFanta, "medication", "Pergolid", "", 0, "08:00")
	ended := e.addItem(seed.HorseFanta, "medication", "Kur", "2026-09-30", 0, "09:00")
	noTime := e.addItem(seed.HorseFanta, "medication", "Salbe", "2026-10-09", 0, "") // no daily reminder, same course rule

	rec := e.call(seed.UserAnna, "POST", donePath(course), nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-10-01" || str(it.LastDoneOn) != "2026-10-01" || it.DailyTime == nil {
		t.Fatalf("course after done = %+v", it)
	}
	// The end date is not pushed out again, however often it is tapped.
	e.expect(e.call(seed.UserAnna, "POST", donePath(course), nil), 200)
	if it := e.item(seed.HorseFanta, course); str(it.DueDate) != "2026-10-01" {
		t.Fatalf("course after the repeat = %+v", it)
	}
	e.expect(e.call(seed.UserAnna, "DELETE", donePath(course), nil), 200)
	if it := e.item(seed.HorseFanta, course); str(it.DueDate) != "2026-10-06" || it.LastDoneOn != nil {
		t.Fatalf("course after the undo = %+v", it)
	}

	// Without an end date the date stays empty (as before), but once per day.
	e.expect(e.call(seed.UserAnna, "POST", donePath(openEnded), nil), 200)
	if it := e.item(seed.HorseFanta, openEnded); it.DueDate != nil || str(it.LastDoneOn) != "2026-10-01" {
		t.Fatalf("open-ended after done = %+v", it)
	}
	// A course that is over keeps its end date.
	e.expect(e.call(seed.UserAnna, "POST", donePath(ended), nil), 200)
	if it := e.item(seed.HorseFanta, ended); str(it.DueDate) != "2026-09-30" {
		t.Fatalf("ended course after done = %+v", it)
	}
	e.expect(e.call(seed.UserAnna, "POST", donePath(noTime), nil), 200)
	if it := e.item(seed.HorseFanta, noTime); str(it.DueDate) != "2026-10-01" {
		t.Fatalf("medication without time after done = %+v", it)
	}
}

func TestMedicationRemindersStopWhenTheCourseIsEnded(t *testing.T) {
	e := setup(t)
	rec := &recorder{}
	job := newJob(e, rec)
	ctx := context.Background()
	id := e.addItem(seed.HorseFanta, "medication", "Equioxx", "2026-10-06", 0, "08:00")
	run := func(at time.Time) []call {
		t.Helper()
		e.now = at
		if err := job.Run(ctx); err != nil {
			t.Fatal(err)
		}
		return rec.take()
	}

	// 07:30: the course is ended before today's reminder is sent: no reminder today.
	e.now = berlinAt(1, 7, 30)
	e.expect(e.call(seed.UserAnna, "POST", donePath(id), nil), 200)
	if c := run(berlinAt(1, 8, 0)); len(c) != 0 {
		t.Fatalf("after the course was ended: %+v", c)
	}
	// Undone: the reminder goes out again.
	e.now = berlinAt(1, 8, 5)
	e.expect(e.call(seed.UserAnna, "DELETE", donePath(id), nil), 200)
	if c := run(berlinAt(1, 8, 10)); len(c) != 1 {
		t.Fatalf("after the undo: %+v", c)
	}
	// Ended for good (after today's reminder): nothing on the following days.
	e.now = berlinAt(1, 9, 0)
	e.expect(e.call(seed.UserAnna, "POST", donePath(id), nil), 200)
	for _, day := range []int{2, 3, 5} {
		if c := run(berlinAt(day, 8, 5)); len(c) != 0 {
			t.Fatalf("day %d after the course was ended: %+v", day, c)
		}
	}
}

func TestPatchKeepsOrClearsTheDoneMarker(t *testing.T) {
	e := setup(t)
	id := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-07", 42, "")
	item := "/api/v1/health-items/" + id
	e.expect(e.call(seed.UserJan, "POST", donePath(id), nil), 200)

	// Editing the label keeps "erledigt", and so does saving the sheet with unchanged values.
	e.expect(e.call(seed.UserJan, "PATCH", item, map[string]any{"label": "Hufschmied Kruse"}), 200)
	e.expect(e.call(seed.UserJan, "PATCH", item, map[string]any{
		"kind": "farrier", "label": "Hufschmied Kruse", "due_date": "2026-11-18", "interval_count": 42, "interval_unit": "day",
		"remind_days": []int{7, 1}, "remind_time": "08:00", "note": "", "daily_time": ""}), 200)
	if it := e.item(seed.HorseLuna, id); str(it.LastDoneOn) != "2026-10-01" || str(it.DueDate) != "2026-11-18" {
		t.Fatalf("after edits without a new date = %+v", it)
	}

	// A new due date is a fresh start: no "erledigt" today, nothing for an undo to restore.
	rec := e.call(seed.UserJan, "PATCH", item, map[string]any{"due_date": "2026-11-20"})
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); it.LastDoneOn != nil || it.LastDoneBy != nil || str(it.DueDate) != "2026-11-20" {
		t.Fatalf("after moving the date = %+v", it)
	}
	rec = e.call(seed.UserJan, "DELETE", donePath(id), nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2026-11-20" {
		t.Fatalf("undo after the hand edit = %+v", it)
	}
	// Done again counts from the new date.
	rec = e.call(seed.UserJan, "POST", donePath(id), nil)
	e.expect(rec, 200)
	if it := decode[health.Item](t, rec); str(it.DueDate) != "2027-01-01" {
		t.Fatalf("done after the hand edit = %+v", it)
	}
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%v\n%s", err, sql)
	}
}

// storedReminders counts the reminder rows of an item that are still listed (not dismissed).
func (e *env) storedReminders(itemID string) int {
	e.t.Helper()
	var n int
	err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM reminders
		WHERE source_table = 'health_items' AND source_id = $1 AND dismissed_at IS NULL`, itemID).Scan(&n)
	if err != nil {
		e.t.Fatal(err)
	}
	return n
}

func TestStoredRemindersAreDismissedWithTheItem(t *testing.T) {
	e := setup(t)
	job := newJob(e, &recorder{})
	ctx := context.Background()
	// Both are due in 7 days: the job sends "Fällig: Luna" to Jan and Mia for each, 2 rows per item.
	done := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-08", 42, "")
	other := e.addItem(seed.HorseLuna, "vaccination", "Influenza", "2026-10-08", 365, "")
	if err := job.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if e.storedReminders(done) != 2 || e.storedReminders(other) != 2 {
		t.Fatalf("stored rows: done %d, other %d, want 2 and 2", e.storedReminders(done), e.storedReminders(other))
	}

	e.expect(e.call(seed.UserJan, "POST", donePath(done), nil), 200)
	if e.storedReminders(done) != 0 || e.storedReminders(other) != 2 {
		t.Errorf("after done: done %d, other %d, want 0 and 2", e.storedReminders(done), e.storedReminders(other))
	}

	// An edit that only touches the label leaves the reminders alone; one that moves the date,
	// changes the interval or the reminder days dismisses them.
	cases := []struct {
		name    string
		body    map[string]any
		dismiss bool
	}{
		{"label", map[string]any{"label": "Influenza Herbst"}, false},
		{"same values", map[string]any{"due_date": "2026-10-08", "interval_count": 365, "interval_unit": "day", "remind_days": []int{7, 1}}, false},
		{"note", map[string]any{"note": "Vormittags"}, false},
		{"due date", map[string]any{"due_date": "2026-10-09"}, true},
	}
	reset := func() {
		e.exec(`UPDATE reminders SET dismissed_at = NULL WHERE source_id = $1`, other)
		e.exec(`UPDATE health_items SET label = 'Influenza', due_date = '2026-10-08', interval_count = 365, interval_unit = 'day',
			note = NULL, remind_days = '{7,1}', remind_time = '08:00' WHERE id = $1`, other)
	}
	for _, c := range cases {
		reset()
		e.expect(e.call(seed.UserJan, "PATCH", "/api/v1/health-items/"+other, c.body), 200)
		if got := e.storedReminders(other); (got == 0) != c.dismiss {
			t.Errorf("patch %s: %d stored rows left, dismissed = %v", c.name, got, c.dismiss)
		}
	}
	for _, body := range []map[string]any{{"interval_count": 6, "interval_unit": "month"}, {"remind_days": []int{1}}, {"remind_time": "18:00"}} {
		reset()
		e.expect(e.call(seed.UserJan, "PATCH", "/api/v1/health-items/"+other, body), 200)
		if got := e.storedReminders(other); got != 0 {
			t.Errorf("patch %v: %d stored rows left", body, got)
		}
	}

	// Deleting the item dismisses its rows too (they have no foreign key and would stay as orphans).
	e.exec(`UPDATE reminders SET dismissed_at = NULL WHERE source_id = $1`, other)
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/health-items/"+other, nil), 204)
	if e.storedReminders(other) != 0 {
		t.Errorf("after delete: %d stored rows left", e.storedReminders(other))
	}
	var rows int
	if err := e.pool.QueryRow(ctx, `SELECT count(*) FROM reminders WHERE source_id = $1`, other).Scan(&rows); err != nil || rows != 2 {
		t.Errorf("rows kept as history = %d (%v), want 2", rows, err)
	}
}

func TestUndoBringsBackTheRemindersDoneDismissed(t *testing.T) {
	e := setup(t)
	job := newJob(e, &recorder{})
	id := e.addItem(seed.HorseLuna, "farrier", "Hufschmied", "2026-10-08", 42, "")
	if err := job.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Mia dismissed her reminder in the reminder center earlier; Jan's is still listed.
	e.exec(`UPDATE reminders SET dismissed_at = $2 WHERE source_id = $1 AND user_id = $3`, id, fixedNow.Add(-time.Hour), seed.UserMia)
	if n := e.storedReminders(id); n != 1 {
		t.Fatalf("listed before done = %d, want 1", n)
	}
	e.expect(e.call(seed.UserJan, "POST", donePath(id), nil), 200)
	if n := e.storedReminders(id); n != 0 {
		t.Fatalf("listed after done = %d, want 0", n)
	}
	e.expect(e.call(seed.UserJan, "DELETE", donePath(id), nil), 200)
	if n := e.storedReminders(id); n != 1 {
		t.Fatalf("listed after the undo = %d, want only Jan's row back", n)
	}
	var user string
	if err := e.pool.QueryRow(context.Background(), `SELECT user_id::text FROM reminders
		WHERE source_id = $1 AND dismissed_at IS NULL`, id).Scan(&user); err != nil || user != seed.UserJan {
		t.Fatalf("restored row belongs to %q (%v), want Jan", user, err)
	}
}

func TestHealthChangedEvents(t *testing.T) {
	e := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	hub := realtime.NewHub(e.pool, nil)
	finished := make(chan struct{})
	go func() { hub.Run(ctx); close(finished) }()
	t.Cleanup(func() { cancel(); <-finished })
	select {
	case <-hub.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("hub did not start listening")
	}
	sub, unsubscribe := hub.Subscribe(seed.StableB)
	defer unsubscribe()
	other, unsubscribeOther := hub.Subscribe(otherStable)
	defer unsubscribeOther()

	expect := func(what, horse string) {
		t.Helper()
		select {
		case ev := <-sub:
			var data map[string]any
			_ = json.Unmarshal(ev.Data, &data)
			if ev.Type != health.EventChanged || data["horse_id"] != horse || len(data) != 1 {
				t.Fatalf("%s: event = %+v", what, ev)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: no event", what)
		}
	}
	quiet := func(what string) {
		t.Helper()
		select {
		case ev := <-sub:
			t.Fatalf("%s: unexpected event %+v", what, ev)
		case ev := <-other:
			t.Fatalf("%s: event leaked to another stable: %+v", what, ev)
		case <-time.After(300 * time.Millisecond):
		}
	}

	rec := e.call(seed.UserJan, "POST", "/api/v1/horses/"+seed.HorseLuna+"/health-items",
		map[string]any{"kind": "farrier", "label": "Hufschmied", "due_date": "2026-10-07", "interval_count": 42})
	e.expect(rec, 201)
	id := decode[health.Item](t, rec).ID
	expect("create", seed.HorseLuna)

	e.expect(e.call(seed.UserJan, "PATCH", "/api/v1/health-items/"+id, map[string]any{"note": "x"}), 200)
	expect("patch", seed.HorseLuna)
	e.expect(e.call(seed.UserJan, "POST", donePath(id), nil), 200)
	expect("done", seed.HorseLuna)
	// A repeat changes nothing and says nothing; neither do refused or failed requests.
	e.expect(e.call(seed.UserJan, "POST", donePath(id), nil), 200)
	e.expect(e.call(seed.UserMia, "POST", donePath(id), nil), 403)
	e.expect(e.call(seed.UserJan, "POST", donePath(id), map[string]any{"due_date": "2000-01-01"}), 200) // a repeat, no conflict
	quiet("repeat and refused")
	e.expect(e.call(seed.UserJan, "DELETE", donePath(id), nil), 200)
	expect("undo", seed.HorseLuna)
	e.expect(e.call(seed.UserJan, "DELETE", donePath(id), nil), 200) // nothing left to undo
	quiet("repeated undo")
	e.expect(e.call(seed.UserJan, "DELETE", "/api/v1/health-items/"+id, nil), 204)
	expect("delete", seed.HorseLuna)
	quiet("end")
}

func TestDueAfterDone(t *testing.T) {
	d := func(s string) *time.Time {
		v, err := time.Parse("2006-01-02", s)
		if err != nil {
			t.Fatal(err)
		}
		return &v
	}
	today := *d("2026-10-01")
	iv := &health.Interval{Count: 42, Unit: "day"}
	cases := []struct {
		name   string
		due    *time.Time
		iv     *health.Interval
		course bool
		want   string
	}{
		{"repeating item follows the interval", d("2026-10-07"), iv, false, "2026-11-18"},
		{"one-off item is cleared", d("2026-10-07"), nil, false, ""},
		{"course ends today", d("2026-10-06"), nil, true, "2026-10-01"},
		{"course ending today stays", d("2026-10-01"), nil, true, "2026-10-01"},
		{"course that is over keeps its end", d("2026-09-30"), nil, true, "2026-09-30"},
		{"course without end stays open", nil, nil, true, ""},
		{"an interval does not extend a course", d("2026-10-06"), iv, true, "2026-10-01"},
	}
	for _, c := range cases {
		got := health.DueAfterDone(c.due, c.iv, c.course, today)
		switch {
		case c.want == "" && got != nil:
			t.Errorf("%s: got %v, want nil", c.name, got)
		case c.want != "" && (got == nil || got.Format("2006-01-02") != c.want):
			t.Errorf("%s: got %v, want %s", c.name, got, c.want)
		}
	}
}
