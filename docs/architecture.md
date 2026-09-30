# Backend architecture

The backend is a single Go service (`backend/`), `net/http` `ServeMux` only, no
framework, Postgres 16 via `pgx/v5`. Everything is scoped by `stable_id`; the Go
API enforces this. Postgres row-level security is **not** used.

## Package layout

```
backend/
  cmd/api/            API server (opens pool, runs migrations, serves HTTP)
  cmd/migrate/        applies pending migrations
  cmd/seed/           migrates, then loads the example data (idempotent)
  cmd/admin/          stallfunk-admin, the operator CLI (see "Operations: admin CLI")
  migrations/         NNNN_description.up.sql, embedded via go:embed (migrations.FS)
  internal/
    config/           environment configuration
    db/               pgx pool (db.Open) and migrator (db.Migrate)
    dbtest/           throwaway databases for tests
    seed/             example data + exported fixed IDs (seed.HorseLuna, ...)
    admincli/         commands of stallfunk-admin (env file parser, stable/user/invite/horse/...)
    scheduler/        reusable periodic / daily jobs (see "Scheduler")
    weather/          DWD MOSMIX client, night summary, weather_snapshots, hourly job
    mistral/          chat completions client for the AI week plan (httpx.Deps.Chat)
    blanketplan/      pure rule evaluation (Recommend, Changed) + LoadRules
    stables/          stable data access: ground condition, stables with coordinates
    httpx/            Deps, WriteJSON, ReadJSON, WriteError (shared by all handlers)
    auth/             sessions, sign-in, middleware, role checks; authtest/ for domain tests
    httpapi/          router assembly: NewHandler, /healthz, /readyz, registration list
    <domain>/         one package per domain, e.g. horses, requests, blankets
```

Import rule: domain packages import `httpx`, `db`, `config` (never `httpapi`, which
imports the domain packages). `httpapi.Deps` is an alias of `httpx.Deps`.

Configuration (environment):

