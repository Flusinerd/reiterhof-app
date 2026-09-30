package push

import (
	"context"
	"errors"
	"testing"
)

func TestClientRoutesByPlatformAndSkipsUnconfigured(t *testing.T) {
	// Only FCM is configured: Android messages are delivered, iOS ones are skipped
	// (neither failures nor invalid tokens), unknown platforms are failures.
	fcm := &Fake{}
	c := &Client{FCM: fcm}
	err := c.Send(context.Background(), []Message{
		{To: "a1", Platform: PlatformAndroid, Body: "x"},
		{To: "i1", Platform: PlatformIOS, Body: "y"},
		{To: "w1", Platform: "web", Body: "z"},
	})
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v, want *SendError", err)
	}
	if len(serr.Failures) != 1 || len(serr.InvalidTokens) != 0 {
		t.Errorf("SendError = %+v, want one failure for the unknown platform", serr)
	}
	if got := fcm.Sent(); len(got) != 1 || got[0].To != "a1" {
		t.Errorf("fcm got %+v", got)
	}
}

func TestClientMergesProblemsOfBothServices(t *testing.T) {
	apns := &Fake{Invalid: map[string]bool{"i-bad": true}}
	fcm := &Fake{Err: errors.New("fcm down")}
	c := &Client{APNS: apns, FCM: fcm}
	err := c.Send(context.Background(), []Message{
		{To: "i-ok", Platform: PlatformIOS}, {To: "i-bad", Platform: PlatformIOS}, {To: "a1", Platform: PlatformAndroid},
	})
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v", err)
	}
	if len(serr.InvalidTokens) != 1 || serr.InvalidTokens[0] != "i-bad" {
		t.Errorf("InvalidTokens = %v", serr.InvalidTokens)
	}
	if len(serr.Failures) != 1 || serr.Failures[0] != "fcm down" {
		t.Errorf("Failures = %v", serr.Failures)
	}
	if got := apns.Sent(); len(got) != 1 || got[0].To != "i-ok" {
		t.Errorf("apns sent %v", got)
	}
}

func TestClientEmptyIsNil(t *testing.T) {
	if err := (&Client{}).Send(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if got := (&Client{}).Configured(); len(got) != 0 {
		t.Errorf("Configured = %v", got)
	}
}

func TestValidToken(t *testing.T) {
	for _, ok := range []string{"0af1", "dLx-3:APA91b_Q", "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789"} {
		if !ValidToken(ok) {
			t.Errorf("ValidToken(%q) = false", ok)
		}
	}
	for _, bad := range []string{"", "ExponentPushToken[abc]", "ExpoPushToken[abc]", "with space", "tab\there", "ünïcode", "[x]"} {
		if ValidToken(bad) {
			t.Errorf("ValidToken(%q) = true", bad)
		}
	}
}

func TestFakeSender(t *testing.T) {
	f := &Fake{Invalid: map[string]bool{"bad": true}}
	err := f.Send(context.Background(), []Message{{To: "ok", Body: "x"}, {To: "bad", Body: "y"}})
	var serr *SendError
	if !errors.As(err, &serr) || len(serr.InvalidTokens) != 1 {
		t.Fatalf("err = %v", err)
	}
	if got := f.Sent(); len(got) != 1 || got[0].To != "ok" {
		t.Errorf("Sent = %v", got)
	}
	f.Reset()
	if len(f.Sent()) != 0 {
		t.Error("Reset did not clear")
	}
}

func TestValidKind(t *testing.T) {
	if len(Kinds()) != 10 {
		t.Errorf("kinds = %d", len(Kinds()))
	}
	if !ValidKind(KindNewRequest) || !ValidKind(KindObservation) || ValidKind("nope") {
		t.Error("ValidKind wrong")
	}
}
