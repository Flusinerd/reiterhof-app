package privacy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files/filestest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpapi"
	"github.com/Flusinerd/reiterhof-app/backend/internal/privacy"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

const (
	otherStable = "00000000-0000-4000-8000-0000000009a1"
	otherUser   = "00000000-0000-4000-8000-0000000009a2"
	otherHorse  = "00000000-0000-4000-8000-0000000009a3"
)

// clock is fixed so that sessions and retention cut-offs are deterministic.
var clock = time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)

type env struct {
	t    *testing.T
	pool *pgxpool.Pool
	h    http.Handler
}

func setup(t *testing.T) *env {
	t.Helper()
	t.Cleanup(files.SetDir(t.TempDir()))
	pool := dbtest.NewSeeded(t)
	e := &env{t: t, pool: pool, h: httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: func() time.Time { return clock }})}
	e.exec(`INSERT INTO stables (id, name) VALUES ($1, 'Anderer Stall')`, otherStable)
	e.exec(`INSERT INTO users (id, stable_id, name, email, phone) VALUES ($1, $2, 'Fremd', 'fremd@example.org', '+49 999')`, otherUser, otherStable)
	e.exec(`INSERT INTO horses (id, stable_id, name, owner_id) VALUES ($1, $2, 'Fremdpferd', $3)`, otherHorse, otherStable, otherUser)
	return e
}

func (e *env) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *env) count(sql string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func (e *env) do(userID, method, path string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	if userID != "" {
		req.Header.Set("Authorization", "Bearer "+authtest.TokenAt(e.t, e.pool, userID, clock))
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestRoutesNeedSignIn(t *testing.T) {
	e := setup(t)
	for _, r := range [][2]string{
		{"GET", "/api/v1/me/consents"}, {"PUT", "/api/v1/me/consents/push"},
		{"GET", "/api/v1/me/export"}, {"POST", "/api/v1/me/delete"},
	} {
		if rec := e.do("", r[0], r[1], nil); rec.Code != 401 {
			t.Errorf("%s %s = %d, want 401", r[0], r[1], rec.Code)
		}
	}
}

func TestConsentsGrantAndRevoke(t *testing.T) {
	e := setup(t)
	type list struct {
		CurrentVersion string            `json:"current_version"`
		Items          []privacy.Consent `json:"items"`
	}
	get := func(user string) list {
		rec := e.do(user, "GET", "/api/v1/me/consents", nil)
		if rec.Code != 200 {
			t.Fatalf("GET consents = %d %s", rec.Code, rec.Body)
		}
		return decode[list](t, rec)
	}
	l := get(seed.UserAnna)
	if l.CurrentVersion != privacy.TextVersion || len(l.Items) != 6 {
		t.Fatalf("list = %+v", l)
	}
	for i, k := range privacy.Kinds() {
		if l.Items[i].Kind != k || l.Items[i].Granted {
			t.Errorf("item %d = %+v, want %s not granted", i, l.Items[i], k)
		}
	}

	rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/location_geofence", map[string]any{"granted": true, "version": privacy.TextVersion})
	if rec.Code != 200 {
		t.Fatalf("grant = %d %s", rec.Code, rec.Body)
	}
	c := decode[privacy.Consent](t, rec)
	if !c.Granted || !c.UpToDate || c.Version == nil || *c.Version != privacy.TextVersion || c.GrantedAt == nil || !c.GrantedAt.Equal(clock) {
		t.Fatalf("granted consent = %+v", c)
	}
	// Other users do not see it (and cannot be granted for by someone else).
	if got := get(seed.UserTom).Items[0]; got.Granted {
		t.Fatalf("Tom sees Anna's consent: %+v", got)
	}

	rec = e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/location_geofence", map[string]any{"granted": false})
	c = decode[privacy.Consent](t, rec)
	if rec.Code != 200 || c.Granted || c.RevokedAt == nil {
		t.Fatalf("revoke = %d %+v", rec.Code, c)
	}
	// Grant again clears the revocation.
	rec = e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/location_geofence", map[string]any{"granted": true})
	if c = decode[privacy.Consent](t, rec); !c.Granted || c.RevokedAt != nil {
		t.Fatalf("re-grant = %+v", c)
	}
	if n := e.count(`SELECT count(*) FROM consents WHERE user_id = $1`, seed.UserAnna); n != 1 {
		t.Fatalf("%d consent rows, want 1 (upsert)", n)
	}

	// A grant for an outdated text is refused; revoking never needs a version.
	if rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": true, "version": "1999-01-01"}); rec.Code != 409 {
		t.Errorf("stale version = %d, want 409", rec.Code)
	}
	if rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/telepathy", map[string]any{"granted": true}); rec.Code != 404 {
		t.Errorf("unknown kind = %d, want 404", rec.Code)
	}
	if rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/photos", map[string]any{}); rec.Code != 400 {
		t.Errorf("missing granted = %d, want 400", rec.Code)
	}
	if rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": true, "extra": 1}); rec.Code != 400 {
		t.Errorf("unknown field = %d, want 400", rec.Code)
	}

	// A grant for an old text version shows up as not up to date.
	e.exec(`UPDATE consents SET version = 'old' WHERE user_id = $1`, seed.UserAnna)
	if got := get(seed.UserAnna).Items[0]; !got.Granted || got.UpToDate {
		t.Errorf("old version = %+v, want granted but not up to date", got)
	}

	// The map consent exists like the others.
	if rec := e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/maps", map[string]any{"granted": true}); rec.Code != 200 {
		t.Errorf("maps = %d %s", rec.Code, rec.Body)
	}
	// A person whose age is not confirmed cannot grant anything (Art. 8); revoking is always possible.
	e.exec(`UPDATE users SET age_confirmed_at = NULL, parental_consent_at = NULL WHERE id = $1`, seed.UserTom)
	if rec := e.do(seed.UserTom, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": true}); rec.Code != 403 || !strings.Contains(rec.Body.String(), "age_unconfirmed") {
		t.Errorf("unconfirmed age grant = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do(seed.UserTom, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": false}); rec.Code != 200 {
		t.Errorf("unconfirmed age revoke = %d %s", rec.Code, rec.Body)
	}
	e.exec(`UPDATE users SET parental_consent_at = now() WHERE id = $1`, seed.UserTom)
	if rec := e.do(seed.UserTom, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": true}); rec.Code != 200 {
		t.Errorf("grant with parental consent = %d %s", rec.Code, rec.Body)
	}
}

