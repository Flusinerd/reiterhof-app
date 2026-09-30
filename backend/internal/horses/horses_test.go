package horses_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/horses"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	otherStable = "00000000-0000-4000-8000-0000000009a1"
	otherUser   = "00000000-0000-4000-8000-0000000009a2"
	otherHorse  = "00000000-0000-4000-8000-0000000009a3"
)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
}

func setup(t *testing.T) *env {
	t.Helper()
	t.Cleanup(files.SetDir(t.TempDir()))
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO stables (id, name) VALUES ('` + otherStable + `', 'Anderer Stall')`,
		`INSERT INTO users (id, stable_id, name, email) VALUES ('` + otherUser + `', '` + otherStable + `', 'Fremd', 'fremd@example.org')`,
		`INSERT INTO horses (id, stable_id, name, owner_id) VALUES ('` + otherHorse + `', '` + otherStable + `', 'Fremdpferd', '` + otherUser + `')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	return &env{t: t, pool: pool, h: httpapi.NewHandler(httpapi.Deps{Pool: pool})}
}

// call sends a JSON request as the user ("" = anonymous) and returns the recorder.
func (e *env) call(user, method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
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

func TestListAndGet(t *testing.T) {
	e := setup(t)

	rec := e.call(seed.UserJan, "GET", "/api/v1/horses", nil)
	e.expect(rec, 200)
	list := decode[[]horses.Horse](t, rec)
	if len(list) != 7 {
		t.Fatalf("horses = %d, want 7 (other stable must not leak)", len(list))
	}
	byName := map[string]horses.Horse{}
	for _, h := range list {
		byName[h.Name] = h
	}
	luna := byName["Luna"]
	if !luna.IsMine || !luna.CanManage || luna.IRide || luna.Owner == nil || luna.Owner.Name != "Jan" {
		t.Errorf("Luna for Jan = %+v", luna)
	}
	if len(luna.Riders) != 1 || luna.Riders[0].Name != "Mia" || !reflect.DeepEqual(luna.Riders[0].Rules, []string{"ride", "groom"}) {
		t.Errorf("Luna riders = %+v", luna.Riders)
	}
	if luna.ColorKey == nil || *luna.ColorKey != "green" {
		t.Errorf("Luna color = %v", luna.ColorKey)
	}
	// Jan is admin: can manage everything but owns only Luna.
	if fanta := byName["Fanta"]; fanta.IsMine || !fanta.CanManage {
		t.Errorf("Fanta for admin Jan = %+v", fanta)
	}
	// Alphabetical.
	if list[0].Name != "Balu" {
		t.Errorf("first = %s", list[0].Name)
	}

	// Rider Mia.
	mia := decode[[]horses.Horse](t, e.call(seed.UserMia, "GET", "/api/v1/horses", nil))
	for _, h := range mia {
		switch h.Name {
		case "Luna":
			if !h.IRide || h.IsMine || h.CanManage || !reflect.DeepEqual(h.MyRules, []string{"ride", "groom"}) {
				t.Errorf("Luna for Mia = %+v", h)
			}
		default:
			if h.IRide || h.IsMine {
				t.Errorf("%s flags for Mia = %+v", h.Name, h)
			}
		}
	}

	// Detail.
	rec = e.call(seed.UserTom, "GET", "/api/v1/horses/"+seed.HorseLuna, nil)
	e.expect(rec, 200)
	if got := decode[horses.Horse](t, rec); got.ID != seed.HorseLuna || got.CanManage || got.MyRules == nil {
		t.Errorf("detail = %+v", got)
	}

	// Other stable: 404 for the horse, own list only.
	e.expect(e.call(otherUser, "GET", "/api/v1/horses/"+seed.HorseLuna, nil), 404)
	e.expect(e.call(seed.UserJan, "GET", "/api/v1/horses/"+otherHorse, nil), 404)
	e.expect(e.call(seed.UserJan, "GET", "/api/v1/horses/not-a-uuid", nil), 404)
	other := decode[[]horses.Horse](t, e.call(otherUser, "GET", "/api/v1/horses", nil))
	if len(other) != 1 || other[0].Name != "Fremdpferd" {
		t.Errorf("other stable list = %+v", other)
	}
	e.expect(e.call("", "GET", "/api/v1/horses", nil), 401)
}

func TestMembers(t *testing.T) {
	e := setup(t)
	rec := e.call(seed.UserTom, "GET", "/api/v1/members", nil)
	e.expect(rec, 200)
	members := decode[[]horses.Member](t, rec)
	if len(members) != 8 {
		t.Fatalf("members = %d, want 8", len(members))
	}
	me := 0
	for _, m := range members {
		if m.IsMe {
			me++
			if m.ID != seed.UserTom {
				t.Errorf("is_me on %s", m.Name)
			}
		}
	}
	if me != 1 {
		t.Errorf("is_me count = %d", me)
	}
	e.expect(e.call("", "GET", "/api/v1/members", nil), 401)
}

func TestPatchPermissions(t *testing.T) {
	e := setup(t)
	path := "/api/v1/horses/" + seed.HorseFanta // Anna's horse, Lea rides it
	body := map[string]any{"box": "20"}

	e.expect(e.call(seed.UserTom, "PATCH", path, body), 403) // plain member
	e.expect(e.call(seed.UserLea, "PATCH", path, body), 403) // rider
	e.expect(e.call(otherUser, "PATCH", path, body), 404)    // other stable
	e.expect(e.call("", "PATCH", path, body), 401)           // anonymous
	rec := e.call(seed.UserAnna, "PATCH", path, body)        // owner
	e.expect(rec, 200)
	if got := decode[horses.Horse](t, rec); got.Box == nil || *got.Box != "20" {
		t.Errorf("box = %v", got.Box)
	}
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"box": "21"}), 200) // admin

	// Fields, clearing and emergency data.
	rec = e.call(seed.UserAnna, "PATCH", path, map[string]any{
		"name": "Fanta II", "sex": "gelding", "birth_year": 2011, "breed": "Haflinger", "color_key": "teal",
		"weight_kg": 450, "helper_note": "Braucht Ruhe", "vet_phone": "0123", "allergies": "Penicillin",
	})
	e.expect(rec, 200)
	got := decode[horses.Horse](t, rec)
	if got.Name != "Fanta II" || *got.Sex != "gelding" || *got.BirthYear != 2011 || *got.WeightKG != 450 || *got.ColorKey != "teal" {
		t.Errorf("patched = %+v", got)
	}
	rec = e.call(seed.UserAnna, "PATCH", path, map[string]any{"breed": "", "weight_kg": 0})
	e.expect(rec, 200)
	got = decode[horses.Horse](t, rec)
	if got.Breed != nil || got.WeightKG != nil || got.Name != "Fanta II" {
		t.Errorf("cleared = %+v", got)
	}
	em := decode[horses.EmergencyCard](t, e.call(seed.UserTom, "GET", path+"/emergency", nil))
	if em.VetPhone == nil || *em.VetPhone != "0123" || em.Allergies == nil || *em.Allergies != "Penicillin" {
		t.Errorf("emergency after patch = %+v", em)
	}

	// Validation.
	for name, b := range map[string]map[string]any{
		"empty name": {"name": "  "},
		"sex":        {"sex": "unicorn"},
		"color":      {"color_key": "pink"},
		"year":       {"birth_year": 1800},
		"future":     {"birth_year": 3000},
		"weight":     {"weight_kg": 5},
		"long note":  {"helper_note": strings.Repeat("x", 2001)},
		"unknown":    {"nonsense": 1},
	} {
		if rec := e.call(seed.UserAnna, "PATCH", path, b); rec.Code != 400 {
			t.Errorf("%s: status = %d %s", name, rec.Code, rec.Body)
		}
	}

	// Only admins change the owner.
	e.expect(e.call(seed.UserAnna, "PATCH", path, map[string]any{"owner_id": seed.UserTom}), 403)
	e.expect(e.call(seed.UserJan, "PATCH", path, map[string]any{"owner_id": otherUser}), 400)
	rec = e.call(seed.UserJan, "PATCH", path, map[string]any{"owner_id": seed.UserTom})
	e.expect(rec, 200)
	if got := decode[horses.Horse](t, rec); got.Owner == nil || got.Owner.ID != seed.UserTom {
		t.Errorf("owner = %+v", got.Owner)
	}
	// Anna lost control.
	e.expect(e.call(seed.UserAnna, "PATCH", path, body), 403)
}

