# Web Push (PWA)

JAN-74. The app ships first as a PWA installed on iPhones through Safari (iOS 16.4+, "Zum Home-Bildschirm").
Notifications there use the standard Web Push protocol. Kinds, consents and opt-outs are the same as for Expo push
(see [architecture.md](../architecture.md#push), [reminders.md](reminders.md), [privacy.md](privacy.md)).

## Setup (operator)

1. `cd backend && go run ./cmd/vapidkeys -subject mailto:you@example.org` prints three lines in env-file format.
2. Put them in the API's env file (`deploy/api.env.example`, section in `deploy/README.md`):

   | Variable | Meaning |
   | --- | --- |
   | `REITERHOF_VAPID_PUBLIC_KEY` | 65 byte uncompressed P-256 point, base64url (the `applicationServerKey`) |
   | `REITERHOF_VAPID_PRIVATE_KEY` | 32 byte scalar, base64url. Secret. |
   | `REITERHOF_VAPID_SUBJECT` | `mailto:` or `https://` contact for the push services |

3. Never change the pair afterwards: a subscription is bound to the public key. (If it happens anyway, the app
   re-subscribes at the next start when the key from `GET /push/web/public-key` differs from the subscription's.)

Without keys web push is off: the public-key and subscription endpoints answer `503 not_configured` and
`push.Notifier` skips web subscriptions with a log line. A half-set or mismatching pair is logged as an error at
startup and also disables it. The pushes still need the `https` origin of the PWA (service workers and the Push API
do not exist on plain http, except `localhost`).

## Backend

- **Storage**: table `web_push_subscriptions` (migration 0130): `endpoint` (unique), `p256dh`, `auth`, `user_id`,
  `stable_id`, `user_agent`, `created_at`. It is a separate table instead of extra columns on `push_tokens`: a
  web subscription is a URL plus two keys that must be present to encrypt every message, `push_tokens.token` is a
  single opaque string and `platform` is CHECKed to `ios`/`android`. Ownership is the same: upsert by endpoint (a
  browser handed to another user is re-assigned; the user must belong to the stable), at most 10 per user (oldest
  dropped), removed with the user, on revoking the `push` consent (`internal/privacy`), on account deletion and when
  the push service answers 404/410. The data export lists them without the endpoint URL (only the push service host).
- **Recipients**: `push.Notifier.NotifyUsers` builds the recipient list for Expo tokens and web subscriptions from
  one SQL fragment (`recipientFilter` in `notifier.go`): stable, current `push` consent, opt-out per kind, opt-in for
  `new_request`. Change consent or opt-out semantics there and both paths follow. The `httpx.Notifier` interface is
  unchanged; the two deliveries are independent (a failure on one does not skip the other; errors are joined).
- **Sender** (`websender.go`, `webcrypto.go`, `vapid.go`): standard library only. RFC 8291 (`aes128gcm`: ECDH P-256,
  HKDF-SHA256 with the auth secret, one record, no padding) and RFC 8292 (`Authorization: vapid t=<ES256 JWT>, k=<public key>`,
  12 h expiry). Headers: `TTL` (24 h; 4 h for `last_person`, 6 h for `weather_change`), `Urgency: high` for
  `urgent_observation` and `last_person`, otherwise `normal`; `Topic` is supported by `WebMessage` but not used yet.
  404/410 delete the subscription, 413/429/5xx/401/403 are returned as `*push.SendError` failures (never with the
  endpoint URL in the text). The HTTP client refuses loopback/private/link-local addresses and redirects, because the
  endpoint is chosen by the client; the API additionally only accepts https URLs with a named host on port 443.
  `push.FakeWeb` records web messages for tests of other packages.
- **Payload** (JSON, at most 3993 bytes): `{"title", "body", "data": {"screen", "kind", ...}}`, `data` identical to the
  Expo message. Every push carries `data.screen` (see architecture.md).
- **Endpoints** (all need a signed-in user with a stable):
  - `GET /api/v1/push/web/public-key` gives `{"public_key": "<base64url>"}`, or `503 not_configured`.
  - `POST /api/v1/me/web-push-subscriptions` with the browser's `PushSubscription.toJSON()`
    (`{"endpoint", "keys": {"p256dh", "auth"}}`; `expirationTime` is accepted and ignored). Validates the https
    endpoint and the key shapes (P-256 point, 16 byte auth): `400 invalid_subscription`; `204` on success; `503` when unconfigured.
  - `DELETE /api/v1/me/web-push-subscriptions` with `{"endpoint"}`; `204` (works without keys).

## Mobile (web build only; native is unchanged)

- `public/sw.js`: the service worker (served at `/sw.js`). `push` shows the notification (always, Safari revokes
  subscriptions that stay silent) with `icon: /icon-192.png` and a `tag` of kind and item id so repeated pushes
  replace each other. `notificationclick` focuses a running app window and `postMessage`s
  `{type: "stallfunk:navigate", route}`, or opens the route as URL when no window exists. The route is only used
  when it passes the whitelist, which is a copy of `lib/notifications-core.ts`; `lib/webpush-sw.test.ts` loads the
  worker and compares its behavior with `notificationRoute`. There is no caching in it. A caching layer belongs into a
  separate file that `sw.js` pulls in with `importScripts` (the worker exports nothing else that could get in the way).
  `/icon-192.png` must exist in the web build (`public/`, owned by the PWA shell).
- `lib/service-worker.web.ts` exports `registerServiceWorker()` (registers `/sw.js` with scope `/`; idempotent,
  never throws, `service-worker.ts` is the native no-op). **Anything else on the web that needs the worker must call
  this function** instead of `navigator.serviceWorker.register`, so there is one registration. `enableWebPush` /
  `syncWebPush` call it.
- `lib/webpush.ts`: `enableWebPush()` (permission, subscribe with the VAPID key, `POST` the subscription), `syncWebPush()`
  (no prompt, only with the permission already granted; run by `useDeviceSetup` at app start), `disableWebPush()`
  (push consent withdrawn), `webPushSupportNow()`, `webPushPermission()`. Pure helpers (base64url, support/reason
  mapping, German messages) are in `lib/webpush-core.ts`.
- **User gesture**: iOS only shows the permission prompt inside a tap handler. `ConsentSheet` ("Erlauben" of the push
  consent, also reached from the settings banner "Mitteilungen erlauben") calls `enableWebPush()` before its first
  `await`. `WebPushCard` (settings) offers the button when the consent exists but the browser was not asked yet.
- **Not installed on iOS** (Safari tab: `navigator.standalone` false, no `PushManager`): `enableWebPush` returns
  `needs_install` and the UI shows "Zum Home-Bildschirm hinzufügen, dann Mitteilungen erlauben". In the consent sheet
  this also means no consent is recorded yet, since "Erlauben" cannot work. Other reasons: `unsupported`,
  `permission_denied`, `not_configured` (503), `error`.
- Tap inside the app: `useWebPushNavigation` (called by `useDeviceSetup`) receives the worker's message, checks the
  route again with `isInternalRoute`, waits for sign-in and calls `router.push`.

## Tests

Go: RFC 8291 appendix A known-answer vector, encrypt/decrypt round trip with a test receiver implemented in the test,
JWT signature check, `httptest` push service (201, 404, 410, 413, 429, 5xx), dispatch with consent/opt-out/opt-in filters,
cleanup on revoke and account deletion. Mobile: `lib/webpush-core.test.ts`, `lib/webpush-sw.test.ts`.

## Manual test on an iPhone

Serve the PWA over https (a real domain or a tunnel), open it in Safari, Share > "Zum Home-Bildschirm", open the icon,
sign in, grant the push consent. Trigger a push, for instance an urgent observation from a second account. With the app in the
background the banner appears; a tap opens the observation.