func TestRevocationConsequences(t *testing.T) {
	e := setup(t)
	e.exec(`INSERT INTO push_tokens (stable_id, user_id, token, platform) VALUES ($1, $2, 'ExponentPushToken[anna]', 'android'), ($1, $3, 'ExponentPushToken[tom]', 'ios')`,
		seed.StableB, seed.UserAnna, seed.UserTom)
	e.exec(`INSERT INTO web_push_subscriptions (stable_id, user_id, endpoint, p256dh, auth) VALUES ($1, $2, 'https://push.example.com/anna', 'k', 'a'), ($1, $3, 'https://push.example.com/tom', 'k', 'a')`,
		seed.StableB, seed.UserAnna, seed.UserTom)
	for _, k := range []string{"push", "presence_sharing"} {
		e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/"+k, map[string]any{"granted": true})
	}
	e.exec(`UPDATE users SET presence_visibility = 'all' WHERE id = $1`, seed.UserAnna)

	e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/push", map[string]any{"granted": false})
	if n := e.count(`SELECT count(*) FROM push_tokens WHERE user_id = $1`, seed.UserAnna); n != 0 {
		t.Errorf("Anna keeps %d push tokens after revoking push", n)
	}
	if n := e.count(`SELECT count(*) FROM push_tokens WHERE user_id = $1`, seed.UserTom); n != 1 {
		t.Errorf("Tom has %d push tokens, want 1", n)
	}
	if n := e.count(`SELECT count(*) FROM web_push_subscriptions WHERE user_id = $1`, seed.UserAnna); n != 0 {
		t.Errorf("Anna keeps %d web push subscriptions after revoking push", n)
	}
	if n := e.count(`SELECT count(*) FROM web_push_subscriptions WHERE user_id = $1`, seed.UserTom); n != 1 {
		t.Errorf("Tom has %d web push subscriptions, want 1", n)
	}
	e.do(seed.UserAnna, "PUT", "/api/v1/me/consents/presence_sharing", map[string]any{"granted": false})
	var vis string
	_ = e.pool.QueryRow(context.Background(), `SELECT presence_visibility FROM users WHERE id = $1`, seed.UserAnna).Scan(&vis)
	if vis != "hidden" {
		t.Errorf("visibility = %q after revoking presence sharing, want hidden", vis)
	}
}

