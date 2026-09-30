package mistral

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteJSON(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" || r.Method != http.MethodPost {
			t.Errorf("auth = %q, method = %s", r.Header.Get("Authorization"), r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"days\":[]}"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	c := &Client{APIKey: "secret", Model: DefaultModel, URL: srv.URL}
	text, err := c.CompleteJSON(context.Background(), "sys", "user")
	if err != nil || text != `{"days":[]}` {
		t.Fatalf("text = %q, err = %v", text, err)
	}
	if got["model"] != DefaultModel || got["response_format"].(map[string]any)["type"] != "json_object" {
		t.Errorf("request = %v", got)
	}
	msgs := got["messages"].([]any)
	if len(msgs) != 2 || msgs[0].(map[string]any)["role"] != "system" || msgs[1].(map[string]any)["content"] != "user" {
		t.Errorf("messages = %v", msgs)
	}
}

func TestCompleteJSONChunksAndErrors(t *testing.T) {
	answer, status := "", http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	defer srv.Close()
	c := &Client{APIKey: "k", Model: DefaultModel, URL: srv.URL}

	answer = `{"choices":[{"message":{"content":[{"type":"text","text":"{\"a\":"},{"type":"text","text":"1}"}]}}]}`
	if text, err := c.CompleteJSON(context.Background(), "s", "u"); err != nil || text != `{"a":1}` {
		t.Errorf("chunks: %q %v", text, err)
	}

	status, answer = http.StatusTooManyRequests, `{"message":"quota"}`
	if _, err := c.CompleteJSON(context.Background(), "s", "u"); !errors.Is(err, ErrLimit) {
		t.Errorf("429: %v", err)
	}
	status, answer = http.StatusUnauthorized, `{"message":"secret user text"}`
	if _, err := c.CompleteJSON(context.Background(), "s", "u"); err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("401 error must not echo the body: %v", err)
	}
	status, answer = http.StatusOK, `{"choices":[{"message":{"content":"{\"days\":["},"finish_reason":"length"}]}`
	if _, err := c.CompleteJSON(context.Background(), "s", "u"); err == nil {
		t.Error("cut off answer accepted")
	}
	status, answer = http.StatusOK, `{"choices":[]}`
	if _, err := c.CompleteJSON(context.Background(), "s", "u"); err == nil {
		t.Error("empty choices accepted")
	}
}

func TestFromEnvAndLabsModels(t *testing.T) {
	t.Setenv(EnvAPIKey, "")
	if _, err := FromEnv(); !errors.Is(err, ErrNotConfigured) {
		t.Errorf("no key: %v", err)
	}
	t.Setenv(EnvAPIKey, " key ")
	t.Setenv(EnvModel, "")
	c, err := FromEnv()
	if err != nil || c.APIKey != "key" || c.Model != DefaultModel {
		t.Fatalf("client = %+v, err = %v", c, err)
	}
	t.Setenv(EnvModel, "labs-mistral-small-creative")
	if _, err := FromEnv(); err == nil {
		t.Error("labs model accepted")
	}
	c = &Client{APIKey: "k", Model: "Labs-Devstral"}
	if _, err := c.CompleteJSON(context.Background(), "s", "u"); err == nil || !strings.Contains(err.Error(), "Labs") {
		t.Errorf("labs model called: %v", err)
	}
}
