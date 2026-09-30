package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
)

// The https fallback page needs no database: verifyPage only reads the query.
func TestVerifyPage(t *testing.T) {
	s := &Service{}
	rec := httptest.NewRecorder()
	s.verifyPage(rec, httptest.NewRequest("GET", "/auth/verify?token=tok_EN-1234567890abcdefghij", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`href="stallfunk://auth/verify?token=tok_EN-1234567890abcdefghij"`,
		`src="data:image/png;base64,`,
		"In der App anmelden",
		"Öffne diesen Link auf dem Gerät mit der Stallfunk-App.",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body lacks %q:\n%s", want, body)
		}
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if strings.Contains(body, " style=") {
		t.Error("inline style attributes are blocked by the CSP; use the stylesheet")
	}

	// The CSP must allow exactly the stylesheet that is in the page, by its hash.
	csp := rec.Header().Get("Content-Security-Policy")
	start, end := strings.Index(body, "<style>"), strings.Index(body, "</style>")
	if start < 0 || end < 0 {
		t.Fatalf("no <style> block in %s", body)
	}
	sum := sha256.Sum256([]byte(body[start+len("<style>") : end]))
	want := "style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	for _, directive := range []string{"default-src 'none'", want, "img-src data:", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP %q lacks %q", csp, directive)
		}
	}
}

func TestVerifyPageRejectsBadToken(t *testing.T) {
	s := &Service{}
	for _, token := range []string{"", "short", "has space and is long enough", "<script>alert(1)</script>x"} {
		rec := httptest.NewRecorder()
		s.verifyPage(rec, httptest.NewRequest("GET", "/auth/verify?token="+strings.ReplaceAll(token, " ", "%20"), nil))
		if rec.Code != 400 {
			t.Errorf("token %q: status = %d, want 400", token, rec.Code)
		}
	}
}