func TestCreate(t *testing.T) {
	e := setup(t)
	rec := e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{
		"name": " Sunny ", "box": "7", "sex": "mare", "birth_year": 2019, "color_key": "rose",
	})
	e.expect(rec, 201)
	got := decode[horses.Horse](t, rec)
	if got.Name != "Sunny" || got.Owner == nil || got.Owner.ID != seed.UserTom || !got.IsMine || !got.CanManage {
		t.Errorf("created = %+v", got)
	}
	var stable string
	if err := e.pool.QueryRow(context.Background(), `SELECT stable_id FROM horses WHERE id = $1`, got.ID).Scan(&stable); err != nil || stable != seed.StableB {
		t.Errorf("stable = %q %v", stable, err)
	}

	e.expect(e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"name": "X", "owner_id": seed.UserAnna}), 403)
	e.expect(e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"name": "X", "owner_id": seed.UserTom}), 201)
	rec = e.call(seed.UserJan, "POST", "/api/v1/horses", map[string]any{"name": "Y", "owner_id": seed.UserAnna})
	e.expect(rec, 201)
	if got := decode[horses.Horse](t, rec); got.Owner.ID != seed.UserAnna || got.IsMine {
		t.Errorf("admin created for Anna = %+v", got)
	}
	e.expect(e.call(seed.UserJan, "POST", "/api/v1/horses", map[string]any{"name": "Z", "owner_id": otherUser}), 400)
	e.expect(e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"box": "1"}), 400)
	e.expect(e.call(seed.UserTom, "POST", "/api/v1/horses", map[string]any{"name": ""}), 400)
	e.expect(e.call("", "POST", "/api/v1/horses", map[string]any{"name": "X"}), 401)
}

