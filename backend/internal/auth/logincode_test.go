package auth_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
)

var (
	codeLineRe    = regexp.MustCompile(`(?m)^(\d{3}) (\d{3})$`)
	codeSubjectRe = regexp.MustCompile(`: (\d{6})$`)
)

// requestCode asks for a link and returns the code and link token of the mail.
func (e *env) requestCode(email string) (code, token string) {
	e.t.Helper()
	token = e.requestLink(email)
	msg := e.mailer.last(e.t)
	m := codeLineRe.FindStringSubmatch(msg.Body)
	if m == nil {
		e.t.Fatalf("no code line in mail: %s", msg.Body)
	}
	return m[1] + m[2], token
}

func (e *env) verifyCode(email, code string) int {
	e.t.Helper()
	return e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": email, "code": code}).Code
}

func wrongCode(code string) string {
	if code == "000000" {
		return "000001"
	}
	return "000000"
}

func TestLoginCodeMail(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{PublicURL: "https://api.example.org"})
	code, token := e.requestCode("code@example.org")
	msg := e.mailer.last(t)
	if m := codeSubjectRe.FindStringSubmatch(msg.Subject); m == nil || m[1] != code {
		t.Fatalf("subject %q does not carry code %s", msg.Subject, code)
	}
	if !strings.HasPrefix(msg.Subject, "Dein Stallfunk-Code: ") {
		t.Fatalf("subject = %q", msg.Subject)
	}
	// Code on its own line, then the links, and the validity hint.
	codeAt, linkAt := -1, -1
	for i, l := range strings.Split(msg.Body, "\n") {
		if l == code[:3]+" "+code[3:] {
			codeAt = i
		}
		if strings.HasPrefix(l, "stallfunk://auth/verify?token="+token) {
			linkAt = i
		}
	}
	if codeAt < 0 || linkAt < 0 || codeAt > linkAt {
		t.Fatalf("code line %d, link line %d in:\n%s", codeAt, linkAt, msg.Body)
	}
	if !strings.Contains(msg.Body, "https://api.example.org/auth/verify?token="+token) || !strings.Contains(msg.Body, "15 Minuten") {
		t.Fatalf("mail lacks fallback link or validity: %s", msg.Body)
	}
}

func TestVerifyCodeHappyPath(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, _ := e.requestCode("Jan@Example.org") // email is case-insensitive
	rec := e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "jan@example.org", "code": code[:3] + " " + code[3:]})
	if rec.Code != 200 {
		t.Fatalf("verify-code = %d: %s", rec.Code, rec.Body)
	}
	s := decode[session](t, rec)
	if s.Token == "" || s.User.Email != "jan@example.org" || !s.User.IsAdmin {
		t.Fatalf("session = %+v", s)
	}
	if rec := e.do("GET", "/probe/stable", s.Token, nil); rec.Code != 200 {
		t.Fatalf("session token unusable: %d", rec.Code)
	}
	// Single use.
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "jan@example.org", "code": code}), 401, "invalid_code")
}

func TestVerifyCodeNewUser(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, _ := e.requestCode("brandnew@example.org")
	rec := e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "brandnew@example.org", "code": code})
	if rec.Code != 200 {
		t.Fatalf("verify-code = %d: %s", rec.Code, rec.Body)
	}
	if s := decode[session](t, rec); s.User.StableID != nil || s.User.Name != "brandnew" {
		t.Fatalf("user = %+v", s.User)
	}
}

func TestVerifyCodeLockout(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, token := e.requestCode("lock@example.org")
	bad := wrongCode(code)
	for i := 0; i < 5; i++ {
		assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "lock@example.org", "code": bad}), 401, "invalid_code")
	}
	// Even the right code is dead now.
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "lock@example.org", "code": code}), 401, "invalid_code")
	// The link of the same attempt is unaffected (it has 256 bits and cannot be guessed).
	if rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}); rec.Code != 200 {
		t.Fatalf("link after code lockout = %d", rec.Code)
	}
}

func TestVerifyCodeFourWrongThenRight(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, _ := e.requestCode("close@example.org")
	for i := 0; i < 4; i++ {
		if got := e.verifyCode("close@example.org", wrongCode(code)); got != 401 {
			t.Fatalf("wrong code = %d", got)
		}
	}
	if got := e.verifyCode("close@example.org", code); got != 200 {
		t.Fatalf("right code after 4 misses = %d", got)
	}
}

