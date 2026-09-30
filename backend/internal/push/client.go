package push

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Environment variables of the native push credentials. Without the APNs key, iOS
// devices get nothing; without the FCM service account, Android devices get nothing.
// Both are files, kept next to api.env (see deploy/api.env.example).
const (
	// EnvAPNSKeyFile is the path of the .p8 key (Apple Developer, Keys, "Apple Push
	// Notifications service").
	EnvAPNSKeyFile = "REITERHOF_APNS_KEY_FILE"
	// EnvAPNSKeyID is the 10 character key ID shown next to the key.
	EnvAPNSKeyID = "REITERHOF_APNS_KEY_ID"
	// EnvAPNSTeamID is the 10 character team ID of the developer account.
	EnvAPNSTeamID = "REITERHOF_APNS_TEAM_ID"
	// EnvAPNSTopic is the bundle identifier; defaults to DefaultAPNSTopic.
	EnvAPNSTopic = "REITERHOF_APNS_TOPIC"
	// EnvAPNSSandbox set to true uses the sandbox host (development builds from Xcode).
	EnvAPNSSandbox = "REITERHOF_APNS_SANDBOX"
	// EnvFCMServiceAccountFile is the path of the Firebase service account JSON key.
	EnvFCMServiceAccountFile = "REITERHOF_FCM_SERVICE_ACCOUNT_FILE"

	// DefaultAPNSTopic is the app's bundle identifier (mobile/app.json).
	DefaultAPNSTopic = "de.flusinerd.stallfunk"
)

// Client is the Sender for native devices: iOS tokens go to APNs, Android tokens to FCM,
// each directly from this server (no push relay in between). A platform without
// credentials is skipped with a log line; its messages are neither failures nor invalid.
type Client struct {
	// APNS delivers to iOS devices (*APNSClient); nil when not configured.
	APNS Sender
	// FCM delivers to Android devices (*FCMClient); nil when not configured.
	FCM Sender
	Log *slog.Logger
}

// NewClientFromEnv builds the Client from the REITERHOF_APNS_* and REITERHOF_FCM_*
// variables. The returned Client is always usable: a platform whose variables are unset
// is left out, one whose credentials are broken is left out and reported in err (the
// other platform still works).
func NewClientFromEnv(log *slog.Logger) (*Client, error) {
	if log == nil {
		log = slog.Default()
	}
	c := &Client{Log: log}
	var errs []error

	if keyFile := os.Getenv(EnvAPNSKeyFile); keyFile != "" {
		apns, err := apnsFromEnv(keyFile)
		if err != nil {
			errs = append(errs, err)
		} else {
			c.APNS = apns
		}
	}
	if saFile := os.Getenv(EnvFCMServiceAccountFile); saFile != "" {
		raw, err := os.ReadFile(saFile)
		if err != nil {
			errs = append(errs, fmt.Errorf("push: %s: %w", EnvFCMServiceAccountFile, err))
		} else if fcm, err := NewFCMClient(raw); err != nil {
			errs = append(errs, err)
		} else {
			c.FCM = fcm
		}
	}
	return c, errors.Join(errs...)
}

func apnsFromEnv(keyFile string) (*APNSClient, error) {
	raw, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("push: %s: %w", EnvAPNSKeyFile, err)
	}
	topic := os.Getenv(EnvAPNSTopic)
	if topic == "" {
		topic = DefaultAPNSTopic
	}
	apns, err := NewAPNSClient(raw, os.Getenv(EnvAPNSKeyID), os.Getenv(EnvAPNSTeamID), topic)
	if err != nil {
		return nil, err
	}
	apns.HTTPClient = NewAPNSHTTPClient()
	if sandbox, _ := strconv.ParseBool(os.Getenv(EnvAPNSSandbox)); sandbox {
		apns.BaseURL = APNSSandboxURL
	}
	return apns, nil
}

// Configured lists the platforms with credentials, for the startup log.
func (c *Client) Configured() []string {
	var out []string
	if c.APNS != nil {
		out = append(out, PlatformIOS)
	}
	if c.FCM != nil {
		out = append(out, PlatformAndroid)
	}
	return out
}

// Send routes every message to the service of its platform. Problems of both services
// are merged into one *SendError.
func (c *Client) Send(ctx context.Context, msgs []Message) error {
	var ios, android []Message
	var serr SendError
	for _, m := range msgs {
		switch m.Platform {
		case PlatformIOS:
			ios = append(ios, m)
		case PlatformAndroid:
			android = append(android, m)
		default:
			serr.Failures = append(serr.Failures, fmt.Sprintf("unknown platform %q", m.Platform))
		}
	}
	serr.merge(c.sendPlatform(ctx, PlatformIOS, ios))
	serr.merge(c.sendPlatform(ctx, PlatformAndroid, android))
	if len(serr.InvalidTokens) == 0 && len(serr.Failures) == 0 {
		return nil
	}
	return &serr
}

func (c *Client) sendPlatform(ctx context.Context, platform string, msgs []Message) error {
	if len(msgs) == 0 {
		return nil
	}
	var sender Sender
	switch platform {
	case PlatformIOS:
		sender = c.APNS
	case PlatformAndroid:
		sender = c.FCM
	}
	if sender == nil {
		c.log().Info("push: no credentials for this platform, skipping devices", "platform", platform, "count", len(msgs))
		return nil
	}
	return sender.Send(ctx, msgs)
}

func (c *Client) log() *slog.Logger {
	if c.Log != nil {
		return c.Log
	}
	return slog.Default()
}

// merge adds the problems of another send (nil, a *SendError or any error) to e.
func (e *SendError) merge(err error) {
	if err == nil {
		return
	}
	var serr *SendError
	if errors.As(err, &serr) {
		e.InvalidTokens = append(e.InvalidTokens, serr.InvalidTokens...)
		e.Failures = append(e.Failures, serr.Failures...)
		return
	}
	e.Failures = append(e.Failures, err.Error())
}
