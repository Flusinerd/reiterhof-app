package observations_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files/filestest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/observations"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push/pushtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/realtime"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	otherStable = "00000000-0000-4000-8000-0000000009a1"
	otherUser   = "00000000-0000-4000-8000-0000000009a2"
	otherHorse  = "00000000-0000-4000-8000-0000000009a3"
)

var (
	pngBytes = filestest.PNG()
	pdfBytes = filestest.PDF()
)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
	fake *push.Fake
	hub  *realtime.Hub
}

func setup(t *testing.T) *env {
	t.Helper()
	t.Cleanup(files.SetDir(t.TempDir()))
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	for _, q := range []string{
		// Start from a clean slate; the seed has a few observations (and a reha plan may refer to one).
		`UPDATE reha_plans SET observation_id = NULL`,
		`DELETE FROM observations`,
		`INSERT INTO stables (id, name) VALUES ('` + otherStable + `', 'Anderer Stall')`,
		`INSERT INTO users (id, stable_id, name, email) VALUES ('` + otherUser + `', '` + otherStable + `', 'Fremd', 'fremd@example.org')`,
		`INSERT INTO horses (id, stable_id, name, owner_id) VALUES ('` + otherHorse + `', '` + otherStable + `', 'Fremdpferd', '` + otherUser + `')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	hub := realtime.NewHub(pool, nil)
	hubCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { hub.Run(hubCtx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-hub.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("hub did not start listening")
	}
	fake := &push.Fake{}
	e := &env{t: t, pool: pool, fake: fake, hub: hub}
	e.h = httpapi.NewHandler(httpapi.Deps{Pool: pool, Notify: push.NewNotifier(pool, fake, nil), Events: hub})
	return e
}

func (e *env) call(user, method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if user != "" {
		authtest.Authorize(e.t, e.pool, req, user)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) expect(rec *httptest.ResponseRecorder, want int) {
	e.t.Helper()
	if rec.Code != want {
		e.t.Fatalf("status = %d, want %d: %s", rec.Code, want, rec.Body)
	}
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatal(err)
	}
}

// tokens registers a push token "tok-<name>" for every user and returns the token -> name map.
func (e *env) tokens() map[string]string {
	e.t.Helper()
	names := map[string]string{
		seed.UserJan: "jan", seed.UserAnna: "anna", seed.UserJonas: "jonas", seed.UserTom: "tom",
		seed.UserSarah: "sarah", seed.UserKai: "kai", seed.UserMia: "mia", seed.UserLea: "lea",
	}
	out := map[string]string{}
	for id, name := range names {
		tok := "ExponentPushToken[" + name + "]"
		if err := push.RegisterToken(context.Background(), e.pool, seed.StableB, id, tok, "android"); err != nil {
			e.t.Fatal(err)
		}
		pushtest.GrantConsent(e.t, e.pool, id)
		out[tok] = name
	}
	return out
}

// recipients returns the sorted names the fake sender received messages for.
func (e *env) recipients(names map[string]string) []string {
	var out []string
	for _, m := range e.fake.Sent() {
		out = append(out, names[m.To])
	}
	slices.Sort(out)
	return out
}

func (e *env) present(user string) {
	e.t.Helper()
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, now() - interval '1 hour')`, seed.StableB, user)
}

func (e *env) photo(stableID string) string {
	e.t.Helper()
	saved, err := files.Save(context.Background(), stableID, bytes.NewReader(pngBytes), "image/png")
	if err != nil {
		e.t.Fatal(err)
	}
	return saved.Path
}