// addMiaData gives Mia records in many tables and returns the path of an uploaded photo.
func (e *env) addMiaData() string {
	e.t.Helper()
	ctx := context.Background()
	saved, err := files.Save(ctx, seed.StableB, bytes.NewReader(filestest.JPEG()), "image/jpeg")
	if err != nil {
		e.t.Fatal(err)
	}
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at, source) VALUES
		($1, $2, $3, $4, 'geofence'), ($1, $5, $3, $4, 'manual')`,
		seed.StableB, seed.UserMia, clock.Add(-48*time.Hour), clock.Add(-46*time.Hour), seed.UserTom)
	e.exec(`INSERT INTO push_tokens (stable_id, user_id, token, platform) VALUES ($1, $2, 'ExponentPushToken[abcdefghijkl]', 'ios')`, seed.StableB, seed.UserMia)
	e.exec(`INSERT INTO web_push_subscriptions (stable_id, user_id, endpoint, p256dh, auth, user_agent) VALUES ($1, $2, 'https://web.push.apple.com/SECRETPATH', 'k', 'a', 'Safari')`, seed.StableB, seed.UserMia)
	e.exec(`INSERT INTO sessions (stable_id, horse_id, user_id, activity, started_at, duration_min, track, source, note)
		VALUES ($1, $2, $3, 'hack', $4, 45, '[{"lat":51.6,"lng":6.9}]', 'tracked', 'schoen'),
		       ($1, $2, $5, 'hack', $4, 30, '[{"lat":50.0,"lng":7.0}]', 'tracked', 'Toms Ritt')`,
		seed.StableB, seed.HorseLuna, seed.UserMia, clock.Add(-24*time.Hour), seed.UserTom)
	e.exec(`INSERT INTO observations (stable_id, horse_id, reported_by, description, media)
		VALUES ($1, $2, $3, 'Mias Beobachtung', $4::jsonb), ($1, $2, $5, 'Toms Beobachtung', '[]')`,
		seed.StableB, seed.HorseLuna, seed.UserMia, `["`+saved.Path+`"]`, seed.UserTom)
	e.exec(`INSERT INTO requests (id, stable_id, type, created_by, date, description) VALUES
		('00000000-0000-4000-8000-0000000007a1', $1, 'exercise', $2, '2026-10-05', 'Mias Anfrage'),
		('00000000-0000-4000-8000-0000000007a2', $1, 'blanket', $3, '2026-10-06', 'Toms Anfrage')`,
		seed.StableB, seed.UserMia, seed.UserTom)
	e.exec(`UPDATE requests SET status = 'assigned' WHERE id = '00000000-0000-4000-8000-0000000007a2'`)
	e.exec(`INSERT INTO request_assignees (stable_id, request_id, user_id) VALUES ($1, '00000000-0000-4000-8000-0000000007a2', $2)`,
		seed.StableB, seed.UserMia)
	e.exec(`INSERT INTO horse_riders (stable_id, horse_id, user_id, rules) VALUES ($1, $2, $3, '["ride"]') ON CONFLICT DO NOTHING`,
		seed.StableB, seed.HorseFanta, seed.UserMia)
	e.exec(`INSERT INTO reha_plans (stable_id, horse_id, diagnosis, start_date, created_by, active) VALUES ($1, $2, 'Mias Plan', '2026-09-01', $3, false), ($1, $2, 'Toms Plan', '2026-09-02', $4, false)`,
		seed.StableB, seed.HorseFanta, seed.UserMia, seed.UserTom)
	e.exec(`INSERT INTO gait_windows (session_id, stable_id, window_count, windows)
		SELECT id, stable_id, 1, ('[{"t":0,"f":[1,2,3,4,5,6],"p":"walk","a":"walk","c":"' || note || '"}]')::jsonb FROM sessions WHERE note IN ('schoen', 'Toms Ritt')`)
	e.exec(`INSERT INTO stable_invites (stable_id, code, created_by, expires_at) VALUES ($1, 'MIASCODE', $2, $3), ($1, 'TOMSCODE', $4, $3)`,
		seed.StableB, seed.UserMia, clock.Add(24*time.Hour), seed.UserTom)
	e.exec(`INSERT INTO login_tokens (token_hash, email, expires_at) VALUES ('\xa1', 'mia@example.org', $1)`, clock.Add(time.Hour))
	e.exec(`INSERT INTO parental_consent_tokens (user_id, token_hash, parent_email, expires_at) VALUES ($1, '\xb1', 'mias.mutter@example.org', $3), ($2, '\xb2', 'toms.vater@example.org', $3)`,
		seed.UserMia, seed.UserTom, clock.Add(time.Hour))
	e.do(seed.UserMia, "PUT", "/api/v1/me/consents/photos", map[string]any{"granted": true})
	return saved.Path
}

func TestExportContainsOnlyOwnData(t *testing.T) {
	e := setup(t)
	photo := e.addMiaData()
	// Data of the other stable's user must never show up.
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at) VALUES ($1, $2, $3, $4)`,
		otherStable, otherUser, clock.Add(-5*time.Hour), clock.Add(-4*time.Hour))

	rec := e.do(seed.UserMia, "GET", "/api/v1/me/export", nil)
	if rec.Code != 200 {
		t.Fatalf("export = %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "reiterhof-export-2026-09-30.json") {
		t.Errorf("Content-Disposition = %q", rec.Header().Get("Content-Disposition"))
	}
	body := rec.Body.String()
	doc := decode[map[string]json.RawMessage](t, rec)
	for _, key := range []string{"profile", "stable", "consents", "push_tokens", "presence_visits", "training_sessions",
		"observations_reported", "requests_created", "requests_helped", "horse_rider_roles", "uploaded_files", "sign_in_sessions", "web_push_subscriptions",
		"reha_plans_created", "gait_windows", "invites_created", "sign_in_attempts", "parental_consent_requests"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("export lacks %q", key)
		}
	}
	var profile struct{ Email, Name string }
	_ = json.Unmarshal(doc["profile"], &profile)
	if profile.Email != "mia@example.org" {
		t.Errorf("profile = %+v", profile)
	}
	count := func(key string) int {
		var v []json.RawMessage
		_ = json.Unmarshal(doc[key], &v)
		return len(v)
	}
	if got := count("training_sessions"); got < 1 { // the seed adds some of its own
		t.Errorf("training_sessions has %d entries", got)
	}
	// The seed also contains observations by Mia, so compare against the database.
	var observations int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM observations WHERE reported_by = $1`, seed.UserMia).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]int{"presence_visits": 1, "observations_reported": observations,
		"requests_created": 1, "requests_helped": 1, "push_tokens": 1, "web_push_subscriptions": 1, "uploaded_files": 1,
		"reha_plans_created": 1, "gait_windows": 1, "invites_created": 1, "sign_in_attempts": 1, "parental_consent_requests": 1} {
		if got := count(key); got != want {
			t.Errorf("%s has %d entries, want %d", key, got, want)
		}
	}
	// The consent list has all kinds, photos is granted.
	if !strings.Contains(string(doc["consents"]), `"kind":"photos","granted":true`) {
		t.Errorf("consents = %s", doc["consents"])
	}
	// The push token is masked, the session hash is not exported, the photo is listed.
	for _, must := range []string{"Expo...jkl]", "web.push.apple.com", "Mias Beobachtung", "Mias Anfrage", "schoen", photo} {
		if !strings.Contains(body, must) {
			t.Errorf("export lacks %q", must)
		}
	}
	for _, must := range []string{"Mias Plan", "mias.mutter@example.org"} {
		if !strings.Contains(body, must) {
			t.Errorf("export lacks %q", must)
		}
	}
	for _, mustNot := range []string{"ExponentPushToken[abcdefghijkl]", "SECRETPATH", "token_hash", "Toms Beobachtung", "Toms Ritt", "Toms Anfrage",
		"tom@example.org", "Fremd", "+49 999", "50.0", "Toms Plan", "MIASCODE", "TOMSCODE", "toms.vater", "code_hash"} {
		if strings.Contains(body, mustNot) {
			t.Errorf("export leaks %q", mustNot)
		}
	}

	// The user of another stable gets their own data only.
	rec = e.do(otherUser, "GET", "/api/v1/me/export", nil)
	if rec.Code != 200 {
		t.Fatalf("export other = %d %s", rec.Code, rec.Body)
	}
	other := rec.Body.String()
	if !strings.Contains(other, "fremd@example.org") || !strings.Contains(other, "Fremdpferd") {
		t.Errorf("other export incomplete: %s", other)
	}
	for _, mustNot := range []string{"mia@example.org", "Mias Beobachtung", "Luna", seed.StableB} {
		if strings.Contains(other, mustNot) {
			t.Errorf("other stable's export leaks %q", mustNot)
		}
	}
}

func TestDeleteAccountAnonymisesAndKeepsOthersIntact(t *testing.T) {
	e := setup(t)
	dir := t.TempDir()
	t.Cleanup(files.SetDir(dir))
	photo := e.addMiaData()
	e.exec(`INSERT INTO login_tokens (token_hash, email, expires_at) VALUES ('\x01', 'mia@example.org', $1), ('\x02', 'tom@example.org', $1)`, clock.Add(time.Hour))
	e.exec(`INSERT INTO auth_identities (user_id, provider, subject, email) VALUES ($1, 'google', 'sub-mia', 'mia@example.org')`, seed.UserMia)
	tomVisits := e.count(`SELECT count(*) FROM presence WHERE user_id = $1`, seed.UserTom)
	otherPresence := e.count(`SELECT count(*) FROM presence WHERE stable_id = $1`, otherStable)

	token := authtest.TokenAt(t, e.pool, seed.UserMia, clock)
	if rec := e.do(seed.UserMia, "POST", "/api/v1/me/delete", map[string]any{}); rec.Code != 400 {
		t.Fatalf("delete without confirm = %d, want 400", rec.Code)
	}
	rec := e.do(seed.UserMia, "POST", "/api/v1/me/delete", map[string]any{"confirm": true})
	if rec.Code != 204 {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}

	// The session is gone, so is everything personal.
	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("old session after deletion = %d, want 401", rec.Code)
	}
	for _, table := range []string{"auth_sessions", "auth_identities", "push_tokens", "web_push_subscriptions", "consents", "presence", "horse_riders", "request_assignees", "parental_consent_tokens"} {
		if n := e.count(`SELECT count(*) FROM `+table+` WHERE user_id = $1`, seed.UserMia); n != 0 {
			t.Errorf("%s keeps %d rows of the deleted user", table, n)
		}
	}
	if n := e.count(`SELECT count(*) FROM login_tokens WHERE email = 'mia@example.org'`); n != 0 {
		t.Errorf("login token of the deleted email remains")
	}
	var name, email string
	var parentEmail *string
	var ageAt, parentalAt *time.Time
	if err := e.pool.QueryRow(context.Background(), `SELECT parent_email, age_confirmed_at, parental_consent_at FROM users WHERE id = $1`, seed.UserMia).
		Scan(&parentEmail, &ageAt, &parentalAt); err != nil {
		t.Fatal(err)
	}
	if parentEmail != nil || ageAt != nil || parentalAt != nil {
		t.Errorf("age data remains: %v %v %v", parentEmail, ageAt, parentalAt)
	}
	var phone, stable *string
	var admin bool
	var deletedAt *time.Time
	if err := e.pool.QueryRow(context.Background(), `SELECT name, email, phone, stable_id::text, is_admin, deleted_at FROM users WHERE id = $1`, seed.UserMia).
		Scan(&name, &email, &phone, &stable, &admin, &deletedAt); err != nil {
		t.Fatal(err)
	}
	if name != privacy.DeletedName || strings.Contains(email, "mia") || !strings.HasSuffix(email, "@deleted.invalid") ||
		phone != nil || stable != nil || admin || deletedAt == nil || !deletedAt.Equal(clock) {
		t.Errorf("user row = %q %q %v %v %v %v", name, email, phone, stable, admin, deletedAt)
	}
	// The email can be used again; the tombstone does not appear as stable member.
	e.exec(`INSERT INTO users (stable_id, name, email) VALUES ($1, 'Neue Mia', 'mia@example.org')`, seed.StableB)
	members := decode[[]struct{ ID string }](t, e.do(seed.UserJan, "GET", "/api/v1/members", nil))
	for _, m := range members {
		if m.ID == seed.UserMia {
			t.Errorf("deleted user is listed as member")
		}
	}

	// Records of horses stay, without personal detail.
	if n := e.count(`SELECT count(*) FROM sessions WHERE user_id = $1 AND track IS NULL AND duration_min = 45`, seed.UserMia); n != 1 {
		t.Errorf("Mia's session should stay without track, got %d", n)
	}
	// The seed adds observations by Mia as well: all of them stay, none keeps media.
	if n := e.count(`SELECT count(*) FROM observations WHERE reported_by = $1`, seed.UserMia); n < 1 {
		t.Errorf("Mia's observations should stay, got %d", n)
	}
	if n := e.count(`SELECT count(*) FROM observations WHERE reported_by = $1 AND media <> '[]'::jsonb`, seed.UserMia); n != 0 {
		t.Errorf("Mia's observations should stay without media, %d still have media", n)
	}
	if _, err := os.Stat(filepath.Join(dir, photo)); !os.IsNotExist(err) {
		t.Errorf("uploaded photo still on disk: %v", err)
	}
	if st := e.count(`SELECT count(*) FROM requests WHERE id = '00000000-0000-4000-8000-0000000007a1' AND status = 'cancelled'`); st != 1 {
		t.Errorf("Mia's open request was not cancelled")
	}
	if st := e.count(`SELECT count(*) FROM requests WHERE id = '00000000-0000-4000-8000-0000000007a2' AND status = 'open'`); st != 1 {
		t.Errorf("Tom's request should be open again after its only helper left")
	}

	// Others are untouched.
	if n := e.count(`SELECT count(*) FROM presence WHERE user_id = $1`, seed.UserTom); n != tomVisits {
		t.Errorf("Tom's visits changed: %d -> %d", tomVisits, n)
	}
	if n := e.count(`SELECT count(*) FROM presence WHERE stable_id = $1`, otherStable); n != otherPresence {
		t.Errorf("other stable's presence changed")
	}
	if n := e.count(`SELECT count(*) FROM sessions WHERE user_id = $1 AND track IS NOT NULL`, seed.UserTom); n != 1 {
		t.Errorf("Tom's track was touched")
	}
	if n := e.count(`SELECT count(*) FROM login_tokens WHERE email = 'tom@example.org'`); n != 1 {
		t.Errorf("Tom's login token was touched")
	}
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1 AND name = 'Fremd'`, otherUser); n != 1 {
		t.Errorf("other stable's user was touched")
	}

	// Deleting twice: the account no longer exists.
	if err := privacy.DeleteAccount(context.Background(), e.pool, seed.UserMia, clock); err != privacy.ErrNotFound {
		t.Errorf("second delete = %v, want ErrNotFound", err)
	}
}

