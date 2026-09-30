package admincli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/admincli"
	"github.com/Flusinerd/reiterhof-app/backend/internal/auth"
	"github.com/Flusinerd/reiterhof-app/backend/internal/config"
	"github.com/Flusinerd/reiterhof-app/backend/migrations"
)

// clock is fixed so that expiry dates and retention are deterministic.
var clock = time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC)

// harness runs commands against a pool with captured output.
type harness struct {
	t    *testing.T
	env  *admincli.Env
	out  bytes.Buffer
	errw bytes.Buffer
	in   *strings.Reader
}

// newHarness builds an Env around pool (nil for commands that need no database).
func newHarness(t *testing.T, pool *pgxpool.Pool) *harness {
	t.Helper()
	h := &harness{t: t, in: strings.NewReader("")}
	h.env = &admincli.Env{
		In: h.in, Out: &h.out, Err: &h.errw, Pool: pool,
		Now:        func() time.Time { return clock },
		Migrations: migrations.FS,
	}
	return h
}

// run executes one command line and returns stdout; the buffers are reset per call.
func (h *harness) run(args ...string) (string, error) {
	h.t.Helper()
	h.out.Reset()
	h.errw.Reset()
	// Flags are global state of the Env; reset them so that calls do not leak into each other.
	h.env.JSON, h.env.Yes = false, false
	err := admincli.Run(context.Background(), h.env, args)
	return h.out.String(), err
}

// mustRun fails the test on error.
func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	out, err := h.run(args...)
	if err != nil {
		h.t.Fatalf("stallfunk-admin %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, out, h.errw.String())
	}
	return out
}

// answer sets what the next confirmation prompt reads from stdin.
func (h *harness) answer(s string) {
	h.env.In = strings.NewReader(s)
}

func TestHelpNeedsNoDatabase(t *testing.T) {
	h := newHarness(t, nil)
	for _, args := range [][]string{nil, {"help"}} {
		out := h.mustRun(args...)
		for _, want := range []string{"stable create", "user delete", "invite create", "horse transfer", "sessions revoke", "mail test", "weather refresh", "migrate status", "migrate up"} {
			if !strings.Contains(out, want) {
				t.Errorf("help %v misses %q", args, want)
			}
		}
	}
}

func TestUsageErrors(t *testing.T) {
	h := newHarness(t, nil) // no pool: all of these must fail before touching the database
	for name, args := range map[string][]string{
		"unknown command":        {"frobnicate"},
		"missing subcommand":     {"user"},
		"flag as subcommand":     {"user", "--json"},
		"unknown subcommand":     {"user", "explode"},
		"unknown flag":           {"user", "list", "--nope"},
		"stray argument":         {"user", "list", "extra"},
		"promote needs a user":   {"user", "promote"},
		"stable without name":    {"stable", "create", "--city", "Dorsten"},
		"lat without lng":        {"stable", "create", "--name", "X", "--lat", "51.6"},
		"lat out of range":       {"stable", "create", "--name", "X", "--lat", "91", "--lng", "7"},
		"lng out of range":       {"stable", "create", "--name", "X", "--lat", "51", "--lng", "181"},
		"bad timezone":           {"stable", "create", "--name", "X", "--timezone", "Mars/Base"},
		"bad reminder time":      {"stable", "create", "--name", "X", "--reminder-time", "25:00"},
		"reminder without zero":  {"stable", "create", "--name", "X", "--reminder-time", "8:30"},
		"auto uncover bad time":  {"stable", "create", "--name", "X", "--auto-uncover", "12:60"},
		"auto uncover too early": {"stable", "create", "--name", "X", "--auto-uncover", "03:59"},
		"auto uncover too late":  {"stable", "create", "--name", "X", "--auto-uncover", "15:00"},
		"auto uncover days bad":  {"stable", "create", "--name", "X", "--auto-uncover-days", "mon,xyz"},
		"auto uncover days rev":  {"stable", "update", "00000000-0000-4000-8000-000000000101", "--auto-uncover-days", "fri-mon"},
		"auto uncover days none": {"stable", "update", "00000000-0000-4000-8000-000000000101", "--auto-uncover-days", ""},
		"bad radius":             {"stable", "create", "--name", "X", "--geofence-radius", "0"},
		"update needs id":        {"stable", "update", "--name", "Y"},
		"update needs a change":  {"stable", "update", "00000000-0000-4000-8000-000000000101"},
		"user create no fields":  {"user", "create"},
		"user create bad email":  {"user", "create", "--email", "not-an-address", "--name", "X"},
		"user create name only":  {"user", "create", "--name", "X"},
		"invite days":            {"invite", "create", "--days", "0"},
		"invite uses":            {"invite", "create", "--max-uses", "101"},
		"transfer without --to":  {"horse", "transfer", "Luna"},
		"mail test without --to": {"mail", "test"},
		"mail test bad address":  {"mail", "test", "--to", "nope"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.run(args...)
			if !errors.Is(err, admincli.ErrUsage) {
				t.Errorf("err = %v, want a usage error", err)
			}
		})
	}
}

