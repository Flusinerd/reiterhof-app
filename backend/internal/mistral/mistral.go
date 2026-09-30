// Package mistral is a minimal client for the chat completions API of Mistral AI (Mistral
// AI SAS, Paris). The training week plan (JAN-89) uses it; see docs/domains/training.md.
//
// Data protection: API calls must not be used for training. That is a setting of the
// Mistral account (Admin Console, API, Privacy: "Allow the use of your API calls to train
// Mistral's AI models" off), not of the request. Labs models are trained on regardless of
// that setting, so the client refuses them.
package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Environment variables.
const (
	// EnvAPIKey is the API key from Mistral Studio (API Keys). Unset disables the client.
	EnvAPIKey = "REITERHOF_MISTRAL_API_KEY"
	// EnvModel overrides DefaultModel.
	EnvModel = "REITERHOF_MISTRAL_MODEL"
)

const (
	// DefaultURL is the chat completions endpoint.
	DefaultURL = "https://api.mistral.ai/v1/chat/completions"
	// DefaultModel is small, cheap and good enough for a JSON week plan.
	DefaultModel = "mistral-small-latest"
	// MaxTokens caps the answer; a week plan needs a few hundred.
	MaxTokens = 1500
)

// ErrNotConfigured is returned by FromEnv when no API key is set.
var ErrNotConfigured = errors.New("mistral: no API key (" + EnvAPIKey + ")")

// ErrLimit means the free credits or the rate limit are used up (HTTP 429).
var ErrLimit = errors.New("mistral: rate limit or credits exhausted")

// Client calls the chat completions API.
type Client struct {
	APIKey string
	Model  string
	// URL defaults to DefaultURL; tests point it at httptest.
	URL string
	// HTTPClient defaults to a client with a 30 s timeout.
	HTTPClient *http.Client
}

// FromEnv builds the client from REITERHOF_MISTRAL_*. Without an API key it returns
// ErrNotConfigured.
func FromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv(EnvAPIKey))
	if key == "" {
		return nil, ErrNotConfigured
	}
	c := &Client{APIKey: key, Model: strings.TrimSpace(os.Getenv(EnvModel))}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if err := checkModel(c.Model); err != nil {
		return nil, err
	}
	return c, nil
}

// checkModel refuses Labs models: Mistral may train on their inputs whatever the privacy
// setting of the account says.
func checkModel(model string) error {
	if strings.Contains(strings.ToLower(model), "labs") {
		return fmt.Errorf("mistral: %s is a Labs model, Mistral trains on its inputs; use a regular model", model)
	}
	return nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompleteJSON sends a system and a user message and returns the text of the answer. The
// model is asked for a JSON object (response_format json_object); the caller parses it.
// Errors never contain the messages or the answer, so they are safe to log.
func (c *Client) CompleteJSON(ctx context.Context, system, user string) (string, error) {
	if err := checkModel(c.Model); err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]any{
		"model":           c.Model,
		"messages":        []message{{"system", system}, {"user", user}},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.3,
		"max_tokens":      MaxTokens,
	})
	if err != nil {
		return "", err
	}
	url := c.URL
	if url == "" {
		url = DefaultURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("mistral: request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("mistral: read answer: %w", err)
	}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		return "", ErrLimit
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("mistral: HTTP %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return "", errors.New("mistral: unexpected answer format")
	}
	ch := out.Choices[0]
	if ch.FinishReason == "length" {
		return "", errors.New("mistral: answer cut off (max_tokens)")
	}
	text := contentText(ch.Message.Content)
	if text == "" {
		return "", errors.New("mistral: empty answer")
	}
	return text, nil
}

// contentText reads message.content, which is a string or a list of chunks.
func contentText(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var chunks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &chunks); err != nil {
		return ""
	}
	var b strings.Builder
	for _, c := range chunks {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}