func TestDeleteBlockedForOwnersAndLastAdmin(t *testing.T) {
	e := setup(t)
	// Jan owns Luna and is the only admin of the stable.
	rec := e.do(seed.UserJan, "POST", "/api/v1/me/delete", map[string]any{"confirm": true})
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "owns_horses") || !strings.Contains(rec.Body.String(), "Luna") {
		t.Fatalf("owner delete = %d %s, want 409 owns_horses naming Luna", rec.Code, rec.Body)
	}
	if n := e.count(`SELECT count(*) FROM users WHERE id = $1 AND deleted_at IS NULL AND email = 'jan@example.org'`, seed.UserJan); n != 1 {
		t.Fatal("blocked deletion changed the user")
	}

	// After the horse is transferred the last-admin rule applies.
	e.exec(`UPDATE horses SET owner_id = $1 WHERE owner_id = $2`, seed.UserAnna, seed.UserJan)
	rec = e.do(seed.UserJan, "POST", "/api/v1/me/delete", map[string]any{"confirm": true})
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), "last_admin") {
		t.Fatalf("last admin delete = %d %s, want 409 last_admin", rec.Code, rec.Body)
	}
	// The admin of another stable does not count.
	e.exec(`UPDATE users SET is_admin = true WHERE id = $1`, otherUser)
	if rec := e.do(seed.UserJan, "POST", "/api/v1/me/delete", map[string]any{"confirm": true}); rec.Code != 409 {
		t.Fatalf("delete with only a foreign admin = %d, want 409", rec.Code)
	}

	e.exec(`UPDATE users SET is_admin = true WHERE id = $1`, seed.UserAnna)
	if rec := e.do(seed.UserJan, "POST", "/api/v1/me/delete", map[string]any{"confirm": true}); rec.Code != 204 {
		t.Fatalf("delete after transfer = %d %s, want 204", rec.Code, rec.Body)
	}
	// The horse stays with its new owner.
	if n := e.count(`SELECT count(*) FROM horses WHERE name = 'Luna' AND owner_id = $1`, seed.UserAnna); n != 1 {
		t.Errorf("Luna should stay with Anna")
	}
}

