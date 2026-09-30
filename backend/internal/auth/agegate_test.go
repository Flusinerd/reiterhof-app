package auth_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
)

type ageResp struct {
	User struct {
		AgeStatus   string  `json:"age_status"`
		ParentEmail *string `json:"parent_email"`
	} `json:"user"`
}

var parentalLinkRe = regexp.MustCompile(`/parental-consent\?token=([A-Za-z0-9_-]+)`)

// postForm sends a browser form (the parent's button) without a session.
func (e *env) postForm(path string, values url.Values) *httptest.ResponseRecorder {
	e.t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *env) invite() string {
	e.t.Helper()
	admin := authtest.TokenAt(e.t, e.pool, seed.UserJan, *e.clock)
	rec := e.do("POST", "/api/v1/stables/invites", admin, nil)
	if rec.Code != 201 {
		e.t.Fatalf("invite = %d %s", rec.Code, rec.Body)
	}
	return decode[struct {
		Code string `json:"code"`
	}](e.t, rec).Code
}

func TestAgeConfirmationGatesTheStable(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code := e.invite()
	s := e.signIn("teen@example.org")
	if s.User.ID == "" {
		t.Fatal("no user")
	}
	me := decode[ageResp](t, e.do("GET", "/api/v1/me", s.Token, nil))
	if me.User.AgeStatus != auth.AgeStatusUnknown || me.User.ParentEmail != nil {
		t.Fatalf("new user = %+v", me.User)
	}
	// Nothing stated yet: no stable.
	assertError(t, e.do("POST", "/api/v1/stables/join", s.Token, map[string]string{"code": code}), 403, "age_unconfirmed")
	assertError(t, e.do("POST", "/api/v1/me/age", s.Token, map[string]any{"over_16": false}), 400, "validation_failed")
	assertError(t, e.do("POST", "/api/v1/me/age", s.Token, map[string]any{}), 400, "validation_failed")
	assertError(t, e.do("POST", "/api/v1/me/age", "", map[string]any{"over_16": true}), 401, "unauthorized")

	rec := e.do("POST", "/api/v1/me/age", s.Token, map[string]any{"over_16": true})
	if rec.Code != 200 || decode[ageResp](t, rec).User.AgeStatus != auth.AgeStatusConfirmed {
		t.Fatalf("confirm age = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("POST", "/api/v1/stables/join", s.Token, map[string]string{"code": code}); rec.Code != 200 {
		t.Fatalf("join after confirming = %d %s", rec.Code, rec.Body)
	}
	// Seed users are confirmed (adults in the demo).
	if me := decode[ageResp](t, e.do("GET", "/api/v1/me", authtest.TokenAt(t, e.pool, seed.UserAnna, *e.clock), nil)); me.User.AgeStatus != auth.AgeStatusConfirmed {
		t.Errorf("seed user = %+v", me.User)
	}
}

func TestParentalConsentFlow(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{PublicURL: "https://api.example.org", WebURL: "https://app.example.org"})
	code := e.invite()
	s := e.signIn("kid@example.org")
	if rec := e.do("PATCH", "/api/v1/me", s.Token, map[string]string{"name": "Kim"}); rec.Code != 200 {
		t.Fatal(rec.Body)
	}

	assertError(t, e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "kid@example.org"}), 400, "validation_failed")
	assertError(t, e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "not an address"}), 400, "validation_failed")
	assertError(t, e.do("POST", "/api/v1/me/parental-consent", "", map[string]string{"parent_email": "mum@example.org"}), 401, "unauthorized")

	if rec := e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "Mum@Example.org"}); rec.Code != 204 {
		t.Fatalf("request = %d %s", rec.Code, rec.Body)
	}
	me := decode[ageResp](t, e.do("GET", "/api/v1/me", s.Token, nil))
	if me.User.AgeStatus != auth.AgeStatusParentPending || me.User.ParentEmail == nil || *me.User.ParentEmail != "mum@example.org" {
		t.Fatalf("after request = %+v", me.User)
	}
	// Still no stable while the parent has not answered.
	assertError(t, e.do("POST", "/api/v1/stables/join", s.Token, map[string]string{"code": code}), 403, "age_unconfirmed")

	mail := e.mailer.last(t)
	if mail.To != "mum@example.org" || !strings.Contains(mail.Subject, "Kim") {
		t.Fatalf("mail = %q %q", mail.To, mail.Subject)
	}
	for _, must := range []string{"Kim", "kid@example.org", "jünger als 16", "https://app.example.org/legal/privacy", "7 Tage"} {
		if !strings.Contains(mail.Body, must) {
			t.Errorf("mail lacks %q:\n%s", must, mail.Body)
		}
	}
	m := parentalLinkRe.FindStringSubmatch(mail.Body)
	if m == nil || !strings.Contains(mail.Body, "https://api.example.org/parental-consent?token=") {
		t.Fatalf("no confirmation link in mail:\n%s", mail.Body)
	}
	token := m[1]

	// The page shows the parent whom it is about; reading it consumes nothing.
	page := e.do("GET", "/parental-consent?token="+token, "", nil)
	if page.Code != 200 || !strings.Contains(page.Body.String(), "Kim") || !strings.Contains(page.Body.String(), "kid@example.org") {
		t.Fatalf("page = %d %s", page.Code, page.Body)
	}
	if !strings.Contains(page.Body.String(), `name="token" value="`+token+`"`) || !strings.Contains(page.Body.String(), "https://app.example.org/legal/privacy") {
		t.Errorf("page lacks the form or the privacy link: %s", page.Body)
	}
	if csp := page.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "form-action 'self'") || !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("csp = %q", csp)
	}
	if page := e.do("GET", "/parental-consent?token=nope", "", nil); page.Code != 400 {
		t.Errorf("bad token page = %d", page.Code)
	}
	if me := decode[ageResp](t, e.do("GET", "/api/v1/me", s.Token, nil)); me.User.AgeStatus != auth.AgeStatusParentPending {
		t.Fatalf("GET consumed the token: %+v", me.User)
	}

	// The parent confirms.
	done := e.postForm("/parental-consent", url.Values{"token": {token}})
	if done.Code != 200 || !strings.Contains(done.Body.String(), "Danke") {
		t.Fatalf("confirm = %d %s", done.Code, done.Body)
	}
	me = decode[ageResp](t, e.do("GET", "/api/v1/me", s.Token, nil))
	if me.User.AgeStatus != auth.AgeStatusConfirmed || me.User.ParentEmail == nil {
		t.Fatalf("after confirm = %+v", me.User)
	}
	if rec := e.do("POST", "/api/v1/stables/join", s.Token, map[string]string{"code": code}); rec.Code != 200 {
		t.Fatalf("join after parental consent = %d %s", rec.Code, rec.Body)
	}
	// Once only.
	if again := e.postForm("/parental-consent", url.Values{"token": {token}}); again.Code != 400 {
		t.Errorf("second confirm = %d", again.Code)
	}
	assertError(t, e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "dad@example.org"}), 409, "already_confirmed")
}