func report(horse string, extra map[string]any) map[string]any {
	m := map[string]any{"horse_id": horse, "category": "cough"}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestReportByAnyMember(t *testing.T) {
	e := setup(t)
	photo := e.photo(seed.StableB)
	// Tom has nothing to do with Luna; every member may report.
	rec := e.call(seed.UserTom, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{
		"body_part": "head", "description": "  Hustet seit heute Morgen.  ", "media": []string{photo}, "urgency": "check",
	}))
	e.expect(rec, 201)
	resp := decode[observations.CreateResponse](t, rec)
	o := resp.Observation
	if o.HorseID != seed.HorseLuna || o.HorseName != "Luna" || o.Reporter.ID != seed.UserTom || o.Reporter.Name != "Tom" {
		t.Errorf("observation = %+v", o)
	}
	if *o.Category != "cough" || *o.BodyPart != "head" || *o.Description != "Hustet seit heute Morgen." {
		t.Errorf("fields = %v %v %v", *o.Category, *o.BodyPart, *o.Description)
	}
	if o.Urgency != "check" || o.Status != "watch" || !o.CanChange {
		t.Errorf("urgency/status/can_change = %s %s %v", o.Urgency, o.Status, o.CanChange)
	}
	if len(o.Media) != 1 || o.Media[0].Path != photo || o.Media[0].ContentType != "image/png" || o.Media[0].URL != files.URLFor(photo) {
		t.Errorf("media = %+v", o.Media)
	}
	if resp.Emergency != nil {
		t.Error("a non-urgent report must not carry the emergency card")
	}

	// Defaults: urgency info, no body part, no description, no media.
	rec = e.call(seed.UserKai, "POST", "/api/v1/observations", report(seed.HorseLuna, nil))
	e.expect(rec, 201)
	o = decode[observations.CreateResponse](t, rec).Observation
	if o.Urgency != "info" || o.BodyPart != nil || o.Description != nil || o.Media == nil || len(o.Media) != 0 {
		t.Errorf("defaults = %+v", o)
	}
}

