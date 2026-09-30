# Presence (Anwesenheit)

Who is at the stable, JAN-24 (check-in/out), JAN-25 (visibility), JAN-26 (screen), JAN-27
(geofence). Backend: `backend/internal/presence`, table `presence` (core schema, no extra
migration). Mobile: `app/presence/index.tsx`, `lib/api/presence.ts`, `lib/geofence*.ts`,
`components/presence-tile.tsx`.

## Data

A visit is one row of `presence(stable_id, user_id, arrived_at, left_at, source)`. `left_at
IS NULL` means open. A unique partial index allows one open visit per user. `source` is
`manual` or `geofence`.

## API (`/api/v1`, all `auth.RequireStable`, always filtered by the user's stable)

| Route | Body | Response |
| --- | --- | --- |
| `POST /presence/check-in` | optional `{source: "manual"\|"geofence"}` (default `manual`) | `200 {visit}`. **Idempotent**: with an open visit that visit is returned unchanged (source is not overwritten). |
| `POST /presence/check-out` | none | `200 {visit}` with `left_at` set, or `{visit: null}` when nothing was open (no-op). |
| `GET /presence` | none | `{me, here, recent}` below. |

The visibility setting itself is `PATCH /api/v1/me {presence_visibility}` (`all`, `only_day`,
`hidden`). Check-in and check-out publish `presence.changed` (no data) on the realtime
stream, see [Realtime](../architecture.md#realtime); the app refetches `GET /presence`.

```json
{
  "me": {
    "open_visit": {"id": "...", "arrived_at": "2026-09-30T16:40:00Z", "left_at": null, "source": "manual"},
    "last_visit": null,
    "visibility": "all"
  },
  "here":   [{"user_id": "...", "name": "Tom", "avatar_color": "teal", "since": "2026-09-30T16:10:00Z"}],
  "recent": [{"user_id": "...", "name": "Kai", "avatar_color": "blue", "last_seen_date": "2026-09-29",
              "last_seen_at": "2026-09-29T17:20:00Z", "today": false, "usual_arrival_hour": 19}]
}
```

- `me` is the caller's own data, always complete. The caller is not part of `here`/`recent`.
- `here`: other people with an open visit, sorted by name (not by arrival, so the order leaks
  nothing).
- `recent`: per other person the end of their last visit within 60 days (max 50, newest
  first), only people who are not `here`. `last_seen_date` is the stable-local calendar day
  (`stables.timezone`), `today` compares it with the stable-local today.

## Visibility (enforced server-side, JAN-25)

| Level | Others see |
| --- | --- |
| `all` | everything: `since`, `last_seen_at`, the usual-arrival hint |
| `only_day` | that the person is there today (`here` with `since: null`) and the date of the last visit (`last_seen_date`, `today`), but **no times** (`last_seen_at: null`) and no hint |
| `hidden` | nothing: not in `here`, not in `recent` (filtered in SQL) |

The person always sees their own full data. **Admins get no exception.** Realtime events carry
no presence data, so they cannot leak anything.

## Usual arrival hint

`usual_arrival_hour` (0-23) is the median arrival time, in the stable's time zone, of a
person's visits in the last 8 weeks, rounded to the hour ("kommt meist gegen 19 Uhr"). Only
set when the person has at least 4 visits in that window and visibility `all`, and only in
`recent`.

## Check-out hook

`presence.AfterCheckOut` (package variable, nil = no-op) is called by `POST /presence/check-out`
after a visit was closed (not by the stale job). `cmd/api` sets it to the blanket reminder for the
last person leaving, see [blankets](blankets.md#check-out-hook).

## Stale visits

People forget to check out, and the phone can miss a geofence exit. The scheduler job
`presence-close-stale` (every 15 minutes and at start) closes every visit that has been open
for **more than 12 hours** (`presence.StaleAfter`). The visit ends at `arrived_at + 12 h`, not
at the time of cleanup, so "zuletzt gesehen" is not inflated by hours nobody witnessed. It
publishes `presence.changed` for the affected stables.

## Geofence (JAN-27)

- `GET /api/v1/me` returns `stable.lat`, `stable.lng` and `stable.geofence_radius_m` (default
  150, clamped to 100 to 2000 m on the device because smaller regions are unreliable).
- Opt-in **per device**: the switch state lives in the device's secure store, not on the
  server. Enabling asks for foreground and then background ("Immer") location permission
  with a German explanation and starts `Location.startGeofencingAsync` (task
  `reiterhof-geofence`, defined at app start by importing `lib/geofence.ts`).
- Enter calls `check-in` with `source: "geofence"`, exit calls `check-out`. The task reads
  the session token itself, because it may run in a fresh JS context; failures are silent.
- The location never leaves the device; the server only learns "arrived" / "left".
- No-op on web and wherever the task manager is unavailable (the switch is hidden).
- The privacy consent (JAN-19) is not built yet; the screen says so. Add the consent before
  the switch can be turned on when it exists.
- Requires a development build or a standalone build; Expo Go cannot run background location.

## Tests

`go test ./internal/presence ./internal/realtime` (needs `REITERHOF_TEST_DATABASE_URL`):
idempotent check-in/out, each visibility level (including admin and own data), cross-stable
isolation, usual-arrival rules, stale closing, SSE delivery, event isolation per stable,
rollback sends nothing, slow client dropped. Mobile: `lib/presence-format.test.ts`,
`lib/geofence-core.test.ts`, `lib/sse.test.ts`, `lib/backoff.test.ts`.