func TestParentalConsentTokensExpireAndAreRevokedByNewRequests(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	s := e.signIn("kid@example.org")
	request := func() string {
		t.Helper()
		if rec := e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "mum@example.org"}); rec.Code != 204 {
			t.Fatalf("request = %d %s", rec.Code, rec.Body)
		}
		m := parentalLinkRe.FindStringSubmatch(e.mailer.last(t).Body)
		if m == nil {
			t.Fatalf("no link:\n%s", e.mailer.last(t).Body)
		}
		return m[1]
	}
	first := request()
	second := request()
	third := request()
	// Three mails a day, then a pause (stops the app being used to spam an address).
	assertError(t, e.do("POST", "/api/v1/me/parental-consent", s.Token, map[string]string{"parent_email": "mum@example.org"}), 429, "rate_limited")
	// Only the newest mail's link works.
	for _, old := range []string{first, second} {
		if rec := e.postForm("/parental-consent", url.Values{"token": {old}}); rec.Code != 400 {
			t.Errorf("revoked link = %d", rec.Code)
		}
	}
	// After seven days the link is dead.
	*e.clock = e.clock.Add(7*24*time.Hour + time.Minute)
	if rec := e.postForm("/parental-consent", url.Values{"token": {third}}); rec.Code != 400 {
		t.Errorf("expired link = %d", rec.Code)
	}
	if me := decode[ageResp](t, e.do("GET", "/api/v1/me", s.Token, nil)); me.User.AgeStatus != auth.AgeStatusParentPending {
		t.Errorf("status = %+v", me.User)
	}
	// A new day, a new mail; its link works.
	fresh := request()
	if rec := e.postForm("/parental-consent", url.Values{"token": {fresh}}); rec.Code != 200 {
		t.Errorf("fresh link = %d", rec.Code)
	}
	// Stating to be 16 or older afterwards keeps the parent's consent but withdraws a pending address.
	other := e.signIn("teen@example.org")
	e.do("POST", "/api/v1/me/parental-consent", other.Token, map[string]string{"parent_email": "dad@example.org"})
	rec := e.do("POST", "/api/v1/me/age", other.Token, map[string]any{"over_16": true})
	if me := decode[ageResp](t, rec); rec.Code != 200 || me.User.AgeStatus != auth.AgeStatusConfirmed || me.User.ParentEmail != nil {
		t.Errorf("over 16 after a parent request = %d %+v", rec.Code, me.User)
	}
	// A confirmation with a malformed form is a 400 page, not a crash.
	if rec := e.postForm("/parental-consent", url.Values{}); rec.Code != http.StatusBadRequest {
		t.Errorf("empty form = %d", rec.Code)
	}
}
