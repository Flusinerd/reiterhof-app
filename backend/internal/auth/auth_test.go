package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

type captureMailer struct {
	mu   sync.Mutex
	msgs []auth.Message
}

func (m *captureMailer) Send(_ context.Context, msg auth.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *captureMailer) last(t *testing.T) auth.Message {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.msgs) == 0 {
		t.Fatal("no mail was sent")
	}
	return m.msgs[len(m.msgs)-1]
}

func (m *captureMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

var tokenRe = regexp.MustCompile(`stallfunk://auth/verify\?token=([A-Za-z0-9_-]+)`)

type env struct {
	t      *testing.T
	pool   *pgxpool.Pool
	h      http.Handler
	mailer *captureMailer
	clock  *time.Time
}

// newEnv builds the auth routes plus a protected probe route behind the middleware.
func newEnv(t *testing.T, opts auth.Options, cfg config.Auth) *env {
	t.Helper()
	pool := dbtest.NewSeeded(t)
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	e := &env{t: t, pool: pool, mailer: &captureMailer{}, clock: &now}
	deps := httpx.Deps{Pool: pool, Config: config.Config{Auth: cfg}, Now: func() time.Time { return *e.clock }}
	opts.Mailer = e.mailer
	mux := http.NewServeMux()
	auth.NewService(deps, opts).Register(mux)
	mux.Handle("GET /probe/stable", auth.RequireStable(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := auth.UserFrom(r.Context())
		httpx.WriteJSON(w, 200, map[string]string{"id": u.ID, "stable_id": u.StableID})
	})))
	mux.Handle("GET /probe/user", auth.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
	e.h = auth.Middleware(deps)(mux)
	return e
}

func (e *env) do(method, path, token string, body any) *httptest.ResponseRecorder {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.RemoteAddr = "192.0.2.1:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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

type session struct {
	Token string `json:"token"`
	User  struct {
		ID       string  `json:"id"`
		Email    string  `json:"email"`
		Name     string  `json:"name"`
		StableID *string `json:"stable_id"`
		IsAdmin  bool    `json:"is_admin"`
		// NameConfirmed is false while the name is the email fallback.
		NameConfirmed bool `json:"name_confirmed"`
	} `json:"user"`
}

func (e *env) requestLink(email string) string {
	e.t.Helper()
	if rec := e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": email}); rec.Code != 204 {
		e.t.Fatalf("magic-link status = %d: %s", rec.Code, rec.Body)
	}
	m := tokenRe.FindStringSubmatch(e.mailer.last(e.t).Body)
	if m == nil {
		e.t.Fatalf("no app link in mail: %s", e.mailer.last(e.t).Body)
	}
	return m[1]
}

func (e *env) signIn(email string) session {
	e.t.Helper()
	rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": e.requestLink(email)})
	if rec.Code != 200 {
		e.t.Fatalf("verify status = %d: %s", rec.Code, rec.Body)
	}
	return decode[session](e.t, rec)
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d: %s", rec.Code, status, rec.Body)
	}
	var b httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.Error.Code != code {
		t.Fatalf("error = %s, want code %q", rec.Body, code)
	}
}

func TestMagicLinkNewUser(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{PublicURL: "https://api.example.org"})
	s := e.signIn("New.Person@Example.org")
	if s.Token == "" || s.User.Email != "new.person@example.org" || s.User.Name != "new.person" || s.User.StableID != nil {
		t.Fatalf("session = %+v", s)
	}
	if !strings.Contains(e.mailer.last(t).Body, "https://api.example.org/auth/verify?token=") {
		t.Fatalf("mail lacks https fallback: %s", e.mailer.last(t).Body)
	}
	// Without stable: /me works, domain routes are 403.
	if rec := e.do("GET", "/api/v1/me", s.Token, nil); rec.Code != 200 {
		t.Fatalf("/me = %d", rec.Code)
	}
	assertError(t, e.do("GET", "/probe/stable", s.Token, nil), 403, "no_stable")
	// Only the hash is stored.
	var n int
	if err := e.pool.QueryRow(context.Background(), `SELECT count(*) FROM auth_sessions WHERE token_hash = $1`, []byte(s.Token)).Scan(&n); err != nil || n != 0 {
		t.Fatalf("plain token stored? n=%d err=%v", n, err)
	}
}