func TestVerifyCodeExpired(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, _ := e.requestCode("late2@example.org")
	*e.clock = e.clock.Add(14*time.Minute + 59*time.Second)
	if got := e.verifyCode("late2@example.org", code); got != 200 {
		t.Fatalf("just before expiry = %d", got)
	}
	code, _ = e.requestCode("late3@example.org")
	*e.clock = e.clock.Add(15*time.Minute + time.Second)
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "late3@example.org", "code": code}), 401, "invalid_code")
}

func TestVerifyCodeAfterLinkUsed(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, token := e.requestCode("both@example.org")
	if rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}); rec.Code != 200 {
		t.Fatalf("link = %d", rec.Code)
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "both@example.org", "code": code}), 401, "invalid_code")

	// And the other way round: the code consumes the link.
	code, token = e.requestCode("both2@example.org")
	if got := e.verifyCode("both2@example.org", code); got != 200 {
		t.Fatalf("code = %d", got)
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": token}), 401, "invalid_token")
}

func TestVerifyCodeOtherEmailAndUnknown(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	code, _ := e.requestCode("mine@example.org")
	_, _ = e.requestCode("theirs@example.org")
	// A valid code of one address does not work for another one.
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "theirs@example.org", "code": code}), 401, "invalid_code")
	// Unknown email and wrong code answer identically.
	a := e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "nobody@example.org", "code": code})
	b := e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "mine@example.org", "code": wrongCode(code)})
	assertError(t, a, 401, "invalid_code")
	if a.Body.String() != b.Body.String() {
		t.Fatalf("unknown email %q differs from wrong code %q", a.Body, b.Body)
	}
	for _, bad := range []string{"", "12345", "1234567", "12a456", "１２３４５６"} {
		assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "mine@example.org", "code": bad}), 401, "invalid_code")
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "not-an-email", "code": code}), 401, "invalid_code")
	// Malformed input does not count as an attempt; two real misses above (theirs, mine) are
	// far below the limit, so the real code still works.
	if got := e.verifyCode("mine@example.org", code); got != 200 {
		t.Fatalf("own code = %d", got)
	}
}

func TestNewRequestRevokesOlderCode(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	oldCode, oldToken := e.requestCode("again@example.org")
	*e.clock = e.clock.Add(time.Minute)
	newCode, _ := e.requestCode("again@example.org")
	if oldCode == newCode {
		t.Skip("codes collided (1 in 10^6)")
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "again@example.org", "code": oldCode}), 401, "invalid_code")
	if got := e.verifyCode("again@example.org", newCode); got != 200 {
		t.Fatalf("new code = %d", got)
	}
	// The older link was not touched by the code revocation, but the code consumed the newer attempt only.
	if rec := e.do("POST", "/api/v1/auth/verify", "", map[string]string{"token": oldToken}); rec.Code != 200 {
		t.Fatalf("old link = %d", rec.Code)
	}
}

func TestVerifyCodeRateLimitPerIP(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	for i := 0; i < 30; i++ {
		assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "x@example.org", "code": "123456"}), 401, "invalid_code")
	}
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "x@example.org", "code": "123456"}), 429, "rate_limited")
	*e.clock = e.clock.Add(16 * time.Minute)
	assertError(t, e.do("POST", "/api/v1/auth/verify-code", "", map[string]string{"email": "x@example.org", "code": "123456"}), 401, "invalid_code")
}

func TestMagicLinkRateLimitStillAppliesToCodes(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{})
	for i := 0; i < 3; i++ {
		e.requestCode("flood@example.org")
	}
	assertError(t, e.do("POST", "/api/v1/auth/magic-link", "", map[string]string{"email": "flood@example.org"}), 429, "rate_limited")
	if e.mailer.count() != 3 {
		t.Fatalf("mails = %d, want 3", e.mailer.count())
	}
}

func TestLoginCodeStoredOnlyAsHash(t *testing.T) {
	e := newEnv(t, auth.Options{}, config.Auth{LoginCodeKey: "test-key"})
	e.requestCode("hash@example.org")
	var n int
	if err := e.pool.QueryRow(t.Context(),
		`SELECT count(*) FROM login_tokens WHERE email = 'hash@example.org' AND length(code_hash) = 32`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("hash row count = %d, err %v", n, err)
	}
}
