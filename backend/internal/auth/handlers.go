package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

const (
	loginTokenTTL = 15 * time.Minute
	appScheme     = "stallfunk"

	inviteAlphabet    = "ABCDEFGHJKMNPQRSTUVWXYZ23456789" // no 0/O, 1/I/L
	inviteCodeLen     = 8
	defaultInviteDays = 7
	defaultInviteUses = 10
)

var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{20,128}$`)

// Options customises a Service. All fields are optional; tests inject fakes.
type Options struct {
	Mailer Mailer
	Google *Verifier
	Apple  *Verifier
}

// Service holds the auth endpoints' state.
type Service struct {
	deps    httpx.Deps
	mailer  Mailer
	google  *Verifier
	apple   *Verifier
	limiter *limiter
}

// NewService builds the service; missing options default from deps.Config.Auth.
func NewService(deps httpx.Deps, opts Options) *Service {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	s := &Service{deps: deps, mailer: opts.Mailer, google: opts.Google, apple: opts.Apple, limiter: newLimiter()}
	cfg := deps.Config.Auth
	if s.mailer == nil {
		s.mailer = NewMailer(cfg, deps.Log)
	}
	if s.google == nil {
		s.google = NewGoogleVerifier(cfg.GoogleClientIDs)
	}
	if s.apple == nil {
		s.apple = NewAppleVerifier(cfg.AppleClientIDs)
	}
	return s
}

// Register mounts the auth, /me and stable-membership routes with production defaults.
func Register(mux *http.ServeMux, deps httpx.Deps) {
	NewService(deps, Options{}).Register(mux)
}

// Register mounts the routes on mux.
func (s *Service) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/magic-link", s.magicLink)
	mux.HandleFunc("POST /api/v1/auth/verify", s.verify)
	mux.HandleFunc("POST /api/v1/auth/google", s.social("google"))
	mux.HandleFunc("POST /api/v1/auth/apple", s.social("apple"))
	mux.HandleFunc("POST /api/v1/auth/dev-login", s.devLogin)
	mux.Handle("POST /api/v1/auth/logout", RequireUser(http.HandlerFunc(s.logout)))
	mux.Handle("GET /api/v1/me", RequireUser(http.HandlerFunc(s.getMe)))
	mux.Handle("PATCH /api/v1/me", RequireUser(http.HandlerFunc(s.patchMe)))
	mux.Handle("POST /api/v1/stables/join", RequireUser(http.HandlerFunc(s.join)))
	mux.Handle("POST /api/v1/stables/invites", RequireAdmin(http.HandlerFunc(s.createInvite)))
	// https fallback of the magic link for devices without the app; see verifyPage.
	mux.HandleFunc("GET /auth/verify", s.verifyPage)
}

func (s *Service) internal(w http.ResponseWriter, what string, err error) {
	if s.deps.Log != nil {
		s.deps.Log.Error("auth: "+what, "err", err)
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

// clientIP returns the caller's IP. Behind the reverse proxy on loopback (Caddy) the
// forwarded header is used, otherwise only the socket address (the header is spoofable).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	return host
}

// normalizeEmail validates a plain address (no display name) and lower-cases it.
func normalizeEmail(in string) (string, bool) {
	in = strings.TrimSpace(in)
	if in == "" || len(in) > 254 {
		return "", false
	}
	a, err := mail.ParseAddress(in)
	if err != nil || a.Address != in || a.Name != "" {
		return "", false
	}
	return strings.ToLower(in), true
}

// --- magic link ---------------------------------------------------------------

func (s *Service) magicLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	email, ok := normalizeEmail(in.Email)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "invalid email address")
		return
	}
	now := s.deps.Now()
	// Limits apply whether or not the address is known, so they reveal nothing.
	if !s.limiter.allow("ml-email:"+email, 3, 15*time.Minute, now) || !s.limiter.allow("ml-ip:"+clientIP(r), 20, time.Hour, now) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again later")
		return
	}
	token, hash, err := newToken()
	if err != nil {
		s.internal(w, "generate token", err)
		return
	}
	// Housekeeping: drop long-dead tokens and sessions.
	_, _ = s.deps.Pool.Exec(r.Context(), `DELETE FROM login_tokens WHERE expires_at < $1`, now.Add(-24*time.Hour))
	_, _ = s.deps.Pool.Exec(r.Context(), `DELETE FROM auth_sessions WHERE expires_at < $1`, now)
	if _, err := s.deps.Pool.Exec(r.Context(),
		`INSERT INTO login_tokens (token_hash, email, expires_at) VALUES ($1, $2, $3)`,
		hash, email, now.Add(loginTokenTTL)); err != nil {
		s.internal(w, "store login token", err)
		return
	}
	appURL := appScheme + "://auth/verify?token=" + url.QueryEscape(token)
	body := "Hallo!\n\nTippe auf diesen Link, um dich bei Stallfunk anzumelden:\n\n" + appURL + "\n"
	if base := s.deps.Config.Auth.PublicURL; base != "" {
		body += "\nFalls sich die App nicht öffnet, nutze diesen Link auf dem Gerät mit der App:\n\n" +
			base + "/auth/verify?token=" + url.QueryEscape(token) + "\n"
	}
	body += "\nDer Link ist 15 Minuten gültig und nur einmal verwendbar. Wenn du dich nicht anmelden wolltest, ignoriere diese E-Mail.\n"
	if err := s.mailer.Send(r.Context(), Message{To: email, Subject: "Dein Anmeldelink für Stallfunk", Body: body}); err != nil {
		// Still 204: the response must not depend on delivery or on the address.
		if s.deps.Log != nil {
			s.deps.Log.Error("auth: send login mail", "err", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	now := s.deps.Now()
	var email string
	// Single statement: marks the token used only if it is still valid, so it works once.
	err := s.deps.Pool.QueryRow(r.Context(),
		`UPDATE login_tokens SET used_at = $2 WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2 RETURNING email`,
		hashToken(in.Token), now).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "link is invalid, expired or already used")
		return
	}
	if err != nil {
		s.internal(w, "consume login token", err)
		return
	}
	userID, err := s.userIDForEmail(r.Context(), s.deps.Pool, email, "")
	if err != nil {
		s.internal(w, "find user", err)
		return
	}
	s.issueSession(w, r, userID)
}

// verifyPage is the https fallback for the magic link. It never consumes the token (mail
// scanners prefetch links); it only offers the app link.
var verifyTmpl = template.Must(template.New("verify").Parse(`<!doctype html>
<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Stallfunk</title></head><body>
<h1>Stallfunk</h1>
<p><a href="{{.}}">In der App anmelden</a></p>
<p>Öffne diesen Link auf dem Gerät, auf dem die Stallfunk-App installiert ist.</p>
</body></html>`))

func (s *Service) verifyPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if !tokenPattern.MatchString(token) {
		http.Error(w, "invalid link", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// The token matches [A-Za-z0-9_-]+ so the URL is safe to mark as trusted.
	_ = verifyTmpl.Execute(w, template.URL(appScheme+"://auth/verify?token="+token))
}

// --- social sign-in -----------------------------------------------------------

func (s *Service) social(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			IDToken string `json:"id_token"`
			Name    string `json:"name"` // Apple only sends the name to the client on first sign-in
		}
		if !httpx.ReadJSON(w, r, &in) {
			return
		}
		v := s.google
		if provider == "apple" {
			v = s.apple
		}
		if len(v.Audiences) == 0 {
			httpx.WriteError(w, http.StatusServiceUnavailable, "not_configured", "sign in with "+provider+" is not configured")
			return
		}
		claims, err := v.Verify(r.Context(), in.IDToken)
		if errors.Is(err, ErrInvalidToken) {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "id token rejected")
			return
		}
		if err != nil {
			s.internal(w, "verify id token", err)
			return
		}
		if provider == "google" {
			in.Name = claims.Name
		}
		userID, err := s.signInWithIdentity(r.Context(), provider, claims, strings.TrimSpace(in.Name))
		if errors.Is(err, errNoVerifiedEmail) {
			httpx.WriteError(w, http.StatusUnauthorized, "email_not_verified", "the account has no verified email address")
			return
		}
		if err != nil {
			s.internal(w, "sign in with identity", err)
			return
		}
		s.issueSession(w, r, userID)
	}
}

var errNoVerifiedEmail = errors.New("no verified email")

// signInWithIdentity resolves the local user for a provider identity. Identity is the
// provider subject. A new identity is attached to the user with the same verified email
// (account merge); Apple relay addresses are ordinary emails and therefore never match
// another account.
func (s *Service) signInWithIdentity(ctx context.Context, provider string, c Claims, name string) (string, error) {
	tx, err := s.deps.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	var userID string
	err = tx.QueryRow(ctx, `SELECT user_id FROM auth_identities WHERE provider = $1 AND subject = $2`, provider, c.Subject).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		if c.Email == "" || !c.EmailVerified {
			return "", errNoVerifiedEmail
		}
		if userID, err = s.userIDForEmail(ctx, tx, c.Email, name); err != nil {
			return "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO auth_identities (user_id, provider, subject, email) VALUES ($1, $2, $3, $4)`,
			userID, provider, c.Subject, c.Email)
	}
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

