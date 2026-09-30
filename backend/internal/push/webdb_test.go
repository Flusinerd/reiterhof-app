package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
)

// webSub returns a valid subscription for endpoint.
func webSub(t *testing.T, endpoint string) WebSubscription {
	t.Helper()
	s, err := ValidateWebSubscription(newReceiver(t).subscription(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func webEndpoints(msgs []WebMessage) []string {
	var out []string
	for _, m := range msgs {
		out = append(out, m.Sub.Endpoint)
	}
	slices.Sort(out)
	return out
}

func TestRegisterWebSubscription(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	sub := webSub(t, "https://push.example.com/a")

	if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.anna, sub, "UA/1"); err != nil {
		t.Fatal(err)
	}
	// Same endpoint again: upsert, re-assigned to another user of the stable.
	if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.ben, sub, "UA/2"); err != nil {
		t.Fatal(err)
	}
	var user, ua string
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*), max(user_id::text), max(user_agent) FROM web_push_subscriptions`).Scan(&n, &user, &ua); err != nil {
		t.Fatal(err)
	}
	if n != 1 || user != f.ben || ua != "UA/2" {
		t.Errorf("after upsert: n=%d user=%s ua=%s", n, user, ua)
	}
	// A user of another stable cannot register into this one.
	if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.cleo, webSub(t, "https://push.example.com/b"), ""); !errors.Is(err, ErrUnknownUser) {
		t.Errorf("foreign user: err = %v, want ErrUnknownUser", err)
	}
	// Delete is scoped to the owner and the stable.
	if ok, err := DeleteWebSubscription(ctx, f.pool, f.stableA, f.anna, sub.Endpoint); err != nil || ok {
		t.Errorf("delete by non-owner: ok=%v err=%v", ok, err)
	}
	if ok, err := DeleteWebSubscription(ctx, f.pool, f.stableA, f.ben, sub.Endpoint); err != nil || !ok {
		t.Errorf("delete by owner: ok=%v err=%v", ok, err)
	}
}

func TestRegisterWebSubscriptionCapsPerUser(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	for i := range MaxWebSubscriptionsPerUser + 3 {
		if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.anna, webSub(t, fmt.Sprintf("https://push.example.com/%02d", i)), ""); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := f.pool.Query(ctx, `SELECT endpoint FROM web_push_subscriptions WHERE user_id = $1`, f.anna)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var eps []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			t.Fatal(err)
		}
		eps = append(eps, e)
	}
	if len(eps) != MaxWebSubscriptionsPerUser {
		t.Fatalf("%d subscriptions kept, want %d", len(eps), MaxWebSubscriptionsPerUser)
	}
	if slices.Contains(eps, "https://push.example.com/00") || !slices.Contains(eps, fmt.Sprintf("https://push.example.com/%02d", MaxWebSubscriptionsPerUser+2)) {
		t.Errorf("oldest must go, newest stay: %v", eps)
	}
}

func TestNotifyUsersWebFilters(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	epAnna, epBen, epCleo := "https://push.example.com/anna", "https://push.example.com/ben", "https://push.example.com/cleo"
	for _, r := range []struct{ stable, user, ep string }{{f.stableA, f.anna, epAnna}, {f.stableA, f.ben, epBen}, {f.stableB, f.cleo, epCleo}} {
		if err := RegisterWebSubscription(ctx, f.pool, r.stable, r.user, webSub(t, r.ep), ""); err != nil {
			t.Fatal(err)
		}
	}
	fake, web := &Fake{}, &FakeWeb{}
	n := NewNotifier(f.pool, fake, nil).WithWeb(web)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := f.pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	send := func(kind string) []string {
		t.Helper()
		web.Reset()
		if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna, f.ben, f.cleo}, kind, "t", "b", nil); err != nil {
			t.Fatal(err)
		}
		return webEndpoints(web.Sent())
	}

	// Stable scoping: cleo is in stable B and not addressed within A.
	if got := send(KindHelper); !slices.Equal(got, []string{epAnna, epBen}) {
		t.Errorf("default: %v", got)
	}
	// Opt-out per kind.
	exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'helper', false)`, f.stableA, f.ben)
	if got := send(KindHelper); !slices.Equal(got, []string{epAnna}) {
		t.Errorf("opted out: %v", got)
	}
	if got := send(KindMedication); !slices.Equal(got, []string{epAnna, epBen}) {
		t.Errorf("other kind is unaffected: %v", got)
	}
	// Opt-in kind: nobody without a row, only the enabled user with one.
	if got := send(KindNewRequest); len(got) != 0 {
		t.Errorf("opt-in without row: %v", got)
	}
	exec(`INSERT INTO reminder_settings (stable_id, user_id, kind, enabled) VALUES ($1, $2, 'new_request', true)`, f.stableA, f.anna)
	if got := send(KindNewRequest); !slices.Equal(got, []string{epAnna}) {
		t.Errorf("opt-in with row: %v", got)
	}
	// Consent: revoked, absent, other kind.
	exec(`UPDATE consents SET revoked_at = now() WHERE user_id = $1 AND kind = 'push'`, f.anna)
	if got := send(KindHelper); len(got) != 0 {
		t.Errorf("revoked consent: %v", got)
	}
	exec(`DELETE FROM consents WHERE user_id = $1 AND kind = 'push'`, f.ben)
	if got := send(KindMedication); len(got) != 0 {
		t.Errorf("absent consent: %v", got)
	}
}