func TestReportPermissionsAndIsolation(t *testing.T) {
	e := setup(t)
	e.expect(e.call("", "POST", "/api/v1/observations", report(seed.HorseLuna, nil)), 401)
	// A horse of another stable is not found, in both directions.
	e.expect(e.call(seed.UserMia, "POST", "/api/v1/observations", report(otherHorse, nil)), 404)
	e.expect(e.call(otherUser, "POST", "/api/v1/observations", report(seed.HorseLuna, nil)), 404)
	e.expect(e.call(seed.UserMia, "POST", "/api/v1/observations", report("not-a-uuid", nil)), 404)
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM observations`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("rows = %d, %v", n, err)
	}

	// Read and change across stables.
	rec := e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseLuna, nil))
	e.expect(rec, 201)
	id := decode[observations.CreateResponse](t, rec).Observation.ID
	e.expect(e.call(otherUser, "GET", "/api/v1/observations/"+id, nil), 404)
	e.expect(e.call(otherUser, "GET", "/api/v1/horses/"+seed.HorseLuna+"/observations", nil), 404)
	e.expect(e.call(otherUser, "PATCH", "/api/v1/observations/"+id, map[string]any{"status": "done"}), 404)
	e.expect(e.call(seed.UserMia, "GET", "/api/v1/horses/"+otherHorse+"/observations", nil), 404)
	e.expect(e.call("", "GET", "/api/v1/observations/"+id, nil), 401)
}

func TestReportValidation(t *testing.T) {
	e := setup(t)
	otherPhoto := e.photo(otherStable)
	pdf, err := files.Save(context.Background(), seed.StableB, bytes.NewReader(pdfBytes), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	own := e.photo(seed.StableB)
	tooMany := make([]string, 7)
	for i := range tooMany {
		tooMany[i] = e.photo(seed.StableB)
	}
	cases := []struct {
		name string
		body map[string]any
	}{
		{"missing horse", map[string]any{"category": "cough"}},
		{"missing category", map[string]any{"horse_id": seed.HorseLuna}},
		{"unknown category", report(seed.HorseLuna, map[string]any{"category": "flu"})},
		{"unknown body part", report(seed.HorseLuna, map[string]any{"body_part": "tail"})},
		{"unknown urgency", report(seed.HorseLuna, map[string]any{"urgency": "panic"})},
		{"description too long", report(seed.HorseLuna, map[string]any{"description": strings.Repeat("a", 2001)})},
		{"path of another stable", report(seed.HorseLuna, map[string]any{"media": []string{otherPhoto}})},
		{"path traversal", report(seed.HorseLuna, map[string]any{"media": []string{"../etc/passwd"}})},
		{"pdf is not a photo", report(seed.HorseLuna, map[string]any{"media": []string{pdf.Path}})},
		{"duplicate media", report(seed.HorseLuna, map[string]any{"media": []string{own, own}})},
		{"too many photos", report(seed.HorseLuna, map[string]any{"media": tooMany})},
		{"unknown field", report(seed.HorseLuna, map[string]any{"status": "done"})},
	}
	for _, c := range cases {
		rec := e.call(seed.UserMia, "POST", "/api/v1/observations", c.body)
		if rec.Code != 400 {
			t.Errorf("%s: status = %d, want 400: %s", c.name, rec.Code, rec.Body)
		}
	}
	// Every allowed value is accepted.
	for _, cat := range observations.Categories {
		e.expect(e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"category": cat})), 201)
	}
	for _, bp := range observations.BodyParts {
		e.expect(e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"body_part": bp})), 201)
	}
}

func TestNotificationRecipients(t *testing.T) {
	e := setup(t)
	names := e.tokens()

	// Non-urgent: owner (Jan) and rider (Mia) of Luna, never the reporter; Tom is present but
	// not involved, so he is not notified.
	e.present(seed.UserTom)
	rec1 := e.call(seed.UserKai, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"urgency": "check"}))
	e.expect(rec1, 201)
	first := decode[observations.CreateResponse](t, rec1)
	if got := e.recipients(names); !slices.Equal(got, []string{"jan", "mia"}) {
		t.Fatalf("check recipients = %v, want [jan mia]", got)
	}
	for _, m := range e.fake.Sent() {
		if m.Data["kind"] != push.KindObservation || m.Data["horse_id"] != seed.HorseLuna || m.Data["urgency"] != "check" || m.Data["screen"] != "/observations/"+first.Observation.ID {
			t.Errorf("data = %v", m.Data)
		}
		if !strings.Contains(m.Body, "Kai meldet: Husten. Bitte ansehen.") || m.Title != "Luna" {
			t.Errorf("text = %q / %q", m.Title, m.Body)
		}
	}

	// The reporter is excluded even if he is the owner.
	e.fake.Reset()
	e.expect(e.call(seed.UserJan, "POST", "/api/v1/observations", report(seed.HorseLuna, nil)), 201)
	if got := e.recipients(names); !slices.Equal(got, []string{"mia"}) {
		t.Fatalf("owner reports: recipients = %v, want [mia]", got)
	}

	// Urgent: owner, rider and everybody present, whatever their visibility. Lea has hidden
	// her presence, Sarah left already, Anna opted out of urgent pushes.
	e.fake.Reset()
	e.exec(`UPDATE users SET presence_visibility = 'hidden' WHERE id = $1`, seed.UserLea)
	e.present(seed.UserLea)
	e.present(seed.UserAnna)
	e.present(seed.UserMia) // rider and present: notified once
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at) VALUES ($1, $2, now() - interval '3 hours', now() - interval '2 hours')`, seed.StableB, seed.UserSarah)
	e.exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'urgent_observation', false)`, seed.StableB, seed.UserAnna)
	rec := e.call(seed.UserKai, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{
		"category": "colic", "body_part": "belly", "urgency": "urgent",
	}))
	e.expect(rec, 201)
	if got := e.recipients(names); !slices.Equal(got, []string{"jan", "lea", "mia", "tom"}) {
		t.Fatalf("urgent recipients = %v, want [jan lea mia tom]", got)
	}
	resp := decode[observations.CreateResponse](t, rec)
	for _, m := range e.fake.Sent() {
		if m.Data["kind"] != push.KindUrgentObservation || m.Data["observation_id"] != resp.Observation.ID || m.Data["screen"] != "/observations/"+resp.Observation.ID {
			t.Errorf("data = %v", m.Data)
		}
		if m.Title != "Dringend: Luna" || !strings.Contains(m.Body, "Kai meldet: Kolik.") {
			t.Errorf("text = %q / %q", m.Title, m.Body)
		}
	}

	// The response carries the emergency card so the app can open it right away.
	card := resp.Emergency
	if card == nil || card.HorseID != seed.HorseLuna || card.HorseName != "Luna" || card.Owner == nil || card.Owner.ID != seed.UserJan {
		t.Fatalf("emergency = %+v", card)
	}
	if card.VetName == nil || card.EmergencyNote == nil || card.CanManage {
		t.Errorf("card = %+v", card)
	}
	// The reporter learns nothing about who was alerted.
	if strings.Contains(rec.Body.String(), "recipients") {
		t.Error("response must not list recipients")
	}

	// A cross-stable presence row never leaks into the recipients.
	e.fake.Reset()
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, now())`, otherStable, otherUser)
	e.expect(e.call(seed.UserKai, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"urgency": "urgent"})), 201)
	for _, m := range e.fake.Sent() {
		if names[m.To] == "" {
			t.Errorf("message to unknown token %s", m.To)
		}
	}
}