func TestUserWithoutStableCanExportAndDelete(t *testing.T) {
	e := setup(t)
	e.exec(`INSERT INTO users (id, name, email, age_confirmed_at) VALUES ('00000000-0000-4000-8000-0000000009b1', 'Neu', 'neu@example.org', now())`)
	id := "00000000-0000-4000-8000-0000000009b1"
	if rec := e.do(id, "PUT", "/api/v1/me/consents/push", map[string]any{"granted": true}); rec.Code != 200 {
		t.Fatalf("consent without stable = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do(id, "GET", "/api/v1/me/export", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), "neu@example.org") {
		t.Fatalf("export = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do(id, "POST", "/api/v1/me/delete", map[string]any{"confirm": true}); rec.Code != 204 {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
}

func TestPrune(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	old := clock.AddDate(0, -13, 0)
	recent := clock.AddDate(0, -11, 0)
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at, left_at) VALUES
		($1, $2, $3, $4), ($1, $2, $5, $6)`, seed.StableB, seed.UserTom, old, old.Add(time.Hour), recent, recent.Add(time.Hour))
	// An open visit is never pruned by this job (presence-close-stale closes it first).
	e.exec(`INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, $3)`, seed.StableB, seed.UserAnna, old)
	e.exec(`INSERT INTO sessions (stable_id, horse_id, user_id, activity, started_at, track) VALUES
		($1, $2, $3, 'hack', $4, '[1]'), ($1, $2, $3, 'hack', $5, '[2]')`, seed.StableB, seed.HorseLuna, seed.UserTom, old, recent)
	e.exec(`INSERT INTO login_tokens (token_hash, email, expires_at) VALUES ('\x11', 'a@example.org', $1), ('\x12', 'b@example.org', $2)`,
		clock.Add(-time.Minute), clock.Add(time.Minute))
	e.exec(`INSERT INTO auth_sessions (token_hash, user_id, expires_at) VALUES ('\x21', $1, $2), ('\x22', $1, $3)`,
		seed.UserTom, clock.Add(-time.Minute), clock.Add(time.Hour))
	e.exec(`INSERT INTO reminders (stable_id, user_id, kind, title, due_at) VALUES ($1, $2, 'health_due', 'alt', $3), ($1, $2, 'health_due', 'neu', $4)`,
		seed.StableB, seed.UserTom, old, recent)
	// Raw gait windows follow the track: the old session's windows go, the recent ones stay.
	e.exec(`INSERT INTO gait_windows (session_id, stable_id, window_count, windows)
		SELECT id, stable_id, 1, '[{}]' FROM sessions WHERE user_id = $1 AND track IS NOT NULL`, seed.UserTom)
	// Invites: long expired ones go, a recently expired and a live one stay.
	e.exec(`INSERT INTO stable_invites (stable_id, code, expires_at) VALUES ($1, 'OLDCODE1', $2), ($1, 'RECENT01', $3), ($1, 'LIVE0001', $4)`,
		seed.StableB, clock.AddDate(0, 0, -31), clock.AddDate(0, 0, -29), clock.Add(time.Hour))
	e.exec(`INSERT INTO parental_consent_tokens (user_id, token_hash, parent_email, expires_at) VALUES ($1, '\x31', 'p@example.org', $2), ($1, '\x32', 'p@example.org', $3)`,
		seed.UserTom, clock.Add(-time.Minute), clock.Add(time.Hour))
	sessionsBefore := e.count(`SELECT count(*) FROM sessions`)

	res, err := privacy.Prune(ctx, e.pool, clock)
	if err != nil {
		t.Fatal(err)
	}
	want := privacy.PruneResult{PresenceVisits: 1, Tracks: 1, Reminders: 1, LoginTokens: 1, AuthSessions: 1, Invites: 1, ParentalTokens: 1}
	// Seed data may add older rows of its own (sessions, tokens); check the effect, not exact totals of those.
	if res.PresenceVisits < want.PresenceVisits || res.Tracks < want.Tracks || res.Reminders < want.Reminders ||
		res.LoginTokens != want.LoginTokens || res.AuthSessions != want.AuthSessions ||
		res.Invites != want.Invites || res.ParentalTokens != want.ParentalTokens {
		t.Errorf("result = %+v, want at least %+v", res, want)
	}
	if n := e.count(`SELECT count(*) FROM stable_invites WHERE code IN ('OLDCODE1', 'RECENT01', 'LIVE0001')`); n != 2 {
		t.Errorf("%d test invites remain, want 2", n)
	}
	if n := e.count(`SELECT count(*) FROM parental_consent_tokens WHERE user_id = $1`, seed.UserTom); n != 1 {
		t.Errorf("%d parental tokens remain, want 1", n)
	}
	if n := e.count(`SELECT count(*) FROM presence WHERE user_id = $1`, seed.UserTom); n != 1 {
		t.Errorf("Tom has %d visits, want only the recent one", n)
	}
	if n := e.count(`SELECT count(*) FROM presence WHERE user_id = $1 AND left_at IS NULL`, seed.UserAnna); n != 1 {
		t.Errorf("open visit was pruned")
	}
	if n := e.count(`SELECT count(*) FROM gait_windows g JOIN sessions s ON s.id = g.session_id WHERE s.started_at < $1`, clock.AddDate(0, -12, 0)); n != 0 {
		t.Errorf("%d old gait windows remain", n)
	}
	if n := e.count(`SELECT count(*) FROM gait_windows g JOIN sessions s ON s.id = g.session_id WHERE s.track = '[2]'::jsonb`); n != 1 {
		t.Errorf("recent gait windows were pruned")
	}
	if n := e.count(`SELECT count(*) FROM sessions WHERE track IS NOT NULL AND started_at < $1`, clock.AddDate(0, -12, 0)); n != 0 {
		t.Errorf("%d old tracks remain", n)
	}
	if n := e.count(`SELECT count(*) FROM sessions WHERE track = '[2]'::jsonb`); n != 1 {
		t.Errorf("recent track was removed")
	}
	if n := e.count(`SELECT count(*) FROM sessions`); n != sessionsBefore {
		t.Errorf("sessions were deleted: %d -> %d", sessionsBefore, n)
	}
	if n := e.count(`SELECT count(*) FROM login_tokens`); n != 1 {
		t.Errorf("%d login tokens remain, want 1 (the valid one)", n)
	}
	if n := e.count(`SELECT count(*) FROM auth_sessions WHERE user_id = $1`, seed.UserTom); n != 1 {
		t.Errorf("%d sessions remain, want 1 (the valid one)", n)
	}
	if n := e.count(`SELECT count(*) FROM reminders WHERE title = 'alt'`); n != 0 {
		t.Errorf("old reminder remains")
	}
	// Second run finds nothing.
	res, err = privacy.Prune(ctx, e.pool, clock)
	if err != nil || res != (privacy.PruneResult{}) {
		t.Errorf("second run = %+v, %v", res, err)
	}
}
