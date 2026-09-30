package auth

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Flusinerd/reiterhof-app/backend/internal/httpx"
)

// Age confirmation and parental consent (Art. 8 GDPR).
//
// Location, photos, maps, push and presence sharing rest on consent, which a person under
// 16 cannot give alone in Germany. So every account states once, right after the name:
//
//   - "16 oder älter": POST /api/v1/me/age {"over_16": true} sets users.age_confirmed_at.
//   - "jünger als 16": POST /api/v1/me/parental-consent {"parent_email"} stores the address
//     and mails the parent a link (<PublicURL>/parental-consent?token=..., 7 days, once).
//     The parent reads what the app does and confirms with one button; that sets
//     users.parental_consent_at. Until then the person cannot join a stable and cannot grant
//     consents (AgeConfirmed); the app shows a waiting screen.
//
// No birth date is stored, only the statement and the parent's address as proof. Nobody but
// the person and the parent is involved: no admin has to vouch, which keeps the stable
// self-administered.

const (
	parentalTokenTTL     = 7 * 24 * time.Hour
	parentalMailsPerDay  = 3
	parentalConfirmLimit = 20 // POST /parental-consent per IP and 15 minutes
)

// Values of userView.AgeStatus.
const (
	AgeStatusUnknown       = "unknown"
	AgeStatusConfirmed     = "confirmed"
	AgeStatusParentPending = "parent_pending"
)

func ageStatusOf(ageConfirmedAt, parentalConsentAt *time.Time, parentEmail *string) string {
	switch {
	case ageConfirmedAt != nil || parentalConsentAt != nil:
		return AgeStatusConfirmed
	case parentEmail != nil:
		return AgeStatusParentPending
	default:
		return AgeStatusUnknown
	}
}