func TestMagicLinkExistingSeedUser(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	s := e.signIn("jan@example.org")
	if s.User.ID != seed.UserJan || !s.User.IsAdmin || s.User.StableID == nil || *s.User.StableID != seed.StableB {
		t.Fatalf("session = %+v", s)
	}
	rec := e.do("GET", "/api/v1/me", s.Token, nil)
	me := decode[struct {
		Stable *struct{ ID, Name string } `json:"stable"`
		Roles  struct {
			Owned []string `json:"owned_horse_ids"`
			Rider []string `json:"rider_horse_ids"`
		} `json:"roles"`
	}](t, rec)
	if me.Stable == nil || me.Stable.ID != seed.StableB || len(me.Roles.Owned) != 1 || me.Roles.Owned[0] != seed.HorseLuna || me.Roles.Rider == nil {
		t.Fatalf("me = %s", rec.Body)
	}
	if rec := e.do("GET", "/probe/stable", s.Token, nil); rec.Code != 200 {
		t.Fatalf("probe = %d", rec.Code)
	}
}

func TestMagicLinkNoEnumerationAndValidation(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	for _, email := range []string{"jan@example.org", "nobody-here@example.org"} {
		if rec := e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": email}); rec.Code != 204 || rec.Body.Len() != 0 {
			t.Fatalf("%s: status %d body %q", email, rec.Code, rec.Body)
		}
	}
	for _, bad := range []string{"", "not-an-email", "Jan <jan@example.org>", "a@b.c\r\nBcc: x@y.z"} {
		assertError(t, e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": bad}), 400, "validation_failed")
	}
}

func TestMagicLinkRateLimit(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	for i := 0; i < 3; i++ {
		if rec := e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": "spam@example.org"}); rec.Code != 204 {
			t.Fatalf("request %d = %d", i, rec.Code)
		}
	}
	assertError(t, e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": "spam@example.org"}), 429, "rate_limited")
	if e.mailer.count() != 3 {
		t.Fatalf("mails = %d, want 3", e.mailer.count())
	}
	// After the window the address may ask again.
	*e.clock = e.clock.Add(16 * time.Minute)
	if rec := e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": "spam@example.org"}); rec.Code != 204 {
		t.Fatalf("after window = %d", rec.Code)
	}
}

func TestMagicLinkExpiredAndUsed(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	token := e.requestLink("late@example.org")
	*e.clock = e.clock.Add(16 * time.Minute)
	assertError(t, e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}), 401, "invalid_token")

	*e.clock = e.clock.Add(-16 * time.Minute)
	token = e.requestLink("once@example.org")
	if rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}); rec.Code != 200 {
		t.Fatalf("first use = %d", rec.Code)
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}), 401, "invalid_token")
	assertError(t, e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": "made-up"}), 401, "invalid_token")
}

func TestVerifyPageDoesNotConsumeToken(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	token := e.requestLink("page@example.org")
	rec := e.do("GET", "/auth/verify?token="+token, "", nil)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `href="stallfunk://auth/verify?token=`+token+`"`) {
		t.Fatalf("page = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}); rec.Code != 200 {
		t.Fatalf("token consumed by GET: %d", rec.Code)
	}
	if rec := e.do("GET", "/auth/verify?token=%22%3E%3Cscript%3E", "", nil); rec.Code != 400 {
		t.Fatalf("hostile token = %d", rec.Code)
	}
}