func TestNotifyUsersWebPayloadAndDelivery(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	ep := "https://push.example.com/anna"
	if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.anna, webSub(t, ep), ""); err != nil {
		t.Fatal(err)
	}
	if err := RegisterToken(ctx, f.pool, f.stableA, f.anna, "apns-a", PlatformIOS); err != nil {
		t.Fatal(err)
	}
	fake, web := &Fake{}, &FakeWeb{}
	n := NewNotifier(f.pool, fake, nil).WithWeb(web)
	data := map[string]any{"screen": "/requests/42", "request_id": "42"}
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindUrgentObservation, "Titel", "Text", data); err != nil {
		t.Fatal(err)
	}
	// Both paths deliver.
	if len(fake.Sent()) != 1 {
		t.Errorf("expo messages: %d", len(fake.Sent()))
	}
	sent := web.Sent()
	if len(sent) != 1 {
		t.Fatalf("web messages: %d", len(sent))
	}
	if sent[0].Urgency != "high" || sent[0].TTL <= 0 {
		t.Errorf("urgency %q ttl %v", sent[0].Urgency, sent[0].TTL)
	}
	var p struct {
		Title, Body string
		Data        map[string]any
	}
	if err := json.Unmarshal(sent[0].Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.Title != "Titel" || p.Body != "Text" || p.Data["screen"] != "/requests/42" || p.Data["kind"] != KindUrgentObservation || p.Data["request_id"] != "42" {
		t.Errorf("payload = %s", sent[0].Payload)
	}
	if _, ok := data["kind"]; ok {
		t.Error("the caller's data map was modified")
	}
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if got := web.Sent()[1].Urgency; got != "normal" {
		t.Errorf("helper urgency = %q", got)
	}
}

func TestNotifyUsersWebGoneAndErrors(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	gone, ok := "https://push.example.com/gone", "https://push.example.com/ok"
	for _, ep := range []string{gone, ok} {
		if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.anna, webSub(t, ep), ""); err != nil {
			t.Fatal(err)
		}
	}
	web := &FakeWeb{Gone: map[string]bool{gone: true}}
	n := NewNotifier(f.pool, &Fake{}, nil).WithWeb(web)
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err != nil {
		t.Fatalf("a gone subscription is not an error: %v", err)
	}
	var left []string
	rows, _ := f.pool.Query(ctx, `SELECT endpoint FROM web_push_subscriptions`)
	for rows.Next() {
		var e string
		_ = rows.Scan(&e)
		left = append(left, e)
	}
	rows.Close()
	if !slices.Equal(left, []string{ok}) {
		t.Errorf("subscriptions left: %v, want only %s", left, ok)
	}
	// Other failures are returned, and do not remove the subscription.
	web.Err = &SendError{Failures: []string{"http 503"}}
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err == nil {
		t.Error("a failing push service must be reported")
	}
	var count int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM web_push_subscriptions`).Scan(&count)
	if count != 1 {
		t.Errorf("subscription removed after a 5xx: %d left", count)
	}
}

func TestNotifyUsersWebNotConfigured(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if err := RegisterWebSubscription(ctx, f.pool, f.stableA, f.anna, webSub(t, "https://push.example.com/a"), ""); err != nil {
		t.Fatal(err)
	}
	if err := RegisterToken(ctx, f.pool, f.stableA, f.anna, "apns-a", PlatformIOS); err != nil {
		t.Fatal(err)
	}
	fake := &Fake{}
	n := NewNotifier(f.pool, fake, nil) // no WithWeb: web push is off
	if err := n.NotifyUsers(ctx, f.stableA, []string{f.anna}, KindHelper, "t", "b", nil); err != nil {
		t.Fatal(err)
	}
	if len(fake.Sent()) != 1 {
		t.Errorf("expo delivery must still work, sent %d", len(fake.Sent()))
	}
}
