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
  migrations/         NNNN_description.up.sql, embedded via go:embed (migrations.FS)
  internal/
    config/           environment configuration
    db/               pgx pool (db.Open) and migrator (db.Migrate)
    dbtest/           throwaway databases for tests
    seed/             example data + exported fixed IDs (seed.HorseLuna, ...)
    scheduler/        reusable periodic / daily jobs (see "Scheduler")
    weather/          DWD MOSMIX client, night summary, weather_snapshots, hourly job
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
| `REITERHOF_EXPO_ACCESS_TOKEN` | unset | optional Expo access token for the push API (`push.NewClientFromEnv`) |
| `REITERHOF_PUBLIC_URL`, `REITERHOF_SMTP_*`, `REITERHOF_*_CLIENT_IDS`, `REITERHOF_DEV_LOGIN` | unset | authentication, see [Authentication and roles](#authentication-and-roles) |

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
  (JAN-23), `0010` auth tables (auth agent). Unapplied lower numbers still apply
  after higher ones, so parallel branches do not block each other. Two files with the
  same number make the migrator fail, which surfaces a clash at merge time.
- Schema conventions: UUID primary keys (`gen_random_uuid()`), `created_at timestamptz`,
  `stable_id uuid NOT NULL REFERENCES stables(id)` on every domain table, enums as
  `text` + `CHECK`, an index starting with `stable_id` for list queries.

## Authentication and roles

Passwordless, no SMS, no external auth service: sessions live in our Postgres
(`backend/internal/auth`, migration `0010_auth.up.sql`).

Sign-in methods (all return `{"token": "...", "user": {...}}`; the token is an opaque
bearer token, only its SHA-256 hash is stored, sessions last 90 days from the last use):

| Route (all `POST`, no auth) | Body | Notes |
| --- | --- | --- |
| `/api/v1/auth/magic-link` | `{email}` | Always `204` (no user enumeration), `429` when rate limited (3 per email / 15 min, 20 per IP / h). Mail contains `reiterhof://auth/verify?token=...` and, if `REITERHOF_PUBLIC_URL` is set, an https fallback `<url>/auth/verify?token=...` (a page that only links to the app, it never consumes the token). Tokens live 15 min and work once. |
| `/api/v1/auth/verify` | `{token}` | Creates the user if the email is new. `401 invalid_token` if unknown, used or expired. |
| `/api/v1/auth/google` | `{id_token}` | RS256 ID token, checked against Google's JWKS (cached), `iss`, `aud` (`REITERHOF_GOOGLE_CLIENT_IDS`), `exp`. |
| `/api/v1/auth/apple` | `{id_token, name?}` | Same for Apple (`REITERHOF_APPLE_CLIENT_IDS`); Apple sends the name only to the client on first sign-in, so the app passes it along. |
| `/api/v1/auth/dev-login` | `{email}` | `404` unless `REITERHOF_DEV_LOGIN=true`. Signs in (or creates) any email, for local testing with seed users (`jan@example.org` is admin). Never enable in production. |

Identity of Google/Apple users is the provider `sub` (`auth_identities`, unique per
provider and subject). A new identity is attached to the account with the same *verified*
email (merge); Apple relay addresses are ordinary emails and so match nothing. Tokens
without verified email are rejected (`401 email_not_verified`).

Authenticated routes: `POST /api/v1/auth/logout` (`204`), `GET /api/v1/me` (user, stable
or `null`, `roles.owned_horse_ids`, `roles.rider_horse_ids`), `PATCH /api/v1/me`
(`name`, `phone`, `avatar_color`, `presence_visibility`; `""` clears phone/avatar_color),
`POST /api/v1/stables/join` (`{code}`, sets `users.stable_id`; `404 invalid_code`,
`409 already_in_stable`), `POST /api/v1/stables/invites` (admin only, optional
`{expires_in_days (1..90, default 7), max_uses (1..100, default 10)}`, returns
`{code: "ABCD-EFGH", expires_at, max_uses}`). Codes are case- and separator-insensitive.

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
| `REITERHOF_PUBLIC_URL` | public base URL of the API, e.g. `https://api.example.org`, for the mail's https fallback link |
| `REITERHOF_SMTP_HOST`, `_PORT` (587), `_USER`, `_PASSWORD`, `_FROM` | SMTP for login mails (STARTTLS, port 465 = implicit TLS). Without host the mail is only logged (dev) |
| `REITERHOF_GOOGLE_CLIENT_IDS` | comma separated OAuth client IDs (web, iOS, Android) accepted as `aud` |
| `REITERHOF_APPLE_CLIENT_IDS` | comma separated accepted `aud` (iOS bundle ID `org.datenlotse.reiterhof`) |
| `REITERHOF_DEV_LOGIN` | `true` enables `/auth/dev-login` |

Rate limits are in memory (per process, reset on restart), fine for the single-VPS setup.
The rate limiter uses `X-Forwarded-For` only when the connection comes from loopback (Caddy).

## Push

Package `internal/push` (backend part of JAN-18). Expo push tokens live in
`push_tokens` (unique per token), per-user opt-outs in `reminder_settings`.

**In handlers, send pushes via `deps.Notify.NotifyUsers(ctx, stableID, userIDs, kind,
title, body, data)`** (`httpx.Notifier`; never nil, a no-op unless `cmd/api` wires the
real `*push.Notifier`). In tests pass `Notify: push.NewNotifier(pool, fake, nil)` with
`fake := &push.Fake{}` and assert on `fake.Sent()`. Devices register their token via
`POST /api/v1/me/push-tokens {token, platform}` / `DELETE` (package `internal/devices`).

- `push.Sender` (`Send(ctx, []Message) error`) is the seam. `push.Client` talks to
  `https://exp.host/--/api/v2/push/send` in batches of 100 (base URL, `*http.Client`
  and access token are fields, so tests use `httptest`). `push.Fake` records messages
  for tests of other packages.
- Problems are returned as `*push.SendError`: `InvalidTokens` (Expo ticket
  `DeviceNotRegistered`) and `Failures` (other ticket or batch errors). A failing
  batch does not stop the following ones. `Client.OnInvalidToken` is an optional
  extra callback.
- `push.NewNotifier(pool, sender, log).NotifyUsers(ctx, stableID, userIDs, kind, title,
  body, data)` loads the tokens of those users **within that stable**, skips users
  whose `reminder_settings` row for `kind` has `enabled = false` (no row = enabled),
  adds `data.kind`, sends, and deletes the invalid tokens (by `stable_id` + token).
  Only non-token failures are returned as error.
- Kinds are constants (`push.KindLastPerson`, `KindWeatherChange`, `KindMedication`,
  `KindHelper`, `KindTrainingPlan`, `KindHealthDue`, `KindRehaCheckup`,
  `KindNewRequest`, `KindUrgentObservation`, `KindObservation`; `push.Kinds()`, `push.ValidKind`).
- Store: `push.RegisterToken(ctx, pool, stableID, userID, token, platform)` upserts
  by token (a device handed to another user is re-assigned; the user must belong to
  the stable, else `ErrUnknownUser`); `push.DeleteToken(ctx, pool, stableID, userID,
  token)`.
- Not wired yet: `POST /api/v1/me/push-tokens` (and `DELETE`) follow once
  `auth.UserFrom` exists; the handler is a thin wrapper around `RegisterToken`.
  Wiring `NewNotifier` into the API is also left to the features that send reminders.
- Mobile: `mobile/lib/push.ts` `registerForPush()` asks for permission and returns
  `{ token, platform }` (needs `expo.extra.eas.projectId` in `app.json` and a real
  device; it does not throw). Sending the token to the backend comes with auth.

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
- Blanket rules semantics (seed and evaluation): rules are evaluated by `position`,
  the first match wins; `temp_min <= temp < temp_max`, `NULL` bound = open, `NULL` rain
  = any; `blanket_id NULL` = no blanket.

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

**Night summary** (`weather.Summarize`): window 18:00 stable-local on the day until 08:00
the next day (wall-clock, so DST nights are 15 h / 13 h). Result: `night_min_c`, max rain
probability, rain sum, max wind. `will_rain = maxProb >= 50 % || sum >= 0.5 mm`
(`weather.RainProbabilityThreshold`, `weather.RainSumThresholdMM`).

**Job:** `weather.Service.Refresh` runs hourly (wired in `cmd/api`): per stable with lat/lng
it stores snapshots for today and tomorrow in `weather_snapshots` (append-only, `raw` holds
station and summary). `weather.Store.Latest(ctx, stableID, day)` returns the newest one.

**Recommendation** (`blanketplan`): `Recommend(rules, Forecast{NightMinC, WillRain})` returns
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

**Authentication:** `Authorization: Bearer <token>`; additionally `?access_token=<token>` for
EventSource libraries in React Native that cannot set headers. Tradeoff: a token in the URL
ends up in access logs and proxies, and it is the same long-lived session token. The query
form is accepted for this path only (`auth.BearerToken`); configure the reverse proxy not to
log the query string of `/api/v1/events`. The mobile app uses `expo/fetch` streaming with the
header and does not need it.

**Mobile:** `useStableEvents(types, handler)` and `useInvalidateOnEvents({ "request.changed":
[["requests"]] })` in `mobile/lib/realtime.ts`; one shared connection, closed in the background.

## Files

Package `internal/files` stores uploads on local disk (`REITERHOF_UPLOAD_DIR`, default `./uploads`, production
`/var/lib/reiterhof/uploads`) as `<stable_id>/<random>.<ext>`. Use it for every photo or document; details in
[domains/horses.md](domains/horses.md#files-internalfiles).

```go
// Save an upload (JPEG, PNG, WebP, HEIC, PDF; max 20 MB; the content is sniffed) and store saved.Path in your table.
saved, err := files.Save(ctx, user.StableID, reader, declaredContentType) // errors: files.ErrTooLarge, files.ErrUnsupportedType
files.Belongs(stableID, path)      // validate a path sent by a client before storing it
f, contentType, err := files.Open(stableID, path) // ErrInvalidPath (also for other stables), ErrNotFound
files.Remove(stableID, path)
files.Serve(w, r, deps, stableID, path) // after your own role check; wrap the route with files.QueryToken(deps) for ?access_token=
```

HTTP: `POST /api/v1/files` (multipart, field `file`) returns `{path, url, content_type, size}`; `GET /api/v1/files/{path...}`
serves files to members of the same stable only (`?access_token=` accepted, see the tradeoff in the domain doc).
In tests call `files.SetDir(t.TempDir())`. The app uses `mobile/lib/upload.ts` (`uploadFile`, `fileSource`).

## Domains

- [Presence](domains/presence.md): check-in/out, visibility, geofence.
- Requests (Anfragen, M4): [docs/domains/requests.md](domains/requests.md). Note for push: `new_request` is an opt-in kind (`push.OptIn`), it is only sent to users with an enabling `reminder_settings` row; all other kinds stay opt-out.
- [Horses, horse record, health, documents, files](domains/horses.md): `internal/horses`, `internal/health`, `internal/files`.
- [Observations (Auffälligkeiten)](domains/observations.md): report, urgent alert to everyone present, status, appointment bundling (`internal/observations`, `internal/health/bundle.go`).
- [Privacy, imprint, consents, export, deletion, retention](domains/privacy.md): `internal/privacy`, texts in `docs/legal/` (JAN-19).
- [Training](domains/training.md): profile, "Was heute?", sessions, week view and exercise library (`internal/trainingapi`, M6).
- [Session tracking](domains/tracking.md): GPS rides, indoor sessions with gait detection, exercise library screens; `internal/trainingapi` stores the track and raw gait windows (M7).
- [Reha plan](domains/reha.md): phases, "Heute erlaubt", checkup reminders and the rule text of exercise requests (`internal/reha`, M7).