func TestUrgentWithoutOtherRecipients(t *testing.T) {
	e := setup(t)
	// Nobody registered a device: the report still succeeds.
	rec := e.call(seed.UserKai, "POST", "/api/v1/observations", report(seed.HorseFanta, map[string]any{"category": "lameness", "urgency": "urgent"}))
	e.expect(rec, 201)
	if len(e.fake.Sent()) != 0 {
		t.Errorf("sent = %v", e.fake.Sent())
	}
	if decode[observations.CreateResponse](t, rec).Emergency == nil {
		t.Error("urgent report without recipients still returns the card")
	}
}

func TestListGetPatch(t *testing.T) {
	e := setup(t)
	ids := map[string]string{}
	base := time.Now().Add(-3 * time.Hour)
	for i, c := range []struct{ key, user, cat string }{
		{"first", seed.UserMia, "cough"}, {"second", seed.UserTom, "lameness"}, {"third", seed.UserMia, "injury"},
	} {
		rec := e.call(c.user, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"category": c.cat}))
		e.expect(rec, 201)
		ids[c.key] = decode[observations.CreateResponse](t, rec).Observation.ID
		e.exec(`UPDATE observations SET created_at = $2 WHERE id = $1`, ids[c.key], base.Add(time.Duration(i)*time.Hour))
	}
	// One for another horse must not show up.
	e.expect(e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseFanta, nil)), 201)

	list := func(user, query string) []observations.Observation {
		rec := e.call(user, "GET", "/api/v1/horses/"+seed.HorseLuna+"/observations"+query, nil)
		e.expect(rec, 200)
		return decode[[]observations.Observation](t, rec)
	}
	got := list(seed.UserKai, "")
	if len(got) != 3 || got[0].ID != ids["third"] || got[1].ID != ids["second"] || got[2].ID != ids["first"] {
		t.Fatalf("order = %+v", got)
	}
	// can_change: Kai has no relation; Mia reported two, Jan owns the horse.
	if got[0].CanChange {
		t.Error("Kai must not be able to change")
	}
	if mia := list(seed.UserMia, ""); !mia[0].CanChange || mia[1].CanChange || !mia[2].CanChange {
		t.Errorf("Mia can_change = %v %v %v", mia[0].CanChange, mia[1].CanChange, mia[2].CanChange)
	}
	if jan := list(seed.UserJan, ""); !jan[0].CanChange || !jan[1].CanChange {
		t.Error("the owner may change all")
	}
	if l := list(seed.UserKai, "?limit=2"); len(l) != 2 {
		t.Errorf("limit: %d", len(l))
	}
	e.expect(e.call(seed.UserKai, "GET", "/api/v1/horses/"+seed.HorseLuna+"/observations?limit=0", nil), 400)
	e.expect(e.call(seed.UserKai, "GET", "/api/v1/horses/"+seed.HorseLuna+"/observations?status=open", nil), 400)

	one := e.call(seed.UserKai, "GET", "/api/v1/observations/"+ids["second"], nil)
	e.expect(one, 200)
	if o := decode[observations.Observation](t, one); o.Reporter.Name != "Tom" || *o.Category != "lameness" {
		t.Errorf("get = %+v", o)
	}
	e.expect(e.call(seed.UserKai, "GET", "/api/v1/observations/00000000-0000-4000-8000-00000000ffff", nil), 404)
	e.expect(e.call(seed.UserKai, "GET", "/api/v1/observations/nope", nil), 404)

	// Status changes: reporter, owner and admin may; a stranger and a rider of another horse may not.
	patch := func(user, id, status string) *httptest.ResponseRecorder {
		return e.call(user, "PATCH", "/api/v1/observations/"+id, map[string]any{"status": status})
	}
	e.expect(patch(seed.UserKai, ids["first"], "done"), 403)
	e.expect(patch(seed.UserLea, ids["first"], "done"), 403)  // rider of Fanta, not of Luna
	e.expect(patch(seed.UserMia, ids["first"], "done"), 200)  // reporter
	e.expect(patch(seed.UserJan, ids["second"], "done"), 200) // owner (and admin)
	e.exec(`UPDATE users SET is_admin = true WHERE id = $1`, seed.UserSarah)
	rec := patch(seed.UserSarah, ids["third"], "done") // admin without any relation
	e.expect(rec, 200)
	if o := decode[observations.Observation](t, rec); o.Status != "done" || !o.CanChange {
		t.Errorf("patched = %+v", o)
	}
	e.expect(patch(seed.UserMia, ids["first"], "watch"), 200) // reopen
	e.expect(patch(seed.UserMia, ids["first"], "gone"), 400)
	e.expect(e.call(seed.UserMia, "PATCH", "/api/v1/observations/"+ids["first"], map[string]any{}), 400)
	e.expect(e.call(seed.UserMia, "PATCH", "/api/v1/observations/"+ids["first"], map[string]any{"status": "done", "urgency": "info"}), 400)

	if l := list(seed.UserKai, "?status=watch"); len(l) != 1 || l[0].ID != ids["first"] {
		t.Errorf("watch filter = %+v", l)
	}
	if l := list(seed.UserKai, "?status=done"); len(l) != 2 || l[0].ID != ids["third"] {
		t.Errorf("done filter = %+v", l)
	}
}