func TestFlagsMayFollowArguments(t *testing.T) {
	// "user demote a@b --yes" and "--yes a@b" are equivalent; this fails before the
	// database lookup only if the flag was parsed, not taken as a second positional.
	h := newHarness(t, nil)
	_, err := h.run("user", "promote", "a@example.org", "--json")
	if errors.Is(err, admincli.ErrUsage) {
		t.Errorf("trailing --json was rejected: %v", err)
	}
}

func TestHelpFlagIsNotAnError(t *testing.T) {
	h := newHarness(t, nil)
	if _, err := h.run("user", "list", "-h"); err != nil {
		t.Errorf("-h: %v", err)
	}
	if !strings.Contains(h.errw.String(), "-stable") {
		t.Errorf("-h should print the flags, got %q", h.errw.String())
	}
}

type fakeMailer struct {
	sent []auth.Message
	err  error
}

func (f *fakeMailer) Send(_ context.Context, m auth.Message) error {
	f.sent = append(f.sent, m)
	return f.err
}

func TestMailTest(t *testing.T) {
	h := newHarness(t, nil)
	h.env.Config = config.Config{Auth: config.Auth{SMTPHost: "smtp.example.org", SMTPPort: 587, SMTPFrom: "Stallfunk <login@example.org>"}}
	fm := &fakeMailer{}
	h.env.Mailer = fm

	out := h.mustRun("mail", "test", "--to", "Ops@Example.org")
	if len(fm.sent) != 1 || fm.sent[0].To != "ops@example.org" || !strings.Contains(fm.sent[0].Subject, "test mail") {
		t.Fatalf("sent = %+v", fm.sent)
	}
	if !strings.Contains(out, "ops@example.org") || !strings.Contains(out, "smtp.example.org:587") {
		t.Errorf("output = %q", out)
	}

	fm.err = errors.New("smtp: auth: 535 bad credentials")
	_, err := h.run("mail", "test", "--to", "ops@example.org")
	if err == nil || !strings.Contains(err.Error(), "535 bad credentials") || !strings.Contains(err.Error(), "smtp.example.org:587") {
		t.Errorf("err = %v, want the SMTP error and the server", err)
	}
}

func TestMailTestNeedsSMTPSettings(t *testing.T) {
	h := newHarness(t, nil)
	_, err := h.run("mail", "test", "--to", "ops@example.org")
	if err == nil || !strings.Contains(err.Error(), "REITERHOF_SMTP_HOST") {
		t.Errorf("without host: %v", err)
	}
	h.env.Config.Auth.SMTPHost = "smtp.example.org"
	_, err = h.run("mail", "test", "--to", "ops@example.org")
	if err == nil || !strings.Contains(err.Error(), "REITERHOF_SMTP_FROM") {
		t.Errorf("without from: %v", err)
	}
}
