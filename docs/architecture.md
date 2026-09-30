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
