-- Web Push (JAN-74): PushSubscriptions of the PWA (Safari on iOS 16.4+, other browsers).
-- A separate table instead of push_tokens: an Expo token is one opaque string per device,
-- a web subscription is an endpoint URL plus two keys (p256dh, auth) that are needed to
-- encrypt every message (RFC 8291); push_tokens.platform is CHECKed to ios/android.
-- Same ownership and lifecycle as push_tokens: unique per endpoint (a browser handed to
-- another user is re-assigned), deleted with the user, on consent revoke and on 404/410.
CREATE TABLE web_push_subscriptions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    endpoint   text NOT NULL UNIQUE,
    p256dh     text NOT NULL,
    auth       text NOT NULL,
    user_agent text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX web_push_subscriptions_user_idx ON web_push_subscriptions (user_id);
CREATE INDEX web_push_subscriptions_stable_idx ON web_push_subscriptions (stable_id);