// userIDForEmail returns the user with that email, creating it (without stable) if needed.
// name is only used for a new user; empty falls back to the part before the "@".
func (s *Service) userIDForEmail(ctx context.Context, q DB, email, name string) (string, error) {
	if name == "" {
		name, _, _ = strings.Cut(email, "@")
	}
	var id string
	// The no-op update makes RETURNING work for the existing row; the unique index on
	// email makes concurrent sign-ups safe.
	err := q.QueryRow(ctx, `INSERT INTO users (name, email) VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email RETURNING id`, name, email).Scan(&id)
	return id, err
}

// --- session endpoints --------------------------------------------------------

func (s *Service) issueSession(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := CreateSession(r.Context(), s.deps.Pool, userID, r.UserAgent(), s.deps.Now())
	if err != nil {
		s.internal(w, "create session", err)
		return
	}
	view, err := s.loadMe(r.Context(), userID)
	if err != nil {
		s.internal(w, "load user", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"token": token, "user": view.User})
}

func (s *Service) devLogin(w http.ResponseWriter, r *http.Request) {
	if !s.deps.Config.Auth.DevLogin {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	email, ok := normalizeEmail(in.Email)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "invalid email address")
		return
	}
	userID, err := s.userIDForEmail(r.Context(), s.deps.Pool, email, "")
	if err != nil {
		s.internal(w, "find user", err)
		return
	}
	s.issueSession(w, r, userID)
}

