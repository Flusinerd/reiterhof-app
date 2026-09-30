package push

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Flusinerd/reiterhof-app/backend/internal/dbtest"
)

type fixture struct {
	pool             *pgxpool.Pool
	stableA, stableB string
	anna, ben, cleo  string // anna+ben in A, cleo in B
}

func setup(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t)
	ctx := context.Background()
	f := fixture{pool: pool}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(pool.QueryRow(ctx, `INSERT INTO stables (name) VALUES ('A') RETURNING id`).Scan(&f.stableA))
	must(pool.QueryRow(ctx, `INSERT INTO stables (name) VALUES ('B') RETURNING id`).Scan(&f.stableB))
	user := func(stable, name string) string {
		var id string
		must(pool.QueryRow(ctx, `INSERT INTO users (stable_id, name, email) VALUES ($1, $2, $2 || '@example.org') RETURNING id`,
			stable, name).Scan(&id))
		return id
	}
	f.anna, f.ben, f.cleo = user(f.stableA, "anna"), user(f.stableA, "ben"), user(f.stableB, "cleo")
	return f
}

func tokensOf(t *testing.T, f fixture) map[string]string {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `SELECT token, user_id FROM push_tokens`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var tok, uid string
		if err := rows.Scan(&tok, &uid); err != nil {
			t.Fatal(err)
		}
		out[tok] = uid
	}
	return out
}

func TestRegisterAndDeleteToken(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	if err := RegisterToken(ctx, f.pool, f.stableA, f.anna, "tok1", PlatformAndroid); err != nil {
		t.Fatal(err)
	}
	// Upsert: same token, other platform and user.
	if err := RegisterToken(ctx, f.pool, f.stableA, f.ben, "tok1", PlatformIOS); err != nil {
		t.Fatal(err)
	}
	got := tokensOf(t, f)
	if len(got) != 1 || got["tok1"] != f.ben {
		t.Fatalf("tokens = %v, want tok1 -> ben", got)
	}
	var platform string
	if err := f.pool.QueryRow(ctx, `SELECT platform FROM push_tokens WHERE token='tok1'`).Scan(&platform); err != nil || platform != "ios" {
		t.Errorf("platform = %q, %v", platform, err)
	}

	// A user of another stable cannot be registered under this stable.
	if err := RegisterToken(ctx, f.pool, f.stableA, f.cleo, "tok2", PlatformIOS); !errors.Is(err, ErrUnknownUser) {
		t.Errorf("cross-stable err = %v, want ErrUnknownUser", err)
	}
	if err := RegisterToken(ctx, f.pool, f.stableA, f.anna, "tok3", "windows"); err == nil {
		t.Error("invalid platform accepted")
	}

	// Delete is scoped by stable and user.
	if ok, _ := DeleteToken(ctx, f.pool, f.stableB, f.ben, "tok1"); ok {
		t.Error("deleted across stables")
	}
	if ok, _ := DeleteToken(ctx, f.pool, f.stableA, f.anna, "tok1"); ok {
		t.Error("deleted another user's token")
	}
	if ok, err := DeleteToken(ctx, f.pool, f.stableA, f.ben, "tok1"); err != nil || !ok {
		t.Errorf("delete = %v, %v", ok, err)
	}
}

func TestNotifyUsers(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	for _, r := range []struct{ stable, user, token string }{
		{f.stableA, f.anna, "anna-1"}, {f.stableA, f.anna, "anna-2"},
		{f.stableA, f.ben, "ben-1"}, {f.stableB, f.cleo, "cleo-1"},
	} {
		if err := RegisterToken(ctx, f.pool, r.stable, r.user, r.token, PlatformAndroid); err != nil {
			t.Fatal(err)
		}
	}
	// Ben disabled medication; anna has an explicit enabled row (still sent).
	for _, s := range []struct {
		user    string
		enabled bool
	}{{f.ben, false}, {f.anna, true}} {
		if _, err := f.pool.Exec(ctx,
			`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1,$2,$3,$4)`,
			f.stableA, s.user, KindMedication, s.enabled); err != nil {
			t.Fatal(err)
		}
	}

	fake := &Fake{}
	n := NewNotifier(f.pool, fake, nil)

	// cleo belongs to stable B: asking for her within stable A yields nothing.
	err := n.NotifyUsers(ctx, f.stableA, []string{f.anna, f.ben, f.cleo}, KindMedication, "Titel", "Text", map[string]any{"horseId": "h1"})
	if err != nil {
		t.Fatal(err)
	}
	sent := fake.Sent()
	if len(sent) != 2 {
		t.Fatalf("sent %d messages, want 2 (anna's tokens only): %+v", len(sent), sent)
	}
	for _, m := range sent {
		if m.To != "anna-1" && m.To != "anna-2" {
			t.Errorf("unexpected recipient %s", m.To)
		}
		if m.Title != "Titel" || m.Body != "Text" || m.Data["horseId"] != "h1" || m.Data["kind"] != KindMedication {
			t.Errorf("message = %+v", m)
		}
	}

	// A different kind is not disabled for ben.
	fake.Reset()
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.ben}, KindHelper, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.Sent(); len(got) != 1 || got[0].To != "ben-1" {
		t.Errorf("sent = %+v", got)
	}

	// Opt-in kind: nothing without an enabling row, then only for the user who enabled it.
	fake.Reset()
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna, f.ben}, KindNewRequest, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.Sent(); len(got) != 0 {
		t.Errorf("opt-in kind sent without opt-in: %+v", got)
	}
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1,$2,$3,true), ($1,$4,$3,false)`,
		f.stableA, f.ben, KindNewRequest, f.anna); err != nil {
		t.Fatal(err)
	}
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna, f.ben}, KindNewRequest, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if got := fake.Sent(); len(got) != 1 || got[0].To != "ben-1" {
		t.Errorf("opt-in sent = %+v", got)
	}

	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, "bogus", "t", "b", nil); err == nil {
		t.Error("unknown kind accepted")
	}
	if err := n.NotifyUsers(ctx, f.stableA, nil, KindNewRequest, "t", "b", nil); err != nil {
		t.Errorf("no users: %v", err)
	}
}

func TestNotifyUsersDeletesInvalidTokens(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	for _, tok := range []string{"good", "dead"} {
		if err := RegisterToken(ctx, f.pool, f.stableA, f.anna, tok, PlatformIOS); err != nil {
			t.Fatal(err)
		}
	}
	// Same token string must never be deleted in another stable: register a
	// different token there to check it survives.
	if err := RegisterToken(ctx, f.pool, f.stableB, f.cleo, "other", PlatformIOS); err != nil {
		t.Fatal(err)
	}

	fake := &Fake{Invalid: map[string]bool{"dead": true}}
	n := NewNotifier(f.pool, fake, nil)
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err != nil {
		t.Fatalf("invalid tokens must not be an error: %v", err)
	}
	got := tokensOf(t, f)
	if _, ok := got["dead"]; ok || len(got) != 2 {
		t.Errorf("tokens after cleanup = %v", got)
	}

	// Other failures are returned, tokens stay.
	fake.Err = errors.New("boom")
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err == nil {
		t.Error("want error from sender")
	}
}
