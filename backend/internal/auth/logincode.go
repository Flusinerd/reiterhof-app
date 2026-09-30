package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

const (
	loginCodeDigits = 6
	// maxCodeAttempts is the number of wrong codes a login attempt tolerates; the fifth wrong
	// guess invalidates the code (the link of the same attempt stays usable).
	maxCodeAttempts = 5
)

// newLoginCode returns a uniformly distributed 6-digit code, leading zeros allowed.
func newLoginCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000)) // rejection sampling inside, no modulo bias
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", loginCodeDigits, n.Int64()), nil
}

// formatLoginCode groups the code for display in the mail: "123 456".
func formatLoginCode(code string) string { return code[:3] + " " + code[3:] }

// normalizeLoginCode accepts what people paste ("123 456", "123-456", " 123456 ") and
// returns the bare 6 digits.
func normalizeLoginCode(in string) (string, bool) {
	var b strings.Builder
	for _, r := range in {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == ' ' || r == ' ':
		default:
			return "", false
		}
	}
	if b.Len() != loginCodeDigits {
		return "", false
	}
	return b.String(), true
}

// hashLoginCode is HMAC-SHA256(key, email || 0x00 || code). A 6-digit code has only 10^6
// values, so a plain or merely salted hash falls to an offline search in milliseconds if the
// table leaks; the server-side key (which is not in the database) prevents that. Binding the
// email makes a hash useless for another address.
func hashLoginCode(key []byte, email, code string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(email))
	m.Write([]byte{0})
	m.Write([]byte(code))
	return m.Sum(nil)
}

// loginCodeKey returns the configured HMAC key or a random per-process one.
func loginCodeKey(configured string, log *slog.Logger) []byte {
	if configured != "" {
		return []byte(configured)
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("auth: no randomness for the login code key: " + err.Error())
	}
	if log == nil {
		log = slog.Default()
	}
	log.Warn("REITERHOF_LOGIN_CODE_KEY not set: using a random key, login codes do not survive a restart")
	return key
}

// verifyCode implements POST /api/v1/auth/verify-code. Every failure looks the same
// (401 invalid_code): unknown email, no open attempt, wrong code, locked out, expired.
func (s *Service) verifyCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	now := s.deps.Now()
	if !s.limiter.allow("vc-ip:"+clientIP(r), 30, 15*time.Minute, now) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again later")
		return
	}
	invalid := func() {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_code", "code is invalid, expired or already used")
	}
	email, okEmail := NormalizeEmail(in.Email)
	code, okCode := normalizeLoginCode(in.Code)
	if !okEmail || !okCode {
		invalid()
		return
	}

	tx, err := s.deps.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, "begin verify-code", err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck // rollback after commit is a no-op

	// At most one row per email holds a usable code (a new request clears older ones). FOR
	// UPDATE serialises parallel guesses so the attempt counter cannot be raced.
	var (
		id     string
		stored []byte
	)
	err = tx.QueryRow(r.Context(),
		`SELECT id, code_hash FROM login_tokens
		 WHERE email = $1 AND code_hash IS NOT NULL AND used_at IS NULL AND expires_at > $2
		 ORDER BY created_at DESC LIMIT 1 FOR UPDATE`, email, now).Scan(&id, &stored)
	found := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.internal(w, "load login code", err)
		return
	}
	if !found {
		stored = make([]byte, sha256.Size) // same work for unknown emails
	}
	match := subtle.ConstantTimeCompare(stored, hashLoginCode(s.codeKey, email, code)) == 1
	if !found || !match {
		if found {
			// Count the miss; the last allowed one also revokes the code. Commit even though
			// the request fails.
			if _, err := tx.Exec(r.Context(),
				`UPDATE login_tokens SET code_attempts = code_attempts + 1,
				        code_hash = CASE WHEN code_attempts + 1 >= $2 THEN NULL ELSE code_hash END
				 WHERE id = $1`, id, maxCodeAttempts); err != nil {
				s.internal(w, "count wrong code", err)
				return
			}
			if err := tx.Commit(r.Context()); err != nil {
				s.internal(w, "commit wrong code", err)
				return
			}
		}
		invalid()
		return
	}
	// Single use: this consumes the link of the same attempt as well.
	if _, err := tx.Exec(r.Context(), `UPDATE login_tokens SET used_at = $2 WHERE id = $1`, id, now); err != nil {
		s.internal(w, "consume login code", err)
		return
	}
	userID, err := s.userIDForEmail(r.Context(), tx, email, "")
	if err != nil {
		s.internal(w, "find user", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internal(w, "commit verify-code", err)
		return
	}
	s.issueSession(w, r, userID)
}