func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	id, _ := r.Context().Value(sessionKey).(string)
	if _, err := s.deps.Pool.Exec(r.Context(), `DELETE FROM auth_sessions WHERE id = $1`, id); err != nil {
		s.internal(w, "delete session", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- /me ----------------------------------------------------------------------

type userView struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	Phone              *string `json:"phone"`
	AvatarColor        *string `json:"avatar_color"`
	PresenceVisibility string  `json:"presence_visibility"`
	IsAdmin            bool    `json:"is_admin"`
	StableID           *string `json:"stable_id"`
}

type stableView struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	FarmName *string `json:"farm_name"`
	City     *string `json:"city"`
	Timezone string  `json:"timezone"`
	// Location and geofence radius for the presence auto check-in (nil until the stable has coordinates).
	Lat             *float64 `json:"lat"`
	Lng             *float64 `json:"lng"`
	GeofenceRadiusM int      `json:"geofence_radius_m"`
}

type rolesView struct {
	OwnedHorseIDs []string `json:"owned_horse_ids"`
	RiderHorseIDs []string `json:"rider_horse_ids"`
}

type meView struct {
	User   userView    `json:"user"`
	Stable *stableView `json:"stable"`
	Roles  rolesView   `json:"roles"`
}

func (s *Service) loadMe(ctx context.Context, userID string) (meView, error) {
	var v meView
	err := s.deps.Pool.QueryRow(ctx, `SELECT id, name, email, phone, avatar_color, presence_visibility, is_admin, stable_id
		FROM users WHERE id = $1`, userID).
		Scan(&v.User.ID, &v.User.Name, &v.User.Email, &v.User.Phone, &v.User.AvatarColor,
			&v.User.PresenceVisibility, &v.User.IsAdmin, &v.User.StableID)
	if err != nil {
		return v, err
	}
	v.Roles = rolesView{OwnedHorseIDs: []string{}, RiderHorseIDs: []string{}}
	if v.User.StableID == nil {
		return v, nil
	}
	v.Stable = &stableView{}
	if err := s.deps.Pool.QueryRow(ctx, `SELECT id, name, farm_name, city, timezone, lat, lng, geofence_radius_m FROM stables WHERE id = $1`, *v.User.StableID).
		Scan(&v.Stable.ID, &v.Stable.Name, &v.Stable.FarmName, &v.Stable.City, &v.Stable.Timezone,
			&v.Stable.Lat, &v.Stable.Lng, &v.Stable.GeofenceRadiusM); err != nil {
		return v, err
	}
	if v.Roles.OwnedHorseIDs, err = s.idList(ctx, `SELECT id FROM horses WHERE owner_id = $1 AND stable_id = $2 ORDER BY name`, userID, *v.User.StableID); err != nil {
		return v, err
	}
	v.Roles.RiderHorseIDs, err = s.idList(ctx, `SELECT hr.horse_id FROM horse_riders hr JOIN horses h ON h.id = hr.horse_id
		WHERE hr.user_id = $1 AND hr.stable_id = $2 ORDER BY h.name`, userID, *v.User.StableID)
	return v, err
}

