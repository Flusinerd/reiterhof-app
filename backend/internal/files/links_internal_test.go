package files

import (
	"strings"
	"testing"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

func TestLinkTokens(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	const user = "00000000-0000-4000-8000-000000000207"
	const path = "/api/v1/files/00000000-0000-4000-8000-000000000101/0123456789abcdef0123456789abcdef.png"
	token := signLink(key, user, path, now.Add(DownloadLinkTTL))

	if got, ok := verifyLink(key, token, path, now); !ok || got != user {
		t.Fatalf("verify = %q %v", got, ok)
	}
	if _, ok := verifyLink(key, token, path, now.Add(DownloadLinkTTL)); ok {
		t.Error("expired token accepted")
	}
	if _, ok := verifyLink(key, token, strings.Replace(path, ".png", ".jpg", 1), now); ok {
		t.Error("token accepted for another path")
	}
	if _, ok := verifyLink([]byte("another key another key another"), token, path, now); ok {
		t.Error("token accepted with another key")
	}
	parts := strings.Split(token, ".")
	forged := "9999999999." + parts[1] + "." + parts[2]
	if _, ok := verifyLink(key, forged, path, now); ok {
		t.Error("token with a changed expiry accepted")
	}
	for _, bad := range []string{"", "a.b", token + ".x", "x." + parts[1] + "." + parts[2]} {
		if _, ok := verifyLink(key, bad, path, now); ok {
			t.Errorf("malformed token %q accepted", bad)
		}
	}
}

func TestLinkKeyDerivation(t *testing.T) {
	withSecret := httpx.Deps{Config: config.Config{Auth: config.Auth{LoginCodeKey: "secret"}}}
	if a, b := linkKey(withSecret), linkKey(withSecret); string(a) != string(b) {
		t.Error("derived key is not stable")
	}
	if string(linkKey(withSecret)) == "secret" {
		t.Error("the login code key is used directly")
	}
	random := linkKey(httpx.Deps{})
	if len(random) != 32 || string(random) == string(linkKey(withSecret)) {
		t.Error("process key not random")
	}
	if string(linkKey(httpx.Deps{})) != string(random) {
		t.Error("process key changes between calls")
	}
}
