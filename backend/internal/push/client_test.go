package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type fakeExpo struct {
	mu      sync.Mutex
	batches [][]Message
	auth    []string
	// handler decides the response for a batch; default: all ok.
	handler func(w http.ResponseWriter, batch []Message, call int)
}

func (f *fakeExpo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var batch []Message
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	f.mu.Lock()
	call := len(f.batches)
	f.batches = append(f.batches, batch)
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	f.mu.Unlock()
	if f.handler != nil {
		f.handler(w, batch, call)
		return
	}
	writeTickets(w, batch, nil)
}

func writeTickets(w http.ResponseWriter, batch []Message, errs map[int]string) {
	tickets := make([]map[string]any, len(batch))
	for i := range batch {
		if code, ok := errs[i]; ok {
			tickets[i] = map[string]any{
				"status": "error", "message": "problem " + code,
				"details": map[string]any{"error": code},
			}
		} else {
			tickets[i] = map[string]any{"status": "ok", "id": fmt.Sprintf("t%d", i)}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": tickets})
}

func msgs(n int) []Message {
	out := make([]Message, n)
	for i := range out {
		out[i] = Message{To: fmt.Sprintf("ExponentPushToken[%03d]", i), Body: "hi"}
	}
	return out
}

func TestClientBatchesOf100(t *testing.T) {
	f := &fakeExpo{}
	srv := httptest.NewServer(f)
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, AccessToken: "secret"}

	if err := c.Send(context.Background(), msgs(250)); err != nil {
		t.Fatal(err)
	}
	if len(f.batches) != 3 || len(f.batches[0]) != 100 || len(f.batches[1]) != 100 || len(f.batches[2]) != 50 {
		t.Fatalf("batches = %d, want sizes 100/100/50", len(f.batches))
	}
	if f.batches[2][0].To != "ExponentPushToken[200]" {
		t.Errorf("order not preserved: %s", f.batches[2][0].To)
	}
	for _, a := range f.auth {
		if a != "Bearer secret" {
			t.Errorf("Authorization = %q", a)
		}
	}
}

func TestClientNoAuthHeaderWithoutToken(t *testing.T) {
	f := &fakeExpo{}
	srv := httptest.NewServer(f)
	defer srv.Close()
	if err := (&Client{BaseURL: srv.URL}).Send(context.Background(), msgs(1)); err != nil {
		t.Fatal(err)
	}
	if f.auth[0] != "" {
		t.Errorf("Authorization = %q, want empty", f.auth[0])
	}
}

func TestClientEmptyDoesNotCallExpo(t *testing.T) {
	f := &fakeExpo{}
	srv := httptest.NewServer(f)
	defer srv.Close()
	if err := (&Client{BaseURL: srv.URL}).Send(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if len(f.batches) != 0 {
		t.Errorf("unexpected request")
	}
}

func TestClientTicketErrors(t *testing.T) {
	f := &fakeExpo{handler: func(w http.ResponseWriter, batch []Message, _ int) {
		writeTickets(w, batch, map[int]string{1: "DeviceNotRegistered", 2: "MessageTooBig"})
	}}
	srv := httptest.NewServer(f)
	defer srv.Close()

	var called []string
	c := &Client{BaseURL: srv.URL, OnInvalidToken: func(_ context.Context, tok string) { called = append(called, tok) }}
	err := c.Send(context.Background(), msgs(4))
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v, want *SendError", err)
	}
	if len(serr.InvalidTokens) != 1 || serr.InvalidTokens[0] != "ExponentPushToken[001]" {
		t.Errorf("InvalidTokens = %v", serr.InvalidTokens)
	}
	if len(called) != 1 || called[0] != "ExponentPushToken[001]" {
		t.Errorf("callback = %v", called)
	}
	if len(serr.Failures) != 1 {
		t.Errorf("Failures = %v, want 1 (MessageTooBig)", serr.Failures)
	}
}

func TestClientBatchFailureDoesNotStopOthers(t *testing.T) {
	f := &fakeExpo{handler: func(w http.ResponseWriter, batch []Message, call int) {
		if call == 0 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"errors":[{"code":"TOO_MANY_REQUESTS","message":"slow down"}]}`))
			return
		}
		writeTickets(w, batch, map[int]string{0: "DeviceNotRegistered"})
	}}
	srv := httptest.NewServer(f)
	defer srv.Close()

	err := (&Client{BaseURL: srv.URL}).Send(context.Background(), msgs(150))
	var serr *SendError
	if !errors.As(err, &serr) {
		t.Fatalf("err = %v", err)
	}
	if len(f.batches) != 2 {
		t.Fatalf("batches = %d, want 2", len(f.batches))
	}
	if len(serr.Failures) != 1 {
		t.Errorf("Failures = %v", serr.Failures)
	}
	if len(serr.InvalidTokens) != 1 || serr.InvalidTokens[0] != "ExponentPushToken[100]" {
		t.Errorf("InvalidTokens = %v", serr.InvalidTokens)
	}
}

func TestClientMalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	if err := (&Client{BaseURL: srv.URL}).Send(context.Background(), msgs(1)); err == nil {
		t.Fatal("want error")
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
	if len(Kinds()) != 9 {
		t.Errorf("kinds = %d", len(Kinds()))
	}
	if !ValidKind(KindNewRequest) || ValidKind("nope") {
		t.Error("ValidKind wrong")
	}
}