func (s *Service) idList(ctx context.Context, sql string, args ...any) ([]string, error) {
	rows, err := s.deps.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Service) getMe(w http.ResponseWriter, r *http.Request) {
	u, _ := UserFrom(r.Context())
	v, err := s.loadMe(r.Context(), u.ID)
	if err != nil {
		s.internal(w, "load me", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

var avatarColorPattern = regexp.MustCompile(`^[a-z_]{1,32}$`)

func (s *Service) patchMe(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name               *string `json:"name"`
		Phone              *string `json:"phone"`
		AvatarColor        *string `json:"avatar_color"`
		PresenceVisibility *string `json:"presence_visibility"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	invalid := func(msg string) { httpx.WriteError(w, http.StatusBadRequest, "validation_failed", msg) }
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" || len(n) > 100 {
			invalid("name must be 1 to 100 characters")
			return
		}
		in.Name = &n
	}
	if in.Phone != nil {
		p := strings.TrimSpace(*in.Phone)
		if len(p) > 32 {
			invalid("phone is too long")
			return
		}
		in.Phone = &p // "" clears the value
	}
	if in.AvatarColor != nil && *in.AvatarColor != "" && !avatarColorPattern.MatchString(*in.AvatarColor) {
		invalid("invalid avatar_color")
		return
	}
	if in.PresenceVisibility != nil {
		switch *in.PresenceVisibility {
		case "all", "only_day", "hidden":
		default:
			invalid("presence_visibility must be all, only_day or hidden")
			return
		}
	}
	u, _ := UserFrom(r.Context())
	// Phone and avatar_color: "" clears, absent leaves unchanged (COALESCE with a NULLIF flag).
	_, err := s.deps.Pool.Exec(r.Context(), `UPDATE users SET
		name = COALESCE($2, name),
		phone = CASE WHEN $3::boolean THEN NULLIF($4, '') ELSE phone END,
		avatar_color = CASE WHEN $5::boolean THEN NULLIF($6, '') ELSE avatar_color END,
		presence_visibility = COALESCE($7, presence_visibility)
		WHERE id = $1`,
		u.ID, in.Name, in.Phone != nil, in.Phone, in.AvatarColor != nil, in.AvatarColor, in.PresenceVisibility)
	if err != nil {
		s.internal(w, "update me", err)
		return
	}
	s.getMe(w, r)
}

// --- stables ------------------------------------------------------------------

// normalizeCode upper-cases and strips separators from a typed invite code.
func normalizeCode(in string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(in) {
		if r != ' ' && r != '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatCode renders ABCDEFGH as ABCD-EFGH for display.
func formatCode(code string) string {
	if len(code) == inviteCodeLen {
		return code[:4] + "-" + code[4:]
	}
	return code
}

func newInviteCode() (string, error) {
	var raw [inviteCodeLen]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	out := make([]byte, inviteCodeLen)
	for i, b := range raw {
		out[i] = inviteAlphabet[int(b)%len(inviteAlphabet)] // slight modulo bias is irrelevant here
	}
	return string(out), nil
}

func (s *Service) join(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code string `json:"code"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	u, _ := UserFrom(r.Context())
	now := s.deps.Now()
	if !s.limiter.allow("join:"+u.ID, 10, 15*time.Minute, now) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many attempts, try again later")
		return
	}
	if u.StableID != "" {
		httpx.WriteError(w, http.StatusConflict, "already_in_stable", "you already belong to a stable")
		return
	}
	code := normalizeCode(in.Code)
	tx, err := s.deps.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, "begin", err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck
	var stableID string
	err = tx.QueryRow(r.Context(), `UPDATE stable_invites SET uses = uses + 1
		WHERE code = $1 AND (expires_at IS NULL OR expires_at > $2) AND (max_uses IS NULL OR uses < max_uses)
		RETURNING stable_id`, code, now).Scan(&stableID)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "invalid_code", "invite code is invalid, expired or used up")
		return
	}
	if err != nil {
		s.internal(w, "redeem invite", err)
		return
	}
	// stable_id IS NULL guards against a concurrent join.
	tag, err := tx.Exec(r.Context(), `UPDATE users SET stable_id = $2 WHERE id = $1 AND stable_id IS NULL`, u.ID, stableID)
	if err != nil {
		s.internal(w, "join stable", err)
		return
	}
	if tag.RowsAffected() != 1 {
		httpx.WriteError(w, http.StatusConflict, "already_in_stable", "you already belong to a stable")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internal(w, "commit", err)
		return
	}
	s.getMe(w, r)
}