func TestRiders(t *testing.T) {
	e := setup(t)
	base := "/api/v1/horses/" + seed.HorseLuna + "/riders/"

	// Owner adds Tom with rules: normalized (order, duplicates).
	rec := e.call(seed.UserJan, "PUT", base+seed.UserTom, map[string]any{"rules": []string{"shows", "log_sessions", "shows"}})
	e.expect(rec, 200)
	r := decode[horses.Rider](t, rec)
	if r.Name != "Tom" || !reflect.DeepEqual(r.Rules, []string{"log_sessions", "shows"}) {
		t.Errorf("rider = %+v", r)
	}
	// Default rules, replace on second PUT.
	e.expect(e.call(seed.UserJan, "PUT", base+seed.UserKai, map[string]any{}), 200)
	rec = e.call(seed.UserJan, "PUT", base+seed.UserTom, map[string]any{"rules": []string{}})
	e.expect(rec, 200)
	detail := decode[horses.Horse](t, e.call(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseLuna, nil))
	rules := map[string][]string{}
	for _, r := range detail.Riders {
		rules[r.Name] = r.Rules
	}
	if len(detail.Riders) != 3 || len(rules["Tom"]) != 0 || !reflect.DeepEqual(rules["Kai"], horses.DefaultRiderRules()) {
		t.Errorf("riders = %+v", detail.Riders)
	}

	// Permissions.
	e.expect(e.call(seed.UserMia, "PUT", base+seed.UserSarah, map[string]any{}), 403)  // rider
	e.expect(e.call(seed.UserTom, "PUT", base+seed.UserSarah, map[string]any{}), 403)  // member
	e.expect(e.call(seed.UserAnna, "PUT", base+seed.UserSarah, map[string]any{}), 403) // other owner
	e.expect(e.call(seed.UserJan, "PUT", base+seed.UserSarah, map[string]any{}), 200)
	e.expect(e.call(otherUser, "PUT", base+seed.UserSarah, map[string]any{}), 404)
	e.expect(e.call(seed.UserMia, "DELETE", base+seed.UserTom, nil), 403)
	// Admin may manage riders of any horse.
	e.expect(e.call(seed.UserJan, "PUT", "/api/v1/horses/"+seed.HorseFanta+"/riders/"+seed.UserTom, map[string]any{}), 200)

	// Validation.
	e.expect(e.call(seed.UserJan, "PUT", base+seed.UserKai, map[string]any{"rules": []string{"fly"}}), 400)
	e.expect(e.call(seed.UserJan, "PUT", base+seed.UserJan, map[string]any{}), 400) // owner
	e.expect(e.call(seed.UserJan, "PUT", base+otherUser, map[string]any{}), 404)
	e.expect(e.call(seed.UserJan, "PUT", base+"nope", map[string]any{}), 404)

	// Delete.
	e.expect(e.call(seed.UserJan, "DELETE", base+seed.UserTom, nil), 204)
	e.expect(e.call(seed.UserJan, "DELETE", base+seed.UserTom, nil), 404)
	e.expect(e.call(seed.UserJan, "DELETE", base+"nope", nil), 404)
}

