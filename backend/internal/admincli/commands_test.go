package admincli_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/admincli"
	"github.com/Flusinerd/reiterhof-app/backend/internal/auth/authtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
	"github.com/Flusinerd/reiterhof-app/backend/internal/files"
	"github.com/Flusinerd/reiterhof-app/backend/internal/push"
	"github.com/Flusinerd/reiterhof-app/backend/internal/seed"
	"github.com/Flusinerd/reiterhof-app/backend/internal/weather"
)

func count(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return lines[len(lines)-1]
}

// TestBootstrapEmptyDatabase is the operator's first-day flow on a fresh server.
func TestBootstrapEmptyDatabase(t *testing.T) {
	pool := dbtest.NewUnmigrated(t)
	h := newHarness(t, pool)

	out := h.mustRun("migrate", "status")
	if !strings.Contains(out, "0 applied") || !strings.Contains(out, "pending") {
		t.Errorf("status before up:\n%s", out)
	}
	out = h.mustRun("migrate", "up")
	if !strings.Contains(out, "applied 0001_core.up.sql") {
		t.Errorf("migrate up:\n%s", out)
	}
	out = h.mustRun("migrate", "up")
	if !strings.Contains(out, "up to date") {
		t.Errorf("second migrate up:\n%s", out)
	}
	if out = h.mustRun("migrate", "status"); !strings.Contains(out, " 0 pending") {
		t.Errorf("status after up:\n%s", out)
	}

	// Without any stable, commands that need one explain what to do.
	_, err := h.run("user", "create", "--email", "jan@example.org", "--name", "Jan")
	if err == nil || !strings.Contains(err.Error(), "stable create") {
		t.Errorf("user create without stable: %v", err)
	}

	stableID := lastLine(h.mustRun("stable", "create", "--name", "Stallgasse B", "--farm", "Hof Ahlers", "--city", "Dorsten",
		"--lat", "51.66", "--lng", "6.96", "--timezone", "Europe/Berlin", "--reminder-time", "21:15", "--geofence-radius", "200"))
	var name, farm, tz, reminder string
	var radius int
	if err := pool.QueryRow(context.Background(), `SELECT name, farm_name, timezone, to_char(reminder_time, 'HH24:MI'), geofence_radius_m
		FROM stables WHERE id = $1`, stableID).Scan(&name, &farm, &tz, &reminder, &radius); err != nil {
		t.Fatal(err)
	}
	if name != "Stallgasse B" || farm != "Hof Ahlers" || tz != "Europe/Berlin" || reminder != "21:15" || radius != 200 {
		t.Errorf("stable row = %s %s %s %s %d", name, farm, tz, reminder, radius)
	}

	// One stable: --stable defaults to it. The address is lower-cased.
	userID := lastLine(h.mustRun("user", "create", "--email", "Jan.Krueger@Example.ORG", "--name", "Jan", "--admin"))
	var email string
	var admin bool
	var userStable string
	if err := pool.QueryRow(context.Background(), `SELECT email, is_admin, stable_id::text FROM users WHERE id = $1`, userID).Scan(&email, &admin, &userStable); err != nil {
		t.Fatal(err)
	}
	if email != "jan.krueger@example.org" || !admin || userStable != stableID {
		t.Errorf("user = %q admin=%v stable=%s", email, admin, userStable)
	}
	if _, err := h.run("user", "create", "--email", "JAN.KRUEGER@example.org", "--name", "Doppelt"); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("duplicate email: %v", err)
	}

	code := lastLine(h.mustRun("invite", "create"))
	if len(code) != 9 || code[4] != '-' {
		t.Errorf("invite code %q, want ABCD-EFGH", code)
	}
	var maxUses int
	var expires time.Time
	if err := pool.QueryRow(context.Background(), `SELECT max_uses, expires_at FROM stable_invites WHERE code = $1 AND stable_id = $2`,
		strings.ReplaceAll(code, "-", ""), stableID).Scan(&maxUses, &expires); err != nil {
		t.Fatal(err)
	}
	if maxUses != 10 || !expires.Equal(clock.Add(7*24*time.Hour)) {
		t.Errorf("invite defaults: max_uses %d, expires %s", maxUses, expires)
	}

	// A second stable turns the defaults into an explicit choice.
	other := lastLine(h.mustRun("stable", "create", "--name", "Zweiter Hof"))
	_, err = h.run("invite", "create")
	if err == nil || !strings.Contains(err.Error(), "--stable") || !strings.Contains(err.Error(), other) {
		t.Errorf("invite create with two stables: %v", err)
	}
	h.mustRun("invite", "create", "--stable", "Zweiter Hof", "--days", "1", "--max-uses", "1")
	if n := count(t, pool, `SELECT count(*) FROM stable_invites WHERE stable_id = $1`, other); n != 1 {
		t.Errorf("invites of the second stable = %d", n)
	}
}

