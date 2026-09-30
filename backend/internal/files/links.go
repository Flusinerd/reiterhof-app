package files

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Download links. Some consumers cannot send the Authorization header: Linking.openURL hands a
// PDF or a full-size photo to the browser or a viewer app. Putting the session token into such
// a URL would leave the key to the whole account in browser histories (synced to the cloud on
// iOS), share sheets and proxy logs. Instead the app asks for a link first:
//
//	POST /api/v1/files/download-link {"url": "/api/v1/files/<path>"}
//	-> {"url": "/api/v1/files/<path>?dl=<token>", "expires_at": ...}
//
// The token names the user, the exact request path and an expiry (DownloadLinkTTL) and is
// signed with an HMAC key derived from REITERHOF_LOGIN_CODE_KEY (or a random per-process
// key). It only authenticates the user for that one path; the route's own role checks still
// run. Nothing else in the API accepts a token in the query string.

// DownloadLinkTTL is how long a download link works.
const DownloadLinkTTL = 5 * time.Minute

// downloadable are the request paths a link may be minted for: the raw file route and the
// role-checked document route.
var downloadable = regexp.MustCompile(`^/api/v1/(files/` + uuidPattern + `/[0-9a-f]{32}\.(?:jpg|png|webp|pdf)` +
	`|horses/` + uuidPattern + `/documents/` + uuidPattern + `/file)$`)

var (
	processKeyOnce sync.Once
	processKey     []byte
)

// linkKey derives the signing key. The login code key is the one secret the operator sets;
// deriving with a label keeps the two uses apart. Without it links stop working at a
// restart, which is fine for a five-minute token.
func linkKey(deps httpx.Deps) []byte {
	if secret := deps.Config.Auth.LoginCodeKey; secret != "" {
		sum := sha256.Sum256([]byte("stallfunk/download-link/v1\x00" + secret))
		return sum[:]
	}
	processKeyOnce.Do(func() {
		processKey = make([]byte, 32)
		if _, err := rand.Read(processKey); err != nil {
			panic("files: no randomness for the download link key: " + err.Error())
		}
	})
	return processKey
}

func linkSignature(key []byte, userID, path string, exp int64) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(strconv.FormatInt(exp, 10)))
	m.Write([]byte{0})
	m.Write([]byte(userID))
	m.Write([]byte{0})
	m.Write([]byte(path))
	return m.Sum(nil)
}

// signLink returns the token for userID and path: "<exp>.<user id>.<signature>".
func signLink(key []byte, userID, path string, exp time.Time) string {
	sig := linkSignature(key, userID, path, exp.Unix())
	return strconv.FormatInt(exp.Unix(), 10) + "." + userID + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// verifyLink checks a token against the request path and the clock and returns the user id.
func verifyLink(key []byte, token, path string, now time.Time) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || !now.Before(time.Unix(exp, 0)) {
		return "", false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", false
	}
	if subtle.ConstantTimeCompare(sig, linkSignature(key, parts[1], path, exp)) != 1 {
		return "", false
	}
	return parts[1], true
}

// DownloadLink lets a GET request without a session authenticate with ?dl=<token> from
// POST /api/v1/files/download-link. Wrap the routes named in downloadable with it; a token
// is bound to one path, so a link for one file never opens another.
func DownloadLink(deps httpx.Deps) func(http.Handler) http.Handler {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := auth.UserFrom(r.Context()); ok || r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			token := r.URL.Query().Get("dl")
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			userID, ok := verifyLink(linkKey(deps), token, r.URL.Path, now())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid_link", "download link is invalid or expired")
				return
			}
			u, err := auth.UserByID(r.Context(), deps.Pool, userID)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "invalid_link", "download link is invalid or expired")
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), u)))
		})
	}
}

// downloadLink implements POST /api/v1/files/download-link.
func (h *handler) downloadLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL string `json:"url"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if !downloadable.MatchString(in.URL) {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "url must be a file route of this API without query")
		return
	}
	user, _ := auth.UserFrom(r.Context())
	exp := h.deps.Now().Add(DownloadLinkTTL)
	token := signLink(linkKey(h.deps), user.ID, in.URL, exp)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"url": in.URL + "?dl=" + token, "expires_at": exp.UTC()})
}