// confirmAge implements POST /api/v1/me/age: the person states to be 16 or older.
func (s *Service) confirmAge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Over16 *bool `json:"over_16"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	if in.Over16 == nil || !*in.Over16 {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "over_16 must be true; a younger person asks a parent through /me/parental-consent")
		return
	}
	u, _ := UserFrom(r.Context())
	// A pending parent request is withdrawn: the person now states to be old enough.
	if _, err := s.deps.Pool.Exec(r.Context(), `UPDATE users SET age_confirmed_at = $2, parent_email = NULL WHERE id = $1`,
		u.ID, s.deps.Now()); err != nil {
		s.internal(w, "confirm age", err)
		return
	}
	s.getMe(w, r)
}

// requestParentalConsent implements POST /api/v1/me/parental-consent: the person is under
// 16 and names a parent, who gets the confirmation mail.
func (s *Service) requestParentalConsent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ParentEmail string `json:"parent_email"`
	}
	if !httpx.ReadJSON(w, r, &in) {
		return
	}
	parent, ok := NormalizeEmail(in.ParentEmail)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "invalid parent email address")
		return
	}
	u, _ := UserFrom(r.Context())
	now := s.deps.Now()
	if !s.limiter.allow("pc-user:"+u.ID, parentalMailsPerDay, 24*time.Hour, now) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again tomorrow")
		return
	}
	var (
		name, email string
		consentedAt *time.Time
	)
	if err := s.deps.Pool.QueryRow(r.Context(), `SELECT name, email, parental_consent_at FROM users WHERE id = $1`, u.ID).
		Scan(&name, &email, &consentedAt); err != nil {
		s.internal(w, "load user", err)
		return
	}
	if parent == email {
		httpx.WriteError(w, http.StatusBadRequest, "validation_failed", "the parent's address must differ from your own")
		return
	}
	if consentedAt != nil {
		httpx.WriteError(w, http.StatusConflict, "already_confirmed", "a parent already consented")
		return
	}
	token, hash, err := newToken()
	if err != nil {
		s.internal(w, "generate token", err)
		return
	}
	tx, err := s.deps.Pool.Begin(r.Context())
	if err != nil {
		s.internal(w, "begin", err)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck // rollback after commit is a no-op
	// Under 16 by their own statement: a self-confirmation from before is withdrawn. Older
	// links of this person stop working; only the newest mail counts.
	if _, err := tx.Exec(r.Context(), `UPDATE users SET parent_email = $2, age_confirmed_at = NULL WHERE id = $1`, u.ID, parent); err != nil {
		s.internal(w, "store parent", err)
		return
	}
	if _, err := tx.Exec(r.Context(), `UPDATE parental_consent_tokens SET expires_at = $2 WHERE user_id = $1 AND used_at IS NULL AND expires_at > $2`, u.ID, now); err != nil {
		s.internal(w, "revoke tokens", err)
		return
	}
	if _, err := tx.Exec(r.Context(), `INSERT INTO parental_consent_tokens (user_id, token_hash, parent_email, expires_at) VALUES ($1, $2, $3, $4)`,
		u.ID, hash, parent, now.Add(parentalTokenTTL)); err != nil {
		s.internal(w, "store token", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.internal(w, "commit", err)
		return
	}
	msg, err := parentalConsentMail(name, email, s.parentalLink(token), s.privacyURL()).Message(parent, "Zustimmung für "+name+" bei Stallfunk")
	if err != nil {
		s.internal(w, "render parental mail", err)
		return
	}
	if err := s.mailer.Send(r.Context(), msg); err != nil {
		if s.deps.Log != nil {
			s.deps.Log.Error("auth: send parental consent mail", "err", err)
		}
		httpx.WriteError(w, http.StatusBadGateway, "mail_failed", "the mail could not be sent, try again later")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parentalLink is the confirmation page for a token; relative when no public URL is set
// (development, where the mail is only logged).
func (s *Service) parentalLink(token string) string {
	return s.deps.Config.Auth.PublicURL + "/parental-consent?token=" + token
}

// privacyURL is the privacy text of the web app, or "" when the web app has no known address.
func (s *Service) privacyURL() string {
	if s.deps.Config.Auth.WebURL == "" {
		return ""
	}
	return s.deps.Config.Auth.WebURL + "/legal/privacy"
}

func parentalConsentMail(childName, childEmail, link, privacyURL string) MailContent {
	c := MailContent{
		Preheader: "Zustimmung für die Nutzung von Stallfunk",
		Heading:   "Zustimmung für " + childName,
		Intro: []string{
			childName + " (" + childEmail + ") möchte die App Stallfunk nutzen und hat angegeben, jünger als 16 Jahre zu sein. " +
				"Dafür brauchen wir die Zustimmung eines Elternteils oder einer erziehungsberechtigten Person.",
			"Stallfunk ist eine private App für die Stallgasse: Anwesenheit im Stall, Decken und Wetter, Anfragen an Helfer, " +
				"Pferdeakte und Training. Gespeichert werden Name und E-Mail-Adresse, Anwesenheiten im Stall und Einträge zu den Pferden. " +
				"Standort beim Reiten, Fotos, Karten und Mitteilungen gibt es nur nach einer Erlaubnis in der App, die " + childName + " mit deiner Zustimmung selbst erteilen darf.",
		},
		Button:    "Zustimmung geben",
		ButtonURL: link,
		Note: "Der Link gilt 7 Tage und nur einmal. Ohne Zustimmung kann " + childName + " die App nicht nutzen. " +
			"Nicht angefordert? Dann ignoriere diese Mail.",
	}
	if privacyURL != "" {
		c.Outro = []string{"Wie die App mit Daten umgeht, steht in der Datenschutzerklärung: " + privacyURL}
	}
	return c
}

// --- the parent's page ------------------------------------------------------------------

var parentalStyle = verifyStyle + `
ul{margin:0;padding-left:20px;color:#6b6560}
li{margin:4px 0}
form{margin:0}
.button{border:0;width:100%;font-size:16px;font-family:inherit;cursor:pointer}
a{color:#2d5a3d}
`

var (
	parentalStyleHash = base64Sum(parentalStyle)
	parentalCSP       = "default-src 'none'; style-src 'sha256-" + parentalStyleHash + "'; img-src data:; form-action 'self'; frame-ancestors 'none'; base-uri 'none'"
)

var parentalTmpl = template.Must(template.New("parental").Parse(`<!doctype html>
<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Stallfunk</title>
<style>{{.Style}}</style></head><body>
<main>
<div class="brand"><img src="{{.Logo}}" alt="" width="44" height="44"><span>Stallfunk</span></div>
{{if .Error}}<h1>Link ungültig</h1>
<p>{{.Error}}</p>
{{else if .Done}}<h1>Danke</h1>
<p>{{.Child}} darf Stallfunk jetzt nutzen. Du kannst dieses Fenster schließen.</p>
<p>Deine Zustimmung kannst du jederzeit widerrufen: Schreibe an die im Impressum der App genannte Adresse, dann wird das Konto gelöscht.</p>
{{else}}<h1>Zustimmung für {{.Child}}</h1>
<p>{{.Child}} ({{.Email}}) möchte die App Stallfunk nutzen und hat angegeben, jünger als 16 Jahre zu sein. Als Elternteil oder erziehungsberechtigte Person kannst du hier zustimmen.</p>
<p>Stallfunk ist eine private App für die Stallgasse. Sie speichert:</p>
<ul>
<li>Name und E-Mail-Adresse, Zugehörigkeit zum Stall</li>
<li>Anwesenheit im Stall und Einträge zu den Pferden (Pferdeakte, Decken, Anfragen, Training)</li>
<li>nur nach einer Erlaubnis in der App, die {{.Child}} mit deiner Zustimmung selbst erteilen darf: Standort beim Reiten, Fotos, Karten und Mitteilungen</li>
</ul>
<form method="post" action="/parental-consent"><input type="hidden" name="token" value="{{.Token}}"><button class="button" type="submit">Ich stimme zu</button></form>
{{if .PrivacyURL}}<small>Alle Einzelheiten stehen in der <a href="{{.PrivacyURL}}">Datenschutzerklärung</a>.</small>{{end}}
<small>Der Link gilt 7 Tage und nur einmal. Wenn du nicht zustimmst, schließe die Seite; {{.Child}} kann die App dann nicht nutzen.</small>
{{end}}
</main>
</body></html>`))

type parentalPage struct {
	Style      template.CSS
	Logo       template.URL
	Child      string
	Email      string
	Token      string
	PrivacyURL string
	Done       bool
	Error      string
}

func (s *Service) renderParental(w http.ResponseWriter, status int, p parentalPage) {
	p.Style = template.CSS(parentalStyle)
	p.Logo = verifyLogoURL
	p.PrivacyURL = s.privacyURL()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", parentalCSP)
	w.WriteHeader(status)
	_ = parentalTmpl.Execute(w, p)
}

const parentalInvalid = "Der Link ist abgelaufen, wurde schon benutzt oder ist unvollständig. Bitte lass dir in der App einen neuen schicken."

// parentalConsentPage implements GET /parental-consent?token=: shows whom the consent is for
// and what the app does; the token is only consumed by the POST (mail scanners open links).
func (s *Service) parentalConsentPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if !tokenPattern.MatchString(token) {
		s.renderParental(w, http.StatusBadRequest, parentalPage{Error: parentalInvalid})
		return
	}
	var name, email string
	err := s.deps.Pool.QueryRow(r.Context(), `SELECT u.name, u.email FROM parental_consent_tokens t JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1 AND t.used_at IS NULL AND t.expires_at > $2 AND u.deleted_at IS NULL`, hashToken(token), s.deps.Now()).
		Scan(&name, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderParental(w, http.StatusBadRequest, parentalPage{Error: parentalInvalid})
		return
	}
	if err != nil {
		s.internal(w, "load parental token", err)
		return
	}
	s.renderParental(w, http.StatusOK, parentalPage{Child: name, Email: email, Token: token})
}

// parentalConsentConfirm implements POST /parental-consent (form field token): the parent's
// click. The token works once; the person's account is marked as consented.
func (s *Service) parentalConsentConfirm(w http.ResponseWriter, r *http.Request) {
	now := s.deps.Now()
	if !s.limiter.allow("pc-ip:"+clientIP(r), parentalConfirmLimit, 15*time.Minute, now) {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again later")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		s.renderParental(w, http.StatusBadRequest, parentalPage{Error: parentalInvalid})
		return
	}
	token := strings.TrimSpace(r.PostForm.Get("token"))
	if !tokenPattern.MatchString(token) {
		s.renderParental(w, http.StatusBadRequest, parentalPage{Error: parentalInvalid})
		return
	}
	name, err := s.consumeParentalToken(r.Context(), token, now)
	if errors.Is(err, pgx.ErrNoRows) {
		s.renderParental(w, http.StatusBadRequest, parentalPage{Error: parentalInvalid})
		return
	}
	if err != nil {
		s.internal(w, "confirm parental consent", err)
		return
	}
	s.renderParental(w, http.StatusOK, parentalPage{Child: name, Done: true})
}

// consumeParentalToken marks the token used and the user as consented; pgx.ErrNoRows when
// the token is unknown, used or expired.
func (s *Service) consumeParentalToken(ctx context.Context, token string, now time.Time) (string, error) {
	tx, err := s.deps.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	var userID, parent string
	if err := tx.QueryRow(ctx, `UPDATE parental_consent_tokens SET used_at = $2
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2 RETURNING user_id::text, parent_email`,
		hashToken(token), now).Scan(&userID, &parent); err != nil {
		return "", err
	}
	var name string
	if err := tx.QueryRow(ctx, `UPDATE users SET parental_consent_at = $2, parent_email = $3
		WHERE id = $1 AND deleted_at IS NULL RETURNING name`, userID, now, parent).Scan(&name); err != nil {
		return "", err
	}
	return name, tx.Commit(ctx)
}