func TestSocialSignIn(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	google := newTestIdP(t, "https://accounts.google.com", "google-client", now)
	apple := newTestIdP(t, "https://appleid.apple.com", "org.example.app", now)
	e := newEnv(t, auth.Options{Google: google.verifier(), Apple: apple.verifier()}, config.Auth{})

	// New Google user, then the same subject signs in again: same account.
	tok := google.sign(t, google.key, map[string]any{"sub": "g-1", "email": "Fresh@Example.org", "name": "Fresh Rider"})
	rec := e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": tok})
	if rec.Code != 200 {
		t.Fatalf("google = %d %s", rec.Code, rec.Body)
	}
	first := decode[session](t, rec)
	if first.User.Email != "fresh@example.org" || first.User.Name != "Fresh Rider" || first.User.StableID != nil {
		t.Fatalf("user = %+v", first.User)
	}
	second := decode[session](t, e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": tok}))
	if second.User.ID != first.User.ID || second.Token == first.Token {
		t.Fatalf("second sign-in: %+v", second)
	}

	// Merge by verified email: Jan already exists, Google links to him.
	tok = google.sign(t, google.key, map[string]any{"sub": "g-jan", "email": "jan@example.org"})
	merged := decode[session](t, e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": tok}))
	if merged.User.ID != seed.UserJan || !merged.User.IsAdmin {
		t.Fatalf("merge failed: %+v", merged.User)
	}
	// ... and Apple with the same email merges into the same account, name comes from the request only for new users.
	tok = apple.sign(t, apple.key, map[string]any{"sub": "a-jan", "email": "jan@example.org", "email_verified": "true"})
	mergedApple := decode[session](t, e.do("POST", "/api/v1/auth/apple", "", map[string]any{"id_token": tok, "name": "Other Name"}))
	if mergedApple.User.ID != seed.UserJan || mergedApple.User.Name != "Jan" {
		t.Fatalf("apple merge: %+v", mergedApple.User)
	}

	// Apple relay addresses are ordinary emails: own account, name from the request.
	tok = apple.sign(t, apple.key, map[string]any{"sub": "a-relay", "email": "abc123@privaterelay.appleid.com", "email_verified": "true"})
	relay := decode[session](t, e.do("POST", "/api/v1/auth/apple", "", map[string]any{"id_token": tok, "name": "Relay Person"}))
	if relay.User.Name != "Relay Person" || relay.User.Email != "abc123@privaterelay.appleid.com" || relay.User.ID == seed.UserJan {
		t.Fatalf("relay: %+v", relay.User)
	}

	// Rejections.
	other := newTestIdP(t, "https://accounts.google.com", "google-client", now)
	bad := map[string]string{
		"bad signature": google.sign(t, other.key, nil),
		"wrong aud":     google.sign(t, google.key, map[string]any{"aud": "someone-else"}),
		"expired":       google.sign(t, google.key, map[string]any{"exp": now.Add(-time.Hour).Unix()}),
		"garbage":       "x.y.z",
	}
	for name, tok := range bad {
		if rec := e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": tok}); rec.Code != 401 {
			t.Errorf("%s: status %d, want 401", name, rec.Code)
		}
	}
	// An Apple token is not accepted by the Google endpoint (issuer and key differ).
	assertError(t, e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": apple.sign(t, apple.key, nil)}), 401, "invalid_token")
	// Unverified email cannot create or merge accounts.
	tok = google.sign(t, google.key, map[string]any{"sub": "g-unverified", "email": "jan@example.org", "email_verified": false})
	assertError(t, e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": tok}), 401, "email_not_verified")
}

func TestSocialNotConfigured(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	assertError(t, e.do("POST", "/api/v1/auth/google", "", map[string]string{"id_token": "x.y.z"}), 503, "not_configured")
	assertError(t, e.do("POST", "/api/v1/auth/apple", "", map[string]string{"id_token": "x.y.z"}), 503, "not_configured")
}

func TestSessionsAndMiddleware(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	assertError(t, e.do("GET", "/probe/user", "", nil), 401, "unauthorized")
	assertError(t, e.do("GET", "/probe/user", "garbage-token", nil), 401, "unauthorized")
	assertError(t, e.do("GET", "/probe/stable", "", nil), 401, "unauthorized")
	assertError(t, e.do("GET", "/api/v1/me", "", nil), 401, "unauthorized")

	token := authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock)
	if rec := e.do("GET", "/probe/stable", token, nil); rec.Code != 200 {
		t.Fatalf("valid token = %d", rec.Code)
	}

	// Sliding expiry: use after > 1h extends the session past the original 90 days.
	*e.clock = e.clock.Add(80 * 24 * time.Hour)
	if rec := e.do("GET", "/probe/user", token, nil); rec.Code != 204 {
		t.Fatalf("day 80 = %d", rec.Code)
	}
	*e.clock = e.clock.Add(80 * 24 * time.Hour)
	if rec := e.do("GET", "/probe/user", token, nil); rec.Code != 204 {
		t.Fatalf("day 160 (slid) = %d", rec.Code)
	}
	// Unused for 90 days: expired.
	*e.clock = e.clock.Add(91 * 24 * time.Hour)
	assertError(t, e.do("GET", "/probe/user", token, nil), 401, "unauthorized")

	// Logout invalidates only that session.
	a := authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock)
	b := authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock)
	if rec := e.do("POST", "/api/v1/auth/logout", a, nil); rec.Code != 204 {
		t.Fatalf("logout = %d", rec.Code)
	}
	assertError(t, e.do("GET", "/probe/user", a, nil), 401, "unauthorized")
	if rec := e.do("GET", "/probe/user", b, nil); rec.Code != 204 {
		t.Fatalf("other session = %d", rec.Code)
	}
}