func TestRuleHelpers(t *testing.T) {
	for _, r := range horses.Rules() {
		if !horses.ValidRule(r) {
			t.Errorf("%s not valid", r)
		}
	}
	if horses.ValidRule("") || horses.ValidRule("admin") {
		t.Error("invalid rule accepted")
	}
	got, err := horses.NormalizeRules([]string{"shows", "ride", "ride"})
	if err != nil || !reflect.DeepEqual(got, []string{"ride", "shows"}) {
		t.Errorf("NormalizeRules = %v %v", got, err)
	}
	if _, err := horses.NormalizeRules([]string{"x"}); err == nil {
		t.Error("unknown rule accepted")
	}
}

func TestEmergencyCard(t *testing.T) {
	e := setup(t)
	path := "/api/v1/horses/" + seed.HorseLuna + "/emergency"

	// Every member reads it, also a member who is neither owner nor rider.
	for _, u := range []string{seed.UserMia, seed.UserTom, seed.UserJan} {
		rec := e.call(u, "GET", path, nil)
		e.expect(rec, 200)
		c := decode[horses.EmergencyCard](t, rec)
		if c.HorseName != "Luna" || c.Owner == nil || c.Owner.Name != "Jan" || c.Owner.Phone == nil || *c.Owner.Phone == "" {
			t.Errorf("%s: card = %+v", u, c)
		}
		if c.VetName == nil || c.EmergencyNote == nil || len(c.Contacts) != 2 || c.WeightKG == nil {
			t.Errorf("%s: incomplete card = %+v", u, c)
		}
		if want := u == seed.UserJan; c.CanManage != want {
			t.Errorf("%s: can_manage = %v", u, c.CanManage)
		}
	}
	e.expect(e.call(otherUser, "GET", path, nil), 404)
	e.expect(e.call("", "GET", path, nil), 401)

	// Contacts CRUD: owner and admin only.
	cpath := "/api/v1/horses/" + seed.HorseLuna + "/emergency-contacts"
	body := map[string]any{"label": "Nachbar", "name": "Herr Meier", "phone": "0170 111"}
	e.expect(e.call(seed.UserTom, "POST", cpath, body), 403)
	e.expect(e.call(seed.UserMia, "POST", cpath, body), 403)
	e.expect(e.call(otherUser, "POST", cpath, body), 404)
	rec := e.call(seed.UserJan, "POST", cpath, body)
	e.expect(rec, 201)
	c := decode[horses.Contact](t, rec)
	if c.ID == "" || c.Name != "Herr Meier" {
		t.Fatalf("contact = %+v", c)
	}
	e.expect(e.call(seed.UserJan, "POST", cpath, map[string]any{"label": "x"}), 400)
	e.expect(e.call(seed.UserJan, "POST", cpath, map[string]any{"label": "x", "name": "y", "phone": " "}), 400)

	e.expect(e.call(seed.UserTom, "PATCH", cpath+"/"+c.ID, map[string]any{"phone": "1"}), 403)
	rec = e.call(seed.UserJan, "PATCH", cpath+"/"+c.ID, map[string]any{"phone": "0170 222"})
	e.expect(rec, 200)
	if got := decode[horses.Contact](t, rec); got.Phone != "0170 222" || got.Name != "Herr Meier" {
		t.Errorf("patched contact = %+v", got)
	}
	// A contact of another horse is not reachable through this horse.
	e.expect(e.call(seed.UserJan, "PATCH", "/api/v1/horses/"+seed.HorseFanta+"/emergency-contacts/"+c.ID, map[string]any{"phone": "1"}), 404)
	e.expect(e.call(seed.UserTom, "DELETE", cpath+"/"+c.ID, nil), 403)
	e.expect(e.call(seed.UserJan, "DELETE", cpath+"/"+c.ID, nil), 204)
	e.expect(e.call(seed.UserJan, "DELETE", cpath+"/"+c.ID, nil), 404)
	e.expect(e.call(seed.UserJan, "DELETE", cpath+"/nope", nil), 404)
	card := decode[horses.EmergencyCard](t, e.call(seed.UserTom, "GET", path, nil))
	if len(card.Contacts) != 2 {
		t.Errorf("contacts = %d", len(card.Contacts))
	}
}

