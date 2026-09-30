package config

import (
	"os"
	"strconv"
	"strings"
)

// Auth holds the authentication settings (see docs/architecture.md, "Authentication").
type Auth struct {
	// PublicURL is the public base URL of the API (REITERHOF_PUBLIC_URL). It is used for the
	// https fallback of the magic link: <PublicURL>/auth/verify?token=... and for the
	// parental consent link.
	PublicURL string
	// WebURL is the public base URL of the web app (REITERHOF_WEB_URL, e.g. https://stallfunk.de).
	// Pages and mails the API renders link to its privacy text (<WebURL>/legal/privacy).
	WebURL string
	// DevLogin enables POST /api/v1/auth/dev-login (REITERHOF_DEV_LOGIN=true). Never in production.
	DevLogin bool
	// GoogleClientIDs are the accepted "aud" values of Google ID tokens
	// (REITERHOF_GOOGLE_CLIENT_IDS, comma separated: web, iOS and Android client IDs).
	GoogleClientIDs []string
	// AppleClientIDs are the accepted "aud" values of Apple ID tokens
	// (REITERHOF_APPLE_CLIENT_IDS, comma separated; the iOS bundle ID for native sign-in).
	AppleClientIDs []string
	// LoginCodeKey is the secret key (REITERHOF_LOGIN_CODE_KEY) for the HMAC that protects the
	// stored 6-digit login codes. Without it a random key is generated per process start, so
	// codes requested before a restart stop working (the links keep working).
	LoginCodeKey string
	// SMTP settings for the magic-link mail. With SMTPHost empty the link is only logged.
	SMTPHost     string // REITERHOF_SMTP_HOST
	SMTPPort     int    // REITERHOF_SMTP_PORT, default 587
	SMTPUser     string // REITERHOF_SMTP_USER
	SMTPPassword string // REITERHOF_SMTP_PASSWORD
	SMTPFrom     string // REITERHOF_SMTP_FROM, e.g. "Stallfunk <login@example.org>"
}

func authFromEnv() Auth {
	port, err := strconv.Atoi(getenv("REITERHOF_SMTP_PORT", "587"))
	if err != nil {
		port = 587
	}
	return Auth{
		PublicURL:       strings.TrimRight(os.Getenv("REITERHOF_PUBLIC_URL"), "/"),
		WebURL:          strings.TrimRight(os.Getenv("REITERHOF_WEB_URL"), "/"),
		DevLogin:        os.Getenv("REITERHOF_DEV_LOGIN") == "true",
		GoogleClientIDs: splitList(os.Getenv("REITERHOF_GOOGLE_CLIENT_IDS")),
		AppleClientIDs:  splitList(os.Getenv("REITERHOF_APPLE_CLIENT_IDS")),
		LoginCodeKey:    os.Getenv("REITERHOF_LOGIN_CODE_KEY"),
		SMTPHost:        os.Getenv("REITERHOF_SMTP_HOST"),
		SMTPPort:        port,
		SMTPUser:        os.Getenv("REITERHOF_SMTP_USER"),
		SMTPPassword:    os.Getenv("REITERHOF_SMTP_PASSWORD"),
		SMTPFrom:        os.Getenv("REITERHOF_SMTP_FROM"),
	}
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