func TestNameConfirmed(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	s := e.signIn("new.rider@example.org")
	if s.User.Name != "new.rider" || s.User.NameConfirmed {
		t.Fatalf("new magic-link user = %+v, want the email fallback, unconfirmed", s.User)
	}
	// Other profile fields do not confirm the name.
	if rec := e.do("PATCH", "/api/v1/me", s.Token, map[string]string{"phone": "0170 1"}); rec.Code != 200 || decode[meResp](t, rec).User.NameConfirmed {
		t.Fatalf("patch phone = %d %s", rec.Code, rec.Body)
	}
	rec := e.do("PATCH", "/api/v1/me", s.Token, map[string]string{"name": "Nina"})
	if me := decode[meResp](t, rec); rec.Code != 200 || me.User.Name != "Nina" || !me.User.NameConfirmed {
		t.Fatalf("patch name = %d %s", rec.Code, rec.Body)
	}
	// Seeded and admin-created users have a real name.
	if me := decode[meResp](t, e.do("GET", "/api/v1/me", authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock), nil)); !me.User.NameConfirmed {
		t.Fatalf("seed user unconfirmed: %+v", me.User)
	}
}

// meResp is the part of GET/PATCH /api/v1/me the tests read.
type meResp struct {
	User struct {
		Name          string `json:"name"`
		NameConfirmed bool   `json:"name_confirmed"`
	} `json:"user"`
}

func TestPatchMe(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	token := authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock)
	rec := e.do("PATCH", "/api/v1/me", token, map[string]string{"name": " Anna B. ", "phone": "0170 1234", "avatar_color": "teal", "presence_visibility": "only_day"})
	if rec.Code != 200 {
		t.Fatalf("patch = %d %s", rec.Code, rec.Body)
	}
	var name, phone, color, vis string
	if err := e.pool.QueryRow(context.Background(), `SELECT name, phone, avatar_color, presence_visibility FROM users WHERE id = $1`, seed.UserAnna).Scan(&name, &phone, &color, &vis); err != nil {
		t.Fatal(err)
	}
	if name != "Anna B." || phone != "0170 1234" || color != "teal" || vis != "only_day" {
		t.Fatalf("stored = %q %q %q %q", name, phone, color, vis)
	}
	// Absent fields stay, empty phone clears.
	e.do("PATCH", "/api/v1/me", token, map[string]string{"phone": ""})
	var p *string
	_ = e.pool.QueryRow(context.Background(), `SELECT phone FROM users WHERE id = $1`, seed.UserAnna).Scan(&p)
	if p != nil {
		t.Fatalf("phone = %q, want NULL", *p)
	}
	assertError(t, e.do("PATCH", "/api/v1/me", token, map[string]string{"presence_visibility": "everyone"}), 400, "validation_failed")
	assertError(t, e.do("PATCH", "/api/v1/me", token, map[string]string{"name": " "}), 400, "validation_failed")
	assertError(t, e.do("PATCH", "/api/v1/me", token, map[string]string{"is_admin": "true"}), 400, "invalid_json")
}

