package push

import (
	"context"
	"slices"
	"testing"
)

func TestNotifyUsersRequiresPushConsent(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	for _, r := range []struct{ user, token string }{{f.anna, "anna-1"}, {f.ben, "ben-1"}} {
		if err := RegisterToken(ctx, f.pool, f.stableA, r.user, r.token, PlatformAndroid); err != nil {
			t.Fatal(err)
		}
	}
	fake := &Fake{}
	n := NewNotifier(f.pool, fake, nil)
	send := func() []string {
		t.Helper()
		fake.Reset()
		if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna, f.ben}, KindHelper, "t", "b", nil); err != nil {
			t.Fatal(err)
		}
		var to []string
		for _, m := range fake.Sent() {
			to = append(to, m.To)
		}
		slices.Sort(to)
		return to
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := f.pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}

	if got := send(); !slices.Equal(got, []string{"anna-1", "ben-1"}) {
		t.Fatalf("with consent: sent to %v", got)
	}
	// Revoked consent: nothing for ben, anna is unaffected.
	exec(`UPDATE consents SET revoked_at = now() WHERE user_id = $1 AND kind = 'push'`, f.ben)
	if got := send(); !slices.Equal(got, []string{"anna-1"}) {
		t.Errorf("revoked consent: sent to %v, want only anna-1", got)
	}
	// No consent row at all: nothing either.
	exec(`DELETE FROM consents WHERE user_id = $1 AND kind = 'push'`, f.anna)
	if got := send(); len(got) != 0 {
		t.Errorf("absent consent: sent to %v, want nobody", got)
	}
	// Other consent kinds do not count.
	exec(`INSERT INTO consents (user_id, stable_id, kind, version, granted_at) VALUES ($1, $2, 'photos', 'x', now())`, f.anna, f.stableA)
	if got := send(); len(got) != 0 {
		t.Errorf("photos consent: sent to %v, want nobody", got)
	}
	// A new grant brings the user back.
	exec(`UPDATE consents SET revoked_at = NULL WHERE user_id = $1 AND kind = 'push'`, f.ben)
	if got := send(); !slices.Equal(got, []string{"ben-1"}) {
		t.Errorf("re-granted consent: sent to %v", got)
	}
}