func TestStableListAndUpdate(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)

	out := h.mustRun("stable", "list")
	for _, want := range []string{seed.StableB, "Stallgasse B", "Hof Ahlers", "Dorsten", "Europe/Berlin", "20:30"} {
		if !strings.Contains(out, want) {
			t.Errorf("stable list misses %q:\n%s", want, out)
		}
	}
	var list []struct {
		ID     string `json:"id"`
		Users  int    `json:"users"`
		Horses int    `json:"horses"`
	}
	if err := json.Unmarshal([]byte(h.mustRun("--json", "stable", "list")), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != seed.StableB || list[0].Users != 8 || list[0].Horses != 7 {
		t.Errorf("json list = %+v", list)
	}

	h.mustRun("stable", "update", seed.StableB, "--city", "Marl", "--reminder-time", "19:00", "--lat", "51.7", "--farm", "")
	var city, reminder string
	var farm *string
	var lat, lng float64
	if err := pool.QueryRow(context.Background(), `SELECT city, farm_name, to_char(reminder_time, 'HH24:MI'), lat, lng FROM stables WHERE id = $1`, seed.StableB).
		Scan(&city, &farm, &reminder, &lat, &lng); err != nil {
		t.Fatal(err)
	}
	if city != "Marl" || farm != nil || reminder != "19:00" || lat != 51.7 || lng != 6.96 {
		t.Errorf("after update: city %q farm %v reminder %q lat %v lng %v (lng must be untouched)", city, farm, reminder, lat, lng)
	}
	if _, err := h.run("stable", "update", "00000000-0000-4000-8000-0000000000ff", "--city", "X"); err == nil {
		t.Error("update of an unknown stable must fail")
	}
}