func TestInviteAndJoin(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	admin := authtest.TokenAt(t, e.pool, seed.UserJan, *e.clock)
	member := authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock)

	assertError(t, e.do("POST", "/api/v1/stables/invites", member, nil), 403, "forbidden")
	assertError(t, e.do("POST", "/api/v1/stables/invites", "", nil), 401, "unauthorized")

	rec := e.do("POST", "/api/v1/stables/invites", admin, map[string]int{"max_uses": 1})
	if rec.Code != 201 {
		t.Fatalf("invite = %d %s", rec.Code, rec.Body)
	}
	code := decode[struct {
		Code string `json:"code"`
	}](t, rec).Code
	if !regexp.MustCompile(`^[A-Z2-9]{4}-[A-Z2-9]{4}$`).MatchString(code) {
		t.Fatalf("code format: %q", code)
	}

	newbie := e.signIn("newbie@example.org")
	assertError(t, e.do("POST", "/api/v1/stables/join", newbie.Token, map[string]string{"code": "WRONG-CODE"}), 404, "invalid_code")
	// Typed sloppily (lower case, spaces) still works.
	rec = e.do("POST", "/api/v1/stables/join", newbie.Token, map[string]string{"code": " " + strings.ToLower(code[:4]) + " " + code[5:] + " "})
	if rec.Code != 200 {
		t.Fatalf("join = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("GET", "/probe/stable", newbie.Token, nil); rec.Code != 200 {
		t.Fatalf("after join probe = %d", rec.Code)
	}
	assertError(t, e.do("POST", "/api/v1/stables/join", newbie.Token, map[string]string{"code": code}), 409, "already_in_stable")

	// max_uses 1: used up.
	second := e.signIn("second@example.org")
	assertError(t, e.do("POST", "/api/v1/stables/join", second.Token, map[string]string{"code": code}), 404, "invalid_code")

	// Expired invite.
	rec = e.do("POST", "/api/v1/stables/invites", admin, map[string]int{"expires_in_days": 1})
	code = decode[struct {
		Code string `json:"code"`
	}](t, rec).Code
	*e.clock = e.clock.Add(25 * time.Hour)
	assertError(t, e.do("POST", "/api/v1/stables/join", second.Token, map[string]string{"code": code}), 404, "invalid_code")
}

func TestDevLogin(t *testing.T) {
	off := newEnv(t, auth.Options{}, config.Auth{})
	assertError(t, off.do("POST", "/api/v1/auth/dev-login", "", map[string]string{"email": "jan@example.org"}), 404, "not_found")
	on := newEnv(t, auth.Options{}, config.Auth{DevLogin: true})
	rec := on.do("POST", "/api/v1/auth/dev-login", "", map[string]string{"email": "tom@example.org"})
	if rec.Code != 200 || decode[session](t, rec).User.ID != seed.UserTom {
		t.Fatalf("dev-login = %d %s", rec.Code, rec.Body)
	}
}

// --- roles ---------------------------------------------------------------------

