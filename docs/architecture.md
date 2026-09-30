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

## Authentication (placeholder)

The auth agent adds middleware that authenticates the request and puts the current
user into the context. Domain handlers will use:

```go
type User struct {
    ID       string // UUID in text form (pgx reads/writes uuid columns as string)
    StableID string
    IsAdmin  bool
}

func auth.UserFrom(ctx context.Context) (User, bool)
```

Until it exists, do not invent your own identity mechanism; write handlers so the user
is obtained in one place at the top of each handler, and take `stable_id` from it.

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