func TestUserListAndCreateWithSeveralStables(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	exec(t, pool, `INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-0000000009a1', 'Anderer Stall')`)

	if _, err := h.run("user", "create", "--email", "neu@example.org", "--name", "Neu"); err == nil || !strings.Contains(err.Error(), "several stables") {
		t.Errorf("user create without --stable: %v", err)
	}
	h.mustRun("user", "create", "--email", "neu@example.org", "--name", "Neu", "--stable", "anderer stall")

	out := h.mustRun("user", "list")
	for _, want := range []string{"jan@example.org", "Stallgasse B", "Anderer Stall", "neu@example.org"} {
		if !strings.Contains(out, want) {
			t.Errorf("user list misses %q:\n%s", want, out)
		}
	}
	out = h.mustRun("user", "list", "--stable", "00000000-0000-4000-8000-0000000009a1")
	if !strings.Contains(out, "neu@example.org") || strings.Contains(out, "jan@example.org") {
		t.Errorf("filtered user list:\n%s", out)
	}
	var users []struct {
		Email   string `json:"email"`
		IsAdmin bool   `json:"is_admin"`
	}
	if err := json.Unmarshal([]byte(h.mustRun("user", "list", "--json", "--stable", seed.StableB)), &users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 8 {
		t.Errorf("json users = %d, want 8", len(users))
	}
	if _, err := h.run("user", "list", "--stable", "does not exist"); err == nil {
		t.Error("unknown stable must fail")
	}
}

func TestPromoteDemote(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	isAdmin := func(id string) bool {
		var b bool
		if err := pool.QueryRow(context.Background(), `SELECT is_admin FROM users WHERE id = $1`, id).Scan(&b); err != nil {
			t.Fatal(err)
		}
		return b
	}

	// Jan is the only admin of the seed stable.
	if _, err := h.run("user", "demote", "jan@example.org", "--yes"); err == nil || !strings.Contains(err.Error(), "last admin") {
		t.Errorf("demoting the last admin: %v", err)
	}
	if !isAdmin(seed.UserJan) {
		t.Fatal("Jan lost the admin flag")
	}

	h.mustRun("user", "promote", "ANNA@example.org")
	if !isAdmin(seed.UserAnna) {
		t.Error("Anna is not an admin")
	}
	if out := h.mustRun("user", "promote", seed.UserAnna); !strings.Contains(out, "already") {
		t.Errorf("second promote: %s", out)
	}

	// The prompt defaults to no.
	h.answer("n\n")
	if _, err := h.run("user", "demote", "jan@example.org"); !errors.Is(err, admincli.ErrAborted) {
		t.Errorf("declined demote: %v", err)
	}
	if !isAdmin(seed.UserJan) {
		t.Error("declined demote changed Jan")
	}
	h.answer("yes\n")
	h.mustRun("user", "demote", "jan@example.org")
	if isAdmin(seed.UserJan) {
		t.Error("Jan is still an admin")
	}
	// Now Anna is the last one.
	if _, err := h.run("user", "demote", seed.UserAnna, "--yes"); err == nil {
		t.Error("Anna is the last admin now")
	}

	// Users without a stable cannot be admins.
	exec(t, pool, `INSERT INTO users (id, name, email) VALUES ('00000000-0000-4000-8000-0000000009b1', 'Lose', 'lose@example.org')`)
	if _, err := h.run("user", "promote", "lose@example.org"); err == nil || !strings.Contains(err.Error(), "not in a stable") {
		t.Errorf("promote without stable: %v", err)
	}
	if _, err := h.run("user", "promote", "nobody@example.org"); err == nil || !strings.Contains(err.Error(), "no user") {
		t.Errorf("promote unknown: %v", err)
	}
}

func TestUserMove(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	const newStable = "00000000-0000-4000-8000-0000000009a1"
	exec(t, pool, `INSERT INTO stables (id, name) VALUES ($1, 'Anderer Stall')`, newStable)
	stableOf := func(id string) string {
		var s *string
		if err := pool.QueryRow(context.Background(), `SELECT stable_id::text FROM users WHERE id = $1`, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		if s == nil {
			return ""
		}
		return *s
	}

	// A user who never joined a stable is assigned without prompt.
	exec(t, pool, `INSERT INTO users (id, name, email) VALUES ('00000000-0000-4000-8000-0000000009b1', 'Lose', 'lose@example.org')`)
	h.mustRun("user", "move", "lose@example.org", "--stable", newStable)
	if stableOf("00000000-0000-4000-8000-0000000009b1") != newStable {
		t.Error("stable-less user was not assigned")
	}

	// Owners and last admins stay.
	if _, err := h.run("user", "move", "anna@example.org", "--stable", newStable, "--yes"); err == nil || !strings.Contains(err.Error(), "Fanta") || !strings.Contains(err.Error(), "Nala") {
		t.Errorf("moving an owner: %v", err)
	}
	if _, err := h.run("user", "move", seed.UserAnna, "--stable", newStable); !errors.Is(err, admincli.ErrAborted) {
		t.Errorf("no confirmation given: %v", err)
	}
	exec(t, pool, `UPDATE horses SET owner_id = $1 WHERE owner_id = $2`, seed.UserJan, seed.UserAnna)
	exec(t, pool, `UPDATE users SET is_admin = true WHERE id = $1`, seed.UserMia)
	if _, err := h.run("user", "move", "jan@example.org", "--stable", newStable, "--yes"); err == nil {
		t.Error("Jan owns horses")
	}

	// Mia (rider of Luna, admin) can go because Jan is admin too; her stable-scoped rows and the admin flag go.
	exec(t, pool, `INSERT INTO presence (stable_id, user_id, arrived_at) VALUES ($1, $2, $3)`, seed.StableB, seed.UserMia, clock)
	h.mustRun("user", "move", "mia@example.org", "--stable", "Anderer Stall", "--yes")
	if stableOf(seed.UserMia) != newStable {
		t.Error("Mia was not moved")
	}
	var admin bool
	if err := pool.QueryRow(context.Background(), `SELECT is_admin FROM users WHERE id = $1`, seed.UserMia).Scan(&admin); err != nil || admin {
		t.Errorf("admin flag after move = %v, %v", admin, err)
	}
	if n := count(t, pool, `SELECT count(*) FROM horse_riders WHERE user_id = $1`, seed.UserMia) +
		count(t, pool, `SELECT count(*) FROM presence WHERE user_id = $1`, seed.UserMia); n != 0 {
		t.Errorf("%d stable-scoped rows of Mia remain", n)
	}

	// Jan is the last admin of the seed stable now (after his horses are given away).
	exec(t, pool, `UPDATE horses SET owner_id = $1 WHERE owner_id = $2`, seed.UserTom, seed.UserJan)
	if _, err := h.run("user", "move", "jan@example.org", "--stable", newStable, "--yes"); err == nil || !strings.Contains(err.Error(), "last admin") {
		t.Errorf("moving the last admin: %v", err)
	}
	if out := h.mustRun("user", "move", "mia@example.org", "--stable", newStable); !strings.Contains(out, "already") {
		t.Errorf("no-op move: %s", out)
	}
}

func TestUserDelete(t *testing.T) {
	t.Cleanup(files.SetDir(t.TempDir()))
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)

	// Blocking case 1: owns horses. The error names them.
	_, err := h.run("user", "delete", "anna@example.org", "--yes")
	if err == nil || !strings.Contains(err.Error(), "owns_horses") || !strings.Contains(err.Error(), "Fanta") ||
		!strings.Contains(err.Error(), "Nala") || !strings.Contains(err.Error(), seed.HorseFanta) {
		t.Errorf("owner delete: %v", err)
	}
	if n := count(t, pool, `SELECT count(*) FROM users WHERE id = $1 AND deleted_at IS NULL`, seed.UserAnna); n != 1 {
		t.Error("Anna must be untouched")
	}

	// Blocking case 2: the only admin of a stable (no horses of their own).
	exec(t, pool, `INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-0000000009a1', 'Anderer Stall')`)
	exec(t, pool, `INSERT INTO users (id, stable_id, name, email, is_admin) VALUES ('00000000-0000-4000-8000-0000000009b2', '00000000-0000-4000-8000-0000000009a1', 'Solo', 'solo@example.org', true)`)
	if _, err := h.run("user", "delete", "solo@example.org", "--yes"); err == nil || !strings.Contains(err.Error(), "last_admin") {
		t.Errorf("last admin delete: %v", err)
	}

	// Declined prompt.
	h.answer("\n")
	if _, err := h.run("user", "delete", "mia@example.org"); !errors.Is(err, admincli.ErrAborted) {
		t.Errorf("declined delete: %v", err)
	}

	// Success: same semantics as the API (anonymised, sessions gone).
	authtest.TokenAt(t, pool, seed.UserMia, clock)
	if n := count(t, pool, `SELECT count(*) FROM auth_sessions WHERE user_id = $1`, seed.UserMia); n != 1 {
		t.Fatalf("sessions before delete = %d", n)
	}
	out := h.mustRun("user", "delete", seed.UserMia, "--yes")
	if !strings.Contains(out, "deleted account") {
		t.Errorf("output %q", out)
	}
	var name, email string
	var deleted *time.Time
	if err := pool.QueryRow(context.Background(), `SELECT name, email, deleted_at FROM users WHERE id = $1`, seed.UserMia).Scan(&name, &email, &deleted); err != nil {
		t.Fatal(err)
	}
	if name != "Gelöschtes Mitglied" || !strings.HasSuffix(email, "@deleted.invalid") || deleted == nil || !deleted.Equal(clock) {
		t.Errorf("after delete: %q %q %v", name, email, deleted)
	}
	if n := count(t, pool, `SELECT count(*) FROM auth_sessions WHERE user_id = $1`, seed.UserMia); n != 0 {
		t.Errorf("sessions after delete = %d", n)
	}
	if strings.Contains(h.mustRun("user", "list"), "mia@example.org") {
		t.Error("deleted user is still listed")
	}
	if _, err := h.run("user", "delete", "mia@example.org", "--yes"); err == nil || !strings.Contains(err.Error(), "no user") {
		t.Errorf("second delete: %v", err)
	}
}

func TestInviteList(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	active := lastLine(h.mustRun("invite", "create", "--days", "3", "--max-uses", "5"))
	exec(t, pool, `INSERT INTO stable_invites (stable_id, code, created_by, expires_at, max_uses, uses) VALUES
		($1, 'EXPIRED2', $2, $3, 5, 0), ($1, 'USEDUPP2', $2, NULL, 2, 2)`, seed.StableB, seed.UserJan, clock.Add(-time.Hour))

	out := h.mustRun("invite", "list")
	for _, want := range []string{active, "EXPI-RED2", "USED-UPP2", "Stallgasse B", "Jan", "0/5", "2/2", "expired", "used up", "active"} {
		if !strings.Contains(out, want) {
			t.Errorf("invite list misses %q:\n%s", want, out)
		}
	}
	var list []struct {
		Code   string `json:"code"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(h.mustRun("invite", "list", "--json", "--stable", seed.StableB)), &list); err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, i := range list {
		statuses[i.Code] = i.Status
	}
	if len(list) != 3 || statuses[active] != "active" || statuses["EXPI-RED2"] != "expired" || statuses["USED-UPP2"] != "used up" {
		t.Errorf("json invites = %+v", list)
	}
}

func TestHorseListAndTransfer(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	owner := func(horseID string) string {
		var o *string
		if err := pool.QueryRow(context.Background(), `SELECT owner_id::text FROM horses WHERE id = $1`, horseID).Scan(&o); err != nil {
			t.Fatal(err)
		}
		if o == nil {
			return ""
		}
		return *o
	}

	out := h.mustRun("horse", "list")
	for _, want := range []string{"Luna", "Jan", "Nala", "Anna", "Stallgasse B", seed.HorseLuna} {
		if !strings.Contains(out, want) {
			t.Errorf("horse list misses %q:\n%s", want, out)
		}
	}

	// By name, new owner by e-mail (any case). The prompt asks first.
	h.answer("n\n")
	if _, err := h.run("horse", "transfer", "luna", "--to", "ANNA@example.org"); !errors.Is(err, admincli.ErrAborted) {
		t.Errorf("declined transfer: %v", err)
	}
	if owner(seed.HorseLuna) != seed.UserJan {
		t.Fatal("declined transfer changed the owner")
	}
	h.mustRun("horse", "transfer", "luna", "--to", "ANNA@example.org", "--yes")
	if owner(seed.HorseLuna) != seed.UserAnna {
		t.Error("Luna did not move to Anna")
	}
	// By id, new owner by id, flag before the argument.
	h.mustRun("horse", "transfer", "--to", seed.UserTom, seed.HorseLuna, "--yes")
	if owner(seed.HorseLuna) != seed.UserTom {
		t.Error("Luna did not move to Tom")
	}
	if out := h.mustRun("horse", "transfer", "Luna", "--to", "tom@example.org"); !strings.Contains(out, "already") {
		t.Errorf("no-op transfer: %s", out)
	}

	// Only within one stable.
	exec(t, pool, `INSERT INTO stables (id, name) VALUES ('00000000-0000-4000-8000-0000000009a1', 'Anderer Stall')`)
	exec(t, pool, `INSERT INTO users (id, stable_id, name, email) VALUES ('00000000-0000-4000-8000-0000000009b3', '00000000-0000-4000-8000-0000000009a1', 'Fremd', 'fremd@example.org')`)
	exec(t, pool, `INSERT INTO users (id, name, email) VALUES ('00000000-0000-4000-8000-0000000009b4', 'Lose', 'lose@example.org')`)
	for _, to := range []string{"fremd@example.org", "lose@example.org"} {
		_, err := h.run("horse", "transfer", "Luna", "--to", to, "--yes")
		if err == nil || !strings.Contains(err.Error(), "within a stable") {
			t.Errorf("transfer to %s: %v", to, err)
		}
	}
	if owner(seed.HorseLuna) != seed.UserTom {
		t.Error("a refused transfer changed the owner")
	}

	// Ambiguous names must not guess.
	exec(t, pool, `INSERT INTO horses (id, stable_id, name) VALUES ('00000000-0000-4000-8000-0000000009c1', '00000000-0000-4000-8000-0000000009a1', 'Luna')`)
	_, err := h.run("horse", "transfer", "Luna", "--to", "tom@example.org", "--yes")
	if err == nil || !strings.Contains(err.Error(), "several horses") || !strings.Contains(err.Error(), "00000000-0000-4000-8000-0000000009c1") {
		t.Errorf("ambiguous name: %v", err)
	}
	h.mustRun("horse", "transfer", "Luna", "--stable", seed.StableB, "--to", "jan@example.org", "--yes")
	if owner(seed.HorseLuna) != seed.UserJan {
		t.Error("--stable did not narrow the name")
	}
	if _, err := h.run("horse", "transfer", "Nobody", "--to", "jan@example.org", "--yes"); err == nil {
		t.Error("unknown horse")
	}
}

func TestSessionsRevoke(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	h := newHarness(t, pool)
	authtest.TokenAt(t, pool, seed.UserJan, clock)
	authtest.TokenAt(t, pool, seed.UserJan, clock)
	authtest.TokenAt(t, pool, seed.UserAnna, clock)

	out := h.mustRun("sessions", "revoke", "JAN@example.org")
	if !strings.Contains(out, "revoked 2 session") {
		t.Errorf("output %q", out)
	}
	if n := count(t, pool, `SELECT count(*) FROM auth_sessions WHERE user_id = $1`, seed.UserJan); n != 0 {
		t.Errorf("Jan has %d sessions left", n)
	}
	if n := count(t, pool, `SELECT count(*) FROM auth_sessions WHERE user_id = $1`, seed.UserAnna); n != 1 {
		t.Errorf("Anna's session count = %d, want 1", n)
	}
	if out := h.mustRun("sessions", "revoke", seed.UserJan); !strings.Contains(out, "revoked 0") {
		t.Errorf("second revoke: %q", out)
	}
}

type fixtureFetcher struct {
	hours []weather.Hour
	err   error
	calls int
}

func (f *fixtureFetcher) Fetch(context.Context, weather.Station) ([]weather.Hour, error) {
	f.calls++
	return f.hours, f.err
}

func TestWeatherRefresh(t *testing.T) {
	pool := dbtest.NewSeeded(t)
	raw, err := os.ReadFile("../weather/testdata/MOSMIX_L_sample_10410.kml")
	if err != nil {
		t.Fatal(err)
	}
	hours, err := weather.ParseMOSMIX(raw)
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, pool)
	ff := &fixtureFetcher{hours: hours}
	h.env.Fetcher = ff
	h.env.Notify = push.NewNotifier(pool, &push.Fake{}, nil)
	h.env.Now = func() time.Time { return time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC) }

	out := h.mustRun("weather", "refresh")
	if !strings.Contains(out, "weather refreshed for 1 stable") || !strings.Contains(out, "weather-change check done") {
		t.Errorf("output %q", out)
	}
	if ff.calls != 1 {
		t.Errorf("fetches = %d, want 1", ff.calls)
	}
	if n := count(t, pool, `SELECT count(*) FROM weather_snapshots WHERE stable_id = $1`, seed.StableB); n == 0 {
		t.Error("no snapshot stored")
	}

	ff.err = errors.New("dwd down")
	if _, err := h.run("weather", "refresh"); err == nil || !strings.Contains(err.Error(), "dwd down") {
		t.Errorf("fetch failure: %v", err)
	}
}