func TestRoleChecks(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	user := func(id string) auth.User {
		u := auth.User{ID: id, StableID: seed.StableB}
		if id == seed.UserJan {
			u.IsAdmin = true
		}
		return u
	}
	check := func(name string, got bool, err error, want bool) {
		t.Helper()
		if err != nil || got != want {
			t.Errorf("%s = %v (err %v), want %v", name, got, err, want)
		}
	}

	// Luna: owner Jan (also admin), rider Mia. Fanta: owner Anna, rider Lea.
	ok, err := auth.CanManageHorse(ctx, pool, user(seed.UserJan), seed.HorseLuna)
	check("owner manages own", ok, err, true)
	ok, err = auth.CanManageHorse(ctx, pool, user(seed.UserAnna), seed.HorseLuna)
	check("other member manages Luna", ok, err, false)
	ok, err = auth.CanManageHorse(ctx, pool, user(seed.UserMia), seed.HorseLuna)
	check("rider manages", ok, err, false)
	ok, err = auth.CanManageHorse(ctx, pool, user(seed.UserJan), seed.HorseFanta)
	check("admin manages any horse in stable", ok, err, true)
	ok, err = auth.CanManageHorse(ctx, pool, user(seed.UserAnna), seed.HorseFanta)
	check("owner Anna manages Fanta", ok, err, true)

	ok, err = auth.IsRider(ctx, pool, user(seed.UserMia), seed.HorseLuna)
	check("Mia rider of Luna", ok, err, true)
	ok, err = auth.IsRider(ctx, pool, user(seed.UserMia), seed.HorseFanta)
	check("Mia rider of Fanta", ok, err, false)
	ok, err = auth.IsRider(ctx, pool, user(seed.UserJan), seed.HorseLuna)
	check("owner is not implicitly a rider", ok, err, false)

	rules, isRider, err := auth.RiderRules(ctx, pool, user(seed.UserMia), seed.HorseLuna)
	if err != nil || !isRider || strings.Join(rules, ",") != "ride,groom" {
		t.Errorf("RiderRules = %v %v %v", rules, isRider, err)
	}
	if _, isRider, err = auth.RiderRules(ctx, pool, user(seed.UserMia), seed.HorseFanta); err != nil || isRider {
		t.Errorf("RiderRules for non-rider = %v %v", isRider, err)
	}

	ok, err = auth.HorseInStable(ctx, pool, user(seed.UserAnna), seed.HorseBalu)
	check("horse in stable", ok, err, true)
	ok, err = auth.HorseInStable(ctx, pool, user(seed.UserAnna), "not-a-uuid")
	check("garbage id", ok, err, false)

	// No stable: everything is false.
	nobody := auth.User{ID: seed.UserJan, IsAdmin: true}
	ok, err = auth.CanManageHorse(ctx, pool, nobody, seed.HorseLuna)
	check("admin without stable", ok, err, false)
}

func TestCrossStableAccessDenied(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	ctx := context.Background()
	const stableX, userX, horseX = "10000000-0000-4000-8000-000000000001", "10000000-0000-4000-8000-000000000002", "10000000-0000-4000-8000-000000000003"
	for _, q := range []string{
		`INSERT INTO stables (id, name) VALUES ('` + stableX + `', 'Other Stable')`,
		`INSERT INTO users (id, stable_id, name, email, is_admin) VALUES ('` + userX + `', '` + stableX + `', 'Xavier', 'x@example.org', true)`,
		`INSERT INTO horses (id, stable_id, name, owner_id) VALUES ('` + horseX + `', '` + stableX + `', 'Xeno', '` + userX + `')`,
		`INSERT INTO horse_riders (stable_id, horse_id, user_id, rules) VALUES ('` + stableX + `', '` + horseX + `', '` + userX + `', '["ride"]')`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	admin := auth.User{ID: seed.UserJan, StableID: seed.StableB, IsAdmin: true}
	foreign := auth.User{ID: userX, StableID: stableX, IsAdmin: true}

	for name, fn := range map[string]func() (bool, error){
		"admin of B manages horse of X":  func() (bool, error) { return auth.CanManageHorse(ctx, pool, admin, horseX) },
		"horse of X is in stable B":      func() (bool, error) { return auth.HorseInStable(ctx, pool, admin, horseX) },
		"admin of X manages horse of B":  func() (bool, error) { return auth.CanManageHorse(ctx, pool, foreign, seed.HorseLuna) },
		"owner X is rider check on Luna": func() (bool, error) { return auth.IsRider(ctx, pool, foreign, seed.HorseLuna) },
		"stable B user claiming X's stable id is still checked against own": func() (bool, error) {
			return auth.IsOwner(ctx, pool, auth.User{ID: userX, StableID: seed.StableB}, horseX)
		},
	} {
		if ok, err := fn(); err != nil || ok {
			t.Errorf("%s = %v (err %v), want false", name, ok, err)
		}
	}
	if ok, err := auth.CanManageHorse(ctx, pool, foreign, horseX); err != nil || !ok {
		t.Errorf("admin of X manages own horse = %v %v", ok, err)
	}
}