func TestDocuments(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	pdf := []byte("%PDF-1.7\nhello\n")
	saved, err := files.Save(ctx, seed.StableB, bytes.NewReader(pdf), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := files.Save(ctx, otherStable, bytes.NewReader(pdf), "application/pdf")
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/horses/" + seed.HorseLuna + "/documents"
	body := map[string]any{"kind": "passport", "title": "Equidenpass", "file_path": saved.Path}

	// Write: owner and admin only.
	e.expect(e.call(seed.UserMia, "POST", base, body), 403)
	e.expect(e.call(seed.UserTom, "POST", base, body), 403)
	e.expect(e.call(otherUser, "POST", base, body), 404)
	e.expect(e.call(seed.UserJan, "POST", base, map[string]any{"kind": "passport", "title": "x", "file_path": foreign.Path}), 400)
	e.expect(e.call(seed.UserJan, "POST", base, map[string]any{"kind": "passport", "title": "x", "file_path": "../../etc/passwd"}), 400)
	e.expect(e.call(seed.UserJan, "POST", base, map[string]any{"kind": "photo", "title": "x", "file_path": saved.Path}), 400)
	e.expect(e.call(seed.UserJan, "POST", base, map[string]any{"kind": "other", "title": " ", "file_path": saved.Path}), 400)
	rec := e.call(seed.UserJan, "POST", base, body)
	e.expect(rec, 201)
	doc := decode[horses.Document](t, rec)
	if doc.ID == "" || doc.ContentType != "application/pdf" || doc.URL != base+"/"+doc.ID+"/file" {
		t.Fatalf("doc = %+v", doc)
	}

	// Read: owner, riders, admin; not plain members, not other stables.
	for u, want := range map[string]int{seed.UserJan: 200, seed.UserMia: 200, seed.UserTom: 403, otherUser: 404, "": 401} {
		if rec := e.call(u, "GET", base, nil); rec.Code != want {
			t.Errorf("list as %q = %d, want %d", u, rec.Code, want)
		}
	}
	list := decode[[]horses.Document](t, e.call(seed.UserMia, "GET", base, nil))
	if len(list) != 1 || list[0].Title != "Equidenpass" || list[0].UploadedBy == nil || list[0].UploadedBy.Name != "Jan" {
		t.Errorf("list = %+v", list)
	}
	// Admin who is not owner reads too.
	e.expect(e.call(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseFanta+"/documents", nil), 200)
	e.expect(e.call(seed.UserTom, "GET", "/api/v1/horses/"+seed.HorseFanta+"/documents", nil), 403)

	// File download goes through the same check.
	rec = e.call(seed.UserMia, "GET", doc.URL, nil)
	e.expect(rec, 200)
	if !bytes.Equal(rec.Body.Bytes(), pdf) || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Errorf("file = %q %v", rec.Body, rec.Header())
	}
	e.expect(e.call(seed.UserTom, "GET", doc.URL, nil), 403)
	e.expect(e.call(otherUser, "GET", doc.URL, nil), 404)
	// Same document id under another horse does not resolve.
	e.expect(e.call(seed.UserJan, "GET", "/api/v1/horses/"+seed.HorseFanta+"/documents/"+doc.ID+"/file", nil), 404)
	// Viewers that cannot send headers get a short-lived link; the role check still applies.
	rec = e.call(seed.UserMia, "POST", "/api/v1/files/download-link", map[string]any{"url": doc.URL})
	e.expect(rec, 200)
	link := decode[struct{ URL string }](t, rec).URL
	e.expect(e.call("", "GET", link, nil), 200)
	rec = e.call(seed.UserTom, "POST", "/api/v1/files/download-link", map[string]any{"url": doc.URL})
	e.expect(rec, 200)
	e.expect(e.call("", "GET", decode[struct{ URL string }](t, rec).URL, nil), 403)
	// Session tokens in the URL are never accepted.
	e.expect(e.call("", "GET", doc.URL+"?access_token="+authtest.Token(t, e.pool, seed.UserMia), nil), 401)

	// Update and delete.
	e.expect(e.call(seed.UserMia, "PATCH", base+"/"+doc.ID, map[string]any{"title": "Neu"}), 403)
	rec = e.call(seed.UserJan, "PATCH", base+"/"+doc.ID, map[string]any{"title": "Impfpass", "kind": "vaccination_record"})
	e.expect(rec, 200)
	if got := decode[horses.Document](t, rec); got.Title != "Impfpass" || got.Kind != "vaccination_record" {
		t.Errorf("patched doc = %+v", got)
	}
	e.expect(e.call(seed.UserJan, "PATCH", base+"/"+doc.ID, map[string]any{"kind": "photo"}), 400)
	e.expect(e.call(seed.UserMia, "DELETE", base+"/"+doc.ID, nil), 403)
	e.expect(e.call(seed.UserJan, "DELETE", base+"/"+doc.ID, nil), 204)
	e.expect(e.call(seed.UserJan, "DELETE", base+"/"+doc.ID, nil), 404)
	if _, _, err := files.Open(seed.StableB, saved.Path); err != files.ErrNotFound {
		t.Errorf("file after delete: %v", err)
	}
}