| Variable | Default | Purpose |
| --- | --- | --- |
| `REITERHOF_ADDR` | `:8080` | listen address |
| `REITERHOF_DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/reiterhof?sslmode=disable` | app database |
| `REITERHOF_TEST_DATABASE_URL` | unset | admin URL for DB tests; unset means they skip |
| `REITERHOF_WEATHER_ENABLED` | `true` | run the hourly DWD weather job |
| `REITERHOF_WEATHER_STATION` | unset | force one MOSMIX station id for all stables; unset = nearest per stable |
| `REITERHOF_APNS_KEY_FILE`, `_KEY_ID`, `_TEAM_ID`, `_TOPIC`, `_SANDBOX` | unset | APNs token authentication (.p8 key file, key ID, team ID; topic defaults to the bundle id; sandbox for Xcode builds). Unset = no iOS push (`push.NewClientFromEnv`) |
| `REITERHOF_FCM_SERVICE_ACCOUNT_FILE` | unset | Firebase service account JSON for FCM HTTP v1. Unset = no Android push |
| `REITERHOF_VAPID_PUBLIC_KEY`, `_PRIVATE_KEY`, `_SUBJECT` | unset | VAPID identity for Web Push (base64url P-256 keys, `mailto:`/`https:` subject); unset = web push off (`cmd/vapidkeys` generates a pair) |
| `REITERHOF_PUBLIC_URL`, `REITERHOF_WEB_URL`, `REITERHOF_LOGIN_CODE_KEY`, `REITERHOF_SMTP_*`, `REITERHOF_*_CLIENT_IDS`, `REITERHOF_DEV_LOGIN` | unset | authentication, see [Authentication and roles](#authentication-and-roles) |

## Adding a domain package

1. Create `backend/internal/<domain>/` with a `Register` function:

   ```go
   package horses

   func Register(mux *http.ServeMux, deps httpx.Deps) {
       h := &handler{deps: deps}
       mux.HandleFunc("GET /api/v1/horses", h.list)
       mux.HandleFunc("GET /api/v1/horses/{id}", h.get)
   }
   ```

2. Add `horses.Register` to the `registrations` list in
   `backend/internal/httpapi/httpapi.go` (one line, keep it alphabetical to keep merge
   conflicts trivial).
3. Routes live under `/api/v1/...`; `/healthz` and `/readyz` are unauthenticated.
   Use Go 1.22+ patterns (`"METHOD /path/{param}"`, `r.PathValue("id")`).
4. Use `deps.Pool` for SQL (plain `pgx`, no ORM), `deps.Log` for logging, and
   `deps.Now()` instead of `time.Now()`.
5. Every query on a domain table filters by the current user's `stable_id`
   (`WHERE stable_id = $1 AND ...`). Never trust a `stable_id` from the request body.
6. If you need a table, add a migration (below) and tests using `dbtest`.

Handlers write responses only through `httpx.WriteJSON` / `httpx.WriteError` and read
bodies through `httpx.ReadJSON` (1 MiB limit, unknown fields rejected; it writes the
error response itself, so `if !httpx.ReadJSON(w, r, &in) { return }`).

## Error format

Every error response has this body, with a snake_case machine-readable `code` and a
human-readable `message`:

```json
{"error": {"code": "not_found", "message": "horse not found"}}
```

Conventions: `400 invalid_json` / `validation_failed`, `401 unauthorized`,
`403 forbidden`, `404 not_found`, `409 conflict`, `413 body_too_large`,
`500 internal` (never leak internals; log them with `deps.Log`).

## Migrations

- Files: `backend/migrations/NNNN_description.up.sql` (four digits, lowercase snake
  case). Up only, no down migrations. Each file runs in one transaction.
- They are embedded, recorded in `schema_migrations(version, name, applied_at)`,
  applied in version order, guarded by a Postgres advisory lock, and run at API start
  (also via `just migrate`).
- Never edit a migration that has been merged; add a new one.
- **Numbering:** use the next free number, or the number assigned to you. Gaps are
  allowed and expected. Current allocation: `0001`-`0003` core schema
  (JAN-23), `0010` auth tables (auth agent), `0140` login code columns (JAN-76), `0150` automatic uncovering (JAN-78). Unapplied lower numbers still apply
  after higher ones, so parallel branches do not block each other. Two files with the
  same number make the migrator fail, which surfaces a clash at merge time.
- Schema conventions: UUID primary keys (`gen_random_uuid()`), `created_at timestamptz`,
  `stable_id uuid NOT NULL REFERENCES stables(id)` on every domain table, enums as
  `text` + `CHECK`, an index starting with `stable_id` for list queries.

## Operations: admin CLI

There is no API to create the first stable or the first admin, and some maintenance
(demoting, moving users, deleting an account by hand) is not an app feature. The operator
tool `stallfunk-admin` (`backend/cmd/admin`, commands in `internal/admincli`) covers this
instead of hand-written SQL:

- Commands: `stable list|create|update`, `user list|create|promote|demote|move|delete`,
  `invite create|list`, `horse list|transfer`, `sessions revoke`, `mail test`,
  `weather refresh`, `migrate status|up`; `stallfunk-admin help` lists the flags. Standard
  library `flag` only. `--json` prints list output as JSON, `--yes` skips the confirmation
  of `user delete|demote|move` and `horse transfer`.
- It reads the same environment as the API. `--env-file` parses a systemd
  `EnvironmentFile` (`admincli.ParseEnvFile`); without the flag `/etc/reiterhof/api.env` is
  used when it exists. Variables that are already set win.
- It reuses the API code instead of duplicating rules: `privacy.DeleteAccount` (same
  blocking errors `owns_horses` / `last_admin`, same anonymisation), `auth.CreateInvite`
  and `auth.NormalizeEmail`, the auth mailer (`mail test`, which refuses to fall back to the
  log mailer), `weather.Service` plus `blankets.CheckWeatherChange` (`weather refresh`),
  `db.Migrate` / `db.Status`. With exactly one stable, `--stable` defaults to it.
- Testability: every command is a function of `admincli.Env` (pool, config, `In`/`Out`/`Err`),
  so tests call `admincli.Run` on a `dbtest.NewSeeded` database and buffers, no exec.
- Delivery: the Deploy workflow builds it as a second static binary and
  `deploy/remote-swap.sh` installs it as `/opt/reiterhof/stallfunk-admin` (after the API
  is healthy, atomic rename, no extra sudo rights). `provision.sh` installs the wrapper
  `/usr/local/bin/stallfunk-admin` (`deploy/stallfunk-admin.sh`), which runs the binary as
  user `reiterhof` (the only non-root user that can read `api.env`) via sudo. Operator
  instructions: `deploy/README.md`, "Betrieb: stallfunk-admin". Locally: `just admin <args>`.

## Authentication and roles

Passwordless, no SMS, no external auth service: sessions live in our Postgres
(`backend/internal/auth`, migration `0010_auth.up.sql`).

Sign-in methods (all return `{"token": "...", "user": {...}}`; the token is an opaque
bearer token, only its SHA-256 hash is stored, sessions last 90 days from the last use):

| Route (all `POST`, no auth) | Body | Notes |
| --- | --- | --- |
| `/api/v1/auth/magic-link` | `{email}` | Always `204` (no user enumeration), `429` when rate limited (3 per email / 15 min, 20 per IP / h). Mail contains `stallfunk://auth/verify?token=...` and, if `REITERHOF_PUBLIC_URL` is set, an https fallback `<url>/auth/verify?token=...` (a page in the app's design that only links to the app, it never consumes the token; it sends its own Content-Security-Policy with the hash of its stylesheet, Caddy's `default-src 'none'` is only a default). Tokens live 15 min and work once. The same mail carries a 6-digit login code (see below); subject `Dein Stallfunk-Code: 123456`. The mail is branded HTML with a plain-text alternative (`auth.MailContent`, `internal/auth/mailtmpl.go`): logo inline as `cid:` image, code box, and a button to the https fallback (HTML never carries the `stallfunk://` link, mail clients drop custom schemes). `mail test` sends the same layout. |
| `/api/v1/auth/verify` | `{token}` | Creates the user if the email is new. `401 invalid_token` if unknown, used or expired. |
| `/api/v1/auth/verify-code` | `{email, code}` | Signs in with the 6-digit code from the mail, exactly like `/auth/verify` (`code` may contain spaces or dashes). `401 invalid_code` for every failure (unknown email, wrong, expired, used, locked out), `429 rate_limited` after 30 calls per IP / 15 min. |
| `/api/v1/auth/google` | `{id_token}` | RS256 ID token, checked against Google's JWKS (cached), `iss`, `aud` (`REITERHOF_GOOGLE_CLIENT_IDS`), `exp`. |
| `/api/v1/auth/apple` | `{id_token, name?}` | Same for Apple (`REITERHOF_APPLE_CLIENT_IDS`); Apple sends the name only to the client on first sign-in, so the app passes it along. |
| `/api/v1/auth/dev-login` | `{email}` | `404` unless `REITERHOF_DEV_LOGIN=true` (the API logs a warning at start while it is on). Signs in (or creates) any email, for local testing with seed users (`jan@example.org` is admin). Never enable in production. |

**Login code** (migration `0140_login_code.up.sql`, `internal/auth/logincode.go`). Why: on iOS the
mail link opens in Safari, not in the installed home-screen PWA (separate storage), so the
user types a code instead; native users can use it too.

- Every magic-link request creates one `login_tokens` row (one login attempt) holding both the link
  token hash and `code_hash`. The code is 6 digits from `crypto/rand` (uniform, leading zeros
  allowed) and valid 15 minutes like the link. Using **either** the link or the code sets
  `used_at` and thereby consumes both.
- Storage: `code_hash = HMAC-SHA256(REITERHOF_LOGIN_CODE_KEY, email || 0x00 || code)`. A plain
  or salted hash of a 6-digit code falls to an offline search in milliseconds if the table
  leaks (a per-row salt does not help, there are only 10^6 inputs); the secret key lives
  outside the database, so a database leak alone is not enough. The email binds a hash to its
  address. Without the env var a random per-process key is used (warning in the log): codes
  requested before a restart stop working, links do not.
- Brute force: at most 5 wrong codes per login attempt; the fifth sets `code_hash = NULL`
  (the code is dead, the 256-bit link of the same attempt stays valid), plus 30 calls per IP
  / 15 min, and the existing request limits (3 mails per email / 15 min) cap guesses at 15 per
  email and window. A row is locked with `FOR UPDATE` while checked, so parallel guesses cannot
  race the counter. The compare is constant time; unknown emails do the same work.
- A new request for an email revokes the codes of all older unused attempts of that email (their
  links keep working until they expire). Only the newest mail's code is valid.
- The code is in the subject on purpose: convenient (visible in the notification, autofill),
  and it is only usable together with the email address, within 15 minutes, once, and with 5
  guesses. Trade-off: it shows on a lock screen. Change the subject in `magicLink` if that matters.

Identity of Google/Apple users is the provider `sub` (`auth_identities`, unique per
provider and subject). A new identity is attached to the account with the same *verified*
email (merge); Apple relay addresses are ordinary emails and so match nothing. Tokens
without verified email are rejected (`401 email_not_verified`).

Authenticated routes: `POST /api/v1/auth/logout` (`204`), `GET /api/v1/me` (user with `age_status`
and `parent_email`, stable or `null`, `roles.owned_horse_ids`, `roles.rider_horse_ids`), `PATCH /api/v1/me`
(`name`, `phone`, `avatar_color`, `presence_visibility`; `""` clears phone/avatar_color),
`POST /api/v1/me/age` and `POST /api/v1/me/parental-consent` (age confirmation, Art. 8 GDPR; see
[domains/privacy.md](domains/privacy.md#age-confirmation-and-parental-consent-jan-86)),
`POST /api/v1/stables/join` (`{code}`, sets `users.stable_id`; `404 invalid_code`,
`409 already_in_stable`, `403 age_unconfirmed`), `POST /api/v1/stables/invites` (admin only, optional
`{expires_in_days (1..90, default 7), max_uses (1..100, default 10)}`, returns
`{code: "ABCD-EFGH", expires_at, max_uses}`). Codes are case- and separator-insensitive.
Without a session: `GET/POST /parental-consent` (the parent's page, own CSP like `/auth/verify`).

The session token travels in the `Authorization` header only; no route reads it from the query
string (histories, proxies and share sheets would keep it). File downloads that cannot send
headers use short-lived download links (`files.DownloadLink`, see [Files](#files)).

Stable-less users: a person may sign up before joining a stable, so `users.stable_id`
is **nullable**. Domain handlers must never see such users; wrap them in `RequireStable`.
(The first stable and its first admin are created by seed/ops, there is no API for it.)

### Contract for domain packages

```go
// IDs are UUIDs in text form (string), as pgx reads and writes uuid columns.
type User struct {
    ID       string
    StableID string // "" only if the user has not joined a stable (never behind RequireStable)
    IsAdmin  bool
    Name     string
}

func UserFrom(ctx context.Context) (User, bool)

func Middleware(deps httpx.Deps) func(http.Handler) http.Handler // wraps the whole mux; httpapi does it
func RequireUser(h http.Handler) http.Handler   // 401 unauthorized
func RequireStable(h http.Handler) http.Handler // 401 unauthorized, 403 no_stable
func RequireAdmin(h http.Handler) http.Handler  // RequireStable + 403 forbidden unless IsAdmin
```

`Middleware` never rejects; it only sets the user when the `Authorization: Bearer` token
is valid. Protect every domain route:

```go
func Register(mux *http.ServeMux, deps httpx.Deps) {
    h := &handler{deps: deps}
    mux.Handle("GET /api/v1/horses/{id}", auth.RequireStable(http.HandlerFunc(h.get)))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
    user, _ := auth.UserFrom(r.Context()) // ok is always true behind RequireStable
    ok, err := auth.HorseInStable(r.Context(), h.deps.Pool, user, r.PathValue("id"))
    ...
}
```

Role model (derived from data, nothing stored; all helpers filter by `stable_id`, take a
`*pgxpool.Pool` or `pgx.Tx` as `q`, return `false` for users without stable or malformed ids):

| Role | Definition | May |
| --- | --- | --- |
| member | user with a stable | see presence, blanket list, requests, emergency cards; accept requests, blanket horses, report |
| rider (RB) | row in `horse_riders` for the horse; `rules` is a positive list (`["ride","groom"]`) | in addition: log sessions, report observations, take week slots for that horse. Never change rules or profiles |
| owner | `horses.owner_id = user` | full control of the own horse: profile, riders and rules |
| admin | `users.is_admin` | everything owners may on every horse of the stable, plus `stables/invites` |

```go
auth.HorseInStable(ctx, q, user, horseID) (bool, error)
auth.IsOwner(ctx, q, user, horseID) (bool, error)
auth.CanManageHorse(ctx, q, user, horseID) (bool, error) // owner or admin
auth.IsRider(ctx, q, user, horseID) (bool, error)        // owners are not implicit riders
auth.RiderRules(ctx, q, user, horseID) (rules []string, isRider bool, err error)
```

Return `404` for a horse that is not in the stable (do not reveal other stables) and `403 forbidden`
for a horse in the stable the user may not change.

Testing helpers (`internal/auth/authtest`):

```go
pool := dbtest.NewSeeded(t)
req := httptest.NewRequest("GET", "/api/v1/horses", nil)
authtest.Authorize(t, pool, req, seed.UserJan)     // real session, Bearer header set
ctx := authtest.WithUser(ctx, auth.User{ID: seed.UserAnna, StableID: seed.StableB}) // for direct handler calls
token := authtest.Token(t, pool, seed.UserMia)      // TokenAt(t, pool, id, now) if the test fixes the clock
```

Configuration (all optional; unset means the feature is off or in dev mode):

| Variable | Purpose |
| --- | --- |
| `REITERHOF_PUBLIC_URL` | public base URL of the API, e.g. `https://api.example.org`, for the mail's https fallback link and the parental consent link |
| `REITERHOF_WEB_URL` | public base URL of the web app, e.g. `https://example.org`; pages and mails link `<url>/legal/privacy` |
| `REITERHOF_LOGIN_CODE_KEY` | HMAC key of the stored login codes; the download link key is derived from it |
| `REITERHOF_SMTP_HOST`, `_PORT` (587), `_USER`, `_PASSWORD`, `_FROM` | SMTP for login mails (STARTTLS, port 465 = implicit TLS). Without host the mail is only logged (dev) |
| `REITERHOF_GOOGLE_CLIENT_IDS` | comma separated OAuth client IDs (web, iOS, Android) accepted as `aud` |
| `REITERHOF_APPLE_CLIENT_IDS` | comma separated accepted `aud` (iOS bundle ID `de.flusinerd.stallfunk`) |
| `REITERHOF_DEV_LOGIN` | `true` enables `/auth/dev-login` |

Rate limits are in memory (per process, reset on restart), fine for the single-VPS setup.
The rate limiter uses `X-Forwarded-For` only when the connection comes from loopback (Caddy).

## Push

Package `internal/push` (backend part of JAN-18; native delivery without Expo since JAN-88).
Native device tokens (APNs token on iOS, FCM registration token on Android) live in
`push_tokens` (unique per token, with `platform`), per-user opt-outs in `reminder_settings`.

**In handlers, send pushes via `deps.Notify.NotifyUsers(ctx, stableID, userIDs, kind,
title, body, data)`** (`httpx.Notifier`; never nil, a no-op unless `cmd/api` wires the
real `*push.Notifier`). In tests pass `Notify: push.NewNotifier(pool, fake, nil)` with
`fake := &push.Fake{}` and assert on `fake.Sent()`. Devices register their token via
`POST /api/v1/me/push-tokens {token, platform}` / `DELETE` (package `internal/devices`).

- `push.Sender` (`Send(ctx, []Message) error`) is the seam. `push.Client` routes by
  `Message.Platform`: `APNSClient` posts to APNs over HTTP/2 with token authentication
  (ES256 provider token from the `.p8` key, cached 50 min, refreshed on
  `ExpiredProviderToken`), `FCMClient` posts to FCM HTTP v1 with a service account
  (RS256 assertion, OAuth access token cached until expiry, refreshed on 401). Standard
  library only; base URLs and `*http.Client` are fields, so tests use `httptest` (the
  APNs test server runs HTTP/2). A platform without credentials is skipped with a log
  line. `push.Fake` records messages for tests of other packages.
- Payloads match what `expo-notifications` expects on the device (it also handled the
  Expo service's messages this way): APNs `{"aps":{"alert":{"title","body"},"sound"},"body":{data}}`
  (the top-level `body` object becomes `notification.request.content.data`); FCM is a
  data-only message with `title`, `message`, `body` (the data as a JSON string) and
  `channelId: "default"` (the channel `mobile/lib/push.ts` creates); a `notification`
  block would bypass the library. `Priority` (`PriorityHigh` for `urgent_observation`
  and `last_person`, else normal: `apns-priority` 10/5, FCM `android.priority`) and
  `TTL` (`apns-expiration`, `android.ttl`) follow the Web Push urgency and TTL.
- Problems are returned as `*push.SendError`: `InvalidTokens` (APNs `BadDeviceToken`,
  `Unregistered`, `DeviceTokenNotForTopic` or 410; FCM `UNREGISTERED`, 404, or
  `INVALID_ARGUMENT` about the registration token; tokens that do not look like a
  device token) and `Failures` (everything else, never with the token in the text).
  One failing device does not stop the others (up to 8 in parallel per service).
- `push.NewNotifier(pool, sender, log).NotifyUsers(ctx, stableID, userIDs, kind, title,
  body, data)` loads the tokens of those users **within that stable**, skips users
  whose `reminder_settings` row for `kind` has `enabled = false` (no row = enabled),
  adds `data.kind`, sends, and deletes the invalid tokens (by `stable_id` + token).
  Only non-token failures are returned as error. It also skips users without a current
  `push` consent (`consents`, not revoked; never consented = nothing is sent). Tests that
  register tokens call `pushtest.GrantConsent`.
- Kinds are constants (`push.KindLastPerson`, `KindWeatherChange`, `KindMedication`,
  `KindHelper`, `KindTrainingPlan`, `KindHealthDue`, `KindRehaCheckup`,
  `KindNewRequest`, `KindUrgentObservation`, `KindObservation`; `push.Kinds()`, `push.ValidKind`).
- Store: `push.RegisterToken(ctx, pool, stableID, userID, token, platform)` upserts
  by token (a device handed to another user is re-assigned; the user must belong to
  the stable, else `ErrUnknownUser`; `push.ValidToken` refuses Expo push tokens and
  anything but printable ASCII, `ErrInvalidToken`); `push.DeleteToken(ctx, pool,
  stableID, userID, token)`. Migration 0220 dropped the Expo tokens of old builds.
- `POST /api/v1/me/push-tokens` (and `DELETE`, package `internal/devices`) wrap
  `RegisterToken` / `DeleteToken`; `cmd/api` builds one `push.NewNotifier` and passes it to
  `httpapi.Deps.Notify` and to the reminder jobs.
- **Tap target: every push carries `data.screen`**, an Expo Router path that exists in
  `mobile/app` (`/requests/<id>`, `/observations/<id>`, `/horses/<id>/health`,
  `/horses/<id>/reha`, `/horses/<id>/blanket-plan`, `/blankets`). Add the ids the screen needs
  (`horse_id`, ...) next to it. `push.Notifier` adds `data.kind`. Health and reha pushes also
  keep the older key `route` with the same value.
- Web Push (PWA, JAN-74) runs next to native push in the same `Notifier` with the same filters:
  [domains/push-web.md](domains/push-web.md).
- Mobile: `mobile/lib/push.ts` `registerForPush()` asks for permission and returns
  `{ token, platform }` from `Notifications.getDevicePushTokenAsync()` (a real device;
  Android needs `google-services.json` in `mobile/`, wired in by `app.config.js` when
  the file exists; it does not throw). `useDeviceSetup` (`lib/use-push-registration.ts`)
  sends the token to the backend once per app start, only with the `push` consent.
  No Expo project id, no EAS.

## Time handling

- Store instants as `timestamptz` (UTC); the API exchanges RFC 3339 timestamps.
- "Day" logic (blanket day, presence day, week slots, reminders at 20:30) uses the
  stable's timezone, `stables.timezone` (default `Europe/Berlin`), via
  `time.LoadLocation`. `date` columns hold the stable-local calendar day.
- `stables.reminder_time` is a local wall-clock time.
- Use `deps.Now()` for the current time so tests can fix the clock.

## Testing

- Standard library only (`testing`, `net/http/httptest`); no testify.
- `dbtest.New(t)` creates a uniquely named database from `REITERHOF_TEST_DATABASE_URL`,
  migrates it, returns a `*pgxpool.Pool` and drops the database on cleanup.
  `dbtest.NewSeeded(t)` also loads the seed data; use the constants in `internal/seed`.
  Without the env variable the test is skipped, so `go test ./...` works without
  Postgres.
- Run with the database: `REITERHOF_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable' just test`.
- Test handlers via `httpapi.NewHandler(httpapi.Deps{Pool: pool, Now: fixedClock})`
  and `httptest`, or call your `Register` on a fresh `http.NewServeMux()`.
- Language model: `deps.Chat` is nil without `REITERHOF_MISTRAL_API_KEY`; handlers must then work
  without it. Tests pass a fake `Chat` (see `trainingapi/plan_test.go`). Never log prompts or answers.
- Blanket rules semantics (seed and evaluation): rules are evaluated by `position`,
  the first match wins; `temp_min <= temp < temp_max`, `NULL` bound = open, `NULL` rain
  = any; with `rain = true` also `rain_min_mm <= rain_mm < rain_max_mm` (`NULL` = open);
  `blanket_id NULL` = no blanket.

## Training logic (`internal/training`)

Pure packages (no HTTP, no DB, no clock; "today" is always an input):

- `training`: shared types (`Activity`, `Profile`, `RiderRules`, `Session`, `Intensity`, ...).
- `training/load` (JAN-62): `Score(minutes, activity, canterShare)` = minutes x intensity
  factor (table in the package doc), `Classify` (light < 30, medium 30-60, intense > 60),
  `DayLoad`, `WeekLoad` (7 segments) and `Assess` (one German sentence vs. the rhythm).
- `training/recommend` (JAN-56): `Recommend(Input) Result`, the rule-based "Was heute?".
  Profile/rider rules hide activities (`Hidden` with a German reason), then overrides
  apply in order (active reha phase, rest day after a show, nothing left to choose),
  hard filters (pause/reha = light only, show today/tomorrow = light only, frozen ground
  = no jumping) and finally a weighted score (variety, recent load, show distance,
  weather/ground, available time, status). All weights are named `Weight*` constants;
  ties are broken by the canonical activity order. The top 3 are returned, each with a
  German one-sentence reason (the strongest factor). Full description in the package doc.
  `Check(Input, activity, minutes)` (JAN-89) tests a proposed unit against the same hard
  rules plus the weekly maximum and returns the rule-based replacement when it fails.
  Both apply the week structure (JAN-93, `structure.go`): unit levels by load, the owner's
  weekday rules and quotas, default structure rules, the week's needs when a whole week is
  planned (`Input.Ahead`) and the load limit (+20 % over the two weeks before).
- `training/weekplan` (JAN-89, JAN-92): plans the open days of a week (activity, minutes, focus, exercise). `Rules` uses the recommender day by
  day (each planned unit becomes a simulated session for the next days); `Prompt`/`Parse` build the
  language model's input (no names, ids, free text or dates) and read its JSON answer; `Merge`
  checks every proposal with `recommend.Check` and lets the rules fill the rest. `Day.Minutes` gives a
  planned closed day its duration and `Day.Exclude` lists the activities the owner rejected for an open
  day (JAN-95, re-planning a single day).
- German reason texts are UI text and live in these packages; everything else is English.

## Scheduler

`internal/scheduler` runs background jobs; reuse it for reminders and other timed work
instead of starting your own tickers.

```go
sched := &scheduler.Scheduler{Log: log}            // Clock defaults to the wall clock
sched.Go(ctx, &wg, scheduler.Job{
    Name:       "weather-snapshot",
    Schedule:   scheduler.Every(time.Hour),        // or scheduler.DailyAt(20, 30, loc)
    RunOnStart: true,
    Run:        func(ctx context.Context) error { ... },
})
```

- `Run` blocks until `ctx` is cancelled, `Go` starts it on a `WaitGroup`.
- The next run time is computed after the previous run finished (no catch-up bursts, no
  overlapping runs of one job). No jitter.
- Errors are logged (`job failed`), panics are recovered and logged (`job panicked`); a job
  never takes the process down.
- `DailyAt` uses wall-clock time in the given location (per-stable reminders: pass the
  stable's `time.LoadLocation(stables.timezone)`; one job per stable).
- Tests inject a fake `Clock` (`Now`, `After`); see `scheduler_test.go`.

## Weather and blankets

**Source: DWD open data, MOSMIX_L single station**
(`https://opendata.dwd.de/weather/local_forecasts/mos/MOSMIX_L/single_stations/<ID>/kml/MOSMIX_L_LATEST_<ID>.kmz`),
no third-party weather API. The KMZ (zip with one KML) has hourly steps for ~10 days;
`weather.ParseMOSMIX` reads `TTT` (K to °C), `R101` (rain probability %, fallback `wwP`),
`RR1c` (mm/h) and `FF` (m/s to km/h). `-` means missing.

**Station:** the nearest entry of the small embedded `weather.Stations` list (Haversine); for
the Dorsten stable (51.66, 6.96) that is Essen-Bredeney, id `10410`. Override with
`REITERHOF_WEATHER_STATION` (any MOSMIX id) or extend the list.

**Forecast window** (`weather.Summarize`, `weather.Window`): the stored snapshot
uses the default window, 18:00 stable-local on the day until 12:30 the next day, i.e. from covering in
the evening until the horses come in and are uncovered (wall-clock, so DST nights are one hour longer
or shorter). Every horse has its own window (`horses.cover_start`, `horses.cover_end`, defaults 18:00
and 12:30, see `docs/domains/blankets.md`), so the snapshot also keeps the raw hourly forecast
(`summary.forecast`, 15:00 on the day to 15:00 the next day, `weather.Span`) and the blanket code
summarises it again per horse. Older snapshots without `forecast` keep the default window for all
horses. Temperature and wind use the hourly
steps in the window; precipitation is per hour ending at the step and an hour counts with its
share inside the window (with the default window 12:00 to 13:00 counts half). Result: `night_min_c` (lowest temperature
of the window), `temp_max_c`, max rain probability, rain sum `rain_mm`, `rain_peak_mm` (most in
one hour), `rain_hours` (hours with at least 0.1 mm), `rain_from`/`rain_until`, max wind and an
hourly `timeline`. `will_rain = maxProb >= 50 % || sum >= 0.5 mm`
(`weather.RainProbabilityThreshold`, `weather.RainSumThresholdMM`). The details live in the
`raw` JSON of the snapshot, so no migration; older snapshots lack them.

**Job:** `weather.Service.Refresh` runs hourly (wired in `cmd/api`): per stable with lat/lng
it stores snapshots for today and tomorrow in `weather_snapshots` (append-only, `raw` holds
station and summary). `weather.Store.Latest(ctx, stableID, day)` returns the newest one.

**Recommendation** (`blanketplan`): `Recommend(rules, Forecast{NightMinC, WillRain, RainMM})` returns
the first matching rule by position (semantics in "Testing" below); `Changed(prev, next)`
tells whether the blanket differs (for change notifications). `LoadRules(ctx, pool,
stableID, horseID)` reads a horse's rules.

**Ground condition:** `stables.SetGroundCondition` / `GetGroundCondition` (`dry`, `wet`,
`frozen`, `muddy`); no endpoint yet.

## Realtime

Package `internal/realtime`: small "something changed" events for all members of a stable,
delivered as Server-Sent Events. Events are hints; the app reacts by refetching over the
normal API, so an event must never carry data some members may not see.

**Publish (any package, usually inside your transaction):**

```go
tx, _ := deps.Pool.Begin(ctx)
// ... change rows ...
err := realtime.Publish(ctx, tx, user.StableID, "request.changed", map[string]any{"id": id})
tx.Commit(ctx) // the event is sent on commit; a rollback sends nothing
```

`Publish(ctx, q, stableID, type, data)` runs `pg_notify('reiterhof_events', {stable_id, type, data})`;
`q` is a `*pgxpool.Pool`, `pgx.Tx` or `*pgx.Conn`. The whole payload must stay below 8 kB;
keep `data` to ids. Naming: `<domain>.changed` (`presence.changed`, `blanket_state.changed`,
`request.changed`, ...). The type `resync` is reserved (see below).

**Subscribe:** every process runs one `realtime.Hub` (started in `cmd/api`, exposed as
`deps.Events`, a no-op in tests unless you pass a hub: `hub := realtime.NewHub(pool, nil);
go hub.Run(ctx); <-hub.Ready()`). It holds one dedicated `LISTEN` connection and reconnects
with backoff (1 s to 30 s); after a reconnect it sends `resync` to all clients because events
may have been missed. Because Postgres fans out NOTIFY, several API processes work.

**Endpoint:** `GET /api/v1/events` (`auth.RequireStable`, filtered by the user's stable).
`text/event-stream`, per event `event: <type>` and `data: {"stable_id","type","data"}`, a
`: keep-alive` comment every 25 s, optional `?types=a,b` filter. Each client has a buffer of
32 events; a client that falls behind is dropped (the stream ends, the app reconnects and
refetches). No replay: clients refetch after (re)connecting.

**Authentication:** `Authorization: Bearer <token>` only; a token in the query string is not
accepted (it would end up in access logs and proxies). The clients use `expo/fetch` streaming
(native) and `fetch` (browser), which both send the header.

**Mobile:** `useStableEvents(types, handler)` and `useInvalidateOnEvents({ "request.changed":
[["requests"]] })` in `mobile/lib/realtime.ts`; one shared connection, closed in the background.

## Files

Package `internal/files` stores uploads on local disk (`REITERHOF_UPLOAD_DIR`, default `./uploads`, production
`/var/lib/reiterhof/uploads`) as `<stable_id>/<random>.<ext>`. Images are stored without metadata (EXIF position,
XMP, comments, trailers). Use it for every photo or document; details in
[domains/horses.md](domains/horses.md#files-internalfiles).

```go
// Save an upload (JPEG, PNG, WebP, PDF; max 20 MB; the content is sniffed, image metadata stripped) and store saved.Path in your table.
saved, err := files.Save(ctx, user.StableID, reader, declaredContentType) // errors: files.ErrTooLarge, files.ErrUnsupportedType, files.ErrCorrupt
files.Belongs(stableID, path)      // validate a path sent by a client before storing it
f, contentType, err := files.Open(stableID, path) // ErrInvalidPath (also for other stables), ErrNotFound
files.Remove(stableID, path)
files.Serve(w, r, deps, stableID, path) // after your own role check; wrap the route with files.DownloadLink(deps) for ?dl= links
```

HTTP: `POST /api/v1/files` (multipart, field `file`) returns `{path, url, content_type, size}`; `GET /api/v1/files/{path...}`
serves a file with the visibility of the record that references it (documents: owner, riders, admins; observation and
blanket photos: members; unreferenced: 404). `POST /api/v1/files/download-link {url}` mints a five-minute, path-bound
`?dl=` link for viewers that cannot send headers. In tests call `files.SetDir(t.TempDir())` and take valid images from
`internal/files/filestest`. The app uses `mobile/lib/upload.ts` (`uploadFile`, `fileSource`, `openStoredFile`).

## Domains

- [Presence](domains/presence.md): check-in/out, visibility, geofence.
- Requests (Anfragen, M4): [docs/domains/requests.md](domains/requests.md). Note for push: `new_request` is an opt-in kind (`push.OptIn`), it is only sent to users with an enabling `reminder_settings` row; all other kinds stay opt-out.
- [Horses, horse record, health, documents, files](domains/horses.md): `internal/horses`, `internal/health`, `internal/files`.
- [Observations (Auffälligkeiten)](domains/observations.md): report, urgent alert to everyone present, status, appointment bundling (`internal/observations`, `internal/health/bundle.go`).
- [Privacy, consents, export, deletion, retention](domains/privacy.md): `internal/privacy`, texts in `docs/legal/` (JAN-19).
- [Blankets](domains/blankets.md): blankets, rules, tonight's plan, day states, last-person and weather change reminders (`internal/blankets`, M3).
- [Training](domains/training.md): profile, "Was heute?", sessions, week view and exercise library (`internal/trainingapi`, M6).
- [Session tracking](domains/tracking.md): GPS rides, indoor sessions with gait detection, exercise library screens; `internal/trainingapi` stores the track and raw gait windows (M7).
- [Reha plan](domains/reha.md): phases, "Heute erlaubt", checkup reminders and the rule text of exercise requests (`internal/reha`, M7).
- [Web Push (PWA)](domains/push-web.md): VAPID, `web_push_subscriptions`, encrypted sender, service worker, iOS home screen requirement (`internal/push`, `internal/devices`, JAN-74).
- [Reminders and settings](domains/reminders.md): reminder center, notification switches per push kind, the stable's reminder time, evening training plan push, notification tap handling (`internal/reminders`, M8).