func TestRehaPlanLink(t *testing.T) {
	e := setup(t)
	rec := e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseLuna, map[string]any{"category": "lameness"}))
	e.expect(rec, 201)
	id := decode[observations.CreateResponse](t, rec).Observation.ID
	get := func() observations.Observation {
		rec := e.call(seed.UserKai, "GET", "/api/v1/observations/"+id, nil)
		e.expect(rec, 200)
		return decode[observations.Observation](t, rec)
	}
	if o := get(); o.RehaPlanID != nil {
		t.Fatalf("reha_plan_id = %v, want null", *o.RehaPlanID)
	}
	// Two plans refer to it: an ended, older one and the active one; the active one is reported.
	var oldID, activeID string
	for _, c := range []struct {
		active bool
		age    string
		dst    *string
	}{{false, "10 days", &oldID}, {true, "1 day", &activeID}} {
		err := e.pool.QueryRow(context.Background(),
			`INSERT INTO reha_plans (stable_id, horse_id, diagnosis, start_date, active, observation_id, created_at)
			 VALUES ($1, $2, 'Test', CURRENT_DATE, $3, $4, now() - $5::interval) RETURNING id::text`,
			seed.StableB, seed.HorseLuna, c.active, id, c.age).Scan(c.dst)
		if err != nil {
			t.Fatal(err)
		}
	}
	if o := get(); o.RehaPlanID == nil || *o.RehaPlanID != activeID {
		t.Fatalf("reha_plan_id = %v, want %s", o.RehaPlanID, activeID)
	}
	// The list carries it as well; without an active plan the newest one wins.
	e.exec(`UPDATE reha_plans SET active = false WHERE id = $1`, activeID)
	rec = e.call(seed.UserKai, "GET", "/api/v1/horses/"+seed.HorseLuna+"/observations", nil)
	e.expect(rec, 200)
	if l := decode[[]observations.Observation](t, rec); len(l) != 1 || l[0].RehaPlanID == nil || *l[0].RehaPlanID != activeID {
		t.Fatalf("list = %+v", l)
	}
	_ = oldID
}

func TestRealtimeEvent(t *testing.T) {
	e := setup(t)
	sub, cancel := e.hub.Subscribe(seed.StableB)
	defer cancel()
	other, cancelOther := e.hub.Subscribe(otherStable)
	defer cancelOther()

	rec := e.call(seed.UserMia, "POST", "/api/v1/observations", report(seed.HorseLuna, nil))
	e.expect(rec, 201)
	id := decode[observations.CreateResponse](t, rec).Observation.ID
	expectEvent := func(what string) {
		t.Helper()
		select {
		case ev := <-sub:
			var data map[string]any
			_ = json.Unmarshal(ev.Data, &data)
			if ev.Type != observations.EventChanged || data["id"] != id || data["horse_id"] != seed.HorseLuna {
				t.Fatalf("%s: event = %+v", what, ev)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: no event", what)
		}
	}
	expectEvent("create")
	e.expect(e.call(seed.UserMia, "PATCH", "/api/v1/observations/"+id, map[string]any{"status": "done"}), 200)
	expectEvent("patch")
	// A rejected change publishes nothing.
	e.expect(e.call(seed.UserKai, "PATCH", "/api/v1/observations/"+id, map[string]any{"status": "watch"}), 403)
	select {
	case ev := <-sub:
		t.Fatalf("unexpected event %+v", ev)
	case ev := <-other:
		t.Fatalf("event leaked to another stable: %+v", ev)
	case <-time.After(300 * time.Millisecond):
	}
}
