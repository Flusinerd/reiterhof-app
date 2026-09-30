package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the Expo Push API endpoint.
	DefaultBaseURL = "https://exp.host/--/api/v2/push/send"
	// EnvAccessToken optionally holds an Expo access token (enhanced security).
	EnvAccessToken = "REITERHOF_EXPO_ACCESS_TOKEN"
	// MaxBatch is the maximum number of messages per Expo request.
	MaxBatch = 100
)

// Client is a Sender for the Expo Push API.
type Client struct {
	// BaseURL is the full send URL; defaults to DefaultBaseURL.
	BaseURL string
	// HTTPClient defaults to a client with a 15 s timeout.
	HTTPClient *http.Client
	// AccessToken is sent as a bearer token when non-empty.
	AccessToken string
	// OnInvalidToken, if set, is called for every DeviceNotRegistered token
	// (in addition to the token being listed in the returned *SendError).
	OnInvalidToken func(ctx context.Context, token string)
}

// NewClientFromEnv returns a Client using REITERHOF_EXPO_ACCESS_TOKEN.
func NewClientFromEnv() *Client {
	return &Client{AccessToken: os.Getenv(EnvAccessToken)}
}

type ticket struct {
	Status  string `json:"status"`
	ID      string `json:"id"`
	Message string `json:"message"`
	Details struct {
		Error string `json:"error"`
	} `json:"details"`
}

type response struct {
	Data   []ticket `json:"data"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// Send posts msgs in batches of at most MaxBatch. A failing batch does not
// stop the following ones. Problems are returned as *SendError.
func (c *Client) Send(ctx context.Context, msgs []Message) error {
	var serr SendError
	for start := 0; start < len(msgs); start += MaxBatch {
		end := min(start+MaxBatch, len(msgs))
		if ctx.Err() != nil {
			serr.Failures = append(serr.Failures, ctx.Err().Error())
			break
		}
		c.sendBatch(ctx, msgs[start:end], &serr)
	}
	if len(serr.InvalidTokens) == 0 && len(serr.Failures) == 0 {
		return nil
	}
	return &serr
}

func (c *Client) sendBatch(ctx context.Context, batch []Message, serr *SendError) {
	body, err := json.Marshal(batch)
	if err != nil {
		serr.Failures = append(serr.Failures, "marshal: "+err.Error())
		return
	}
	url := c.BaseURL
	if url == "" {
		url = DefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		serr.Failures = append(serr.Failures, "request: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		serr.Failures = append(serr.Failures, fmt.Sprintf("batch of %d: %v", len(batch), err))
		return
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		serr.Failures = append(serr.Failures, fmt.Sprintf("batch of %d: read: %v", len(batch), err))
		return
	}
	var out response
	jsonErr := json.Unmarshal(raw, &out)
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(raw))
		if jsonErr == nil && len(out.Errors) > 0 {
			msg = out.Errors[0].Code + ": " + out.Errors[0].Message
		}
		serr.Failures = append(serr.Failures, fmt.Sprintf("batch of %d: http %d: %.200s", len(batch), resp.StatusCode, msg))
		return
	}
	if jsonErr != nil {
		serr.Failures = append(serr.Failures, fmt.Sprintf("batch of %d: invalid response: %v", len(batch), jsonErr))
		return
	}
	if len(out.Data) != len(batch) {
		// Still evaluate the tickets that can be matched by index.
		serr.Failures = append(serr.Failures, fmt.Sprintf("batch of %d: got %d tickets", len(batch), len(out.Data)))
	}
	for i, t := range out.Data {
		if i >= len(batch) || t.Status == "ok" {
			continue
		}
		if t.Details.Error == "DeviceNotRegistered" {
			serr.InvalidTokens = append(serr.InvalidTokens, batch[i].To)
			if c.OnInvalidToken != nil {
				c.OnInvalidToken(ctx, batch[i].To)
			}
			continue
		}
		serr.Failures = append(serr.Failures, fmt.Sprintf("%s: %s %s", batch[i].To, t.Details.Error, t.Message))
	}
}