func (s *Service) createInvite(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ExpiresInDays *int `json:"expires_in_days"`
		MaxUses       *int `json:"max_uses"`
	}
	if r.ContentLength != 0 && !httpx.ReadJSON(w, r, &in) {
		return
	}
	days, uses := defaultInviteDays, defaultInviteUses
	if in.ExpiresInDays != nil {
		days = *in.ExpiresInDays
	}
	if in.MaxUses != nil {
		uses = *in.MaxUses
	}
	if days < 1 || days > 90 || uses < 1 || uses > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "expires_in_days must be 1..90 and max_uses 1..100")
		return
	}
	u, _ := UserFrom(r.Context())
	expires := s.deps.Now().Add(time.Duration(days) * 24 * time.Hour)
	for range 5 { // retry on the (very unlikely) code collision
		code, err := newInviteCode()
		if err != nil {
			s.internal(w, "generate code", err)
			return
		}
		tag, err := s.deps.Pool.Exec(r.Context(), `INSERT INTO stable_invites (stable_id, code, created_by, expires_at, max_uses)
			VALUES ($1, $2, $3, $4, $5) ON CONFLICT (code) DO NOTHING`, u.StableID, code, u.ID, expires, uses)
		if err != nil {
			s.internal(w, "store invite", err)
			return
		}
		if tag.RowsAffected() == 1 {
			httpx.WriteJSON(w, http.StatusCreated, map[string]any{"code": formatCode(code), "expires_at": expires, "max_uses": uses})
			return
		}
	}
	s.internal(w, "store invite", fmt.Errorf("could not find a free code"))
}
