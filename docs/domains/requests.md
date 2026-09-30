# Requests (Anfragen, M4)

Help requests to the whole stable ("Turniertrottel", trailer seat, "Bewegen", ...).
Backend: `backend/internal/requests/`, migration `0050_requests.up.sql`. Mobile: tab
`app/(tabs)/requests.tsx`, screens `app/requests/{new,[id],calendar}.tsx`, helpers
`lib/requests*.ts`, `lib/api/requests.ts`, `components/request-*.tsx`.

Requests go to **everyone** in the stable (not only people who are present). Everything is
scoped by `stable_id`; all routes use `auth.RequireStable`. Other stables get `404`.

## Model

`requests` (see `0002_care.up.sql`) plus `request_assignees(request_id, user_id, thanked)`.
`0050` adds `requests.series_id` (unique with `date`) and a unique index on `reminders` for
`source_table = 'requests'`.

Status: `open` -> `assigned` (when helpers == `helpers_needed`) -> `done`; `cancelled` from
`open`/`assigned`. A withdrawal moves `assigned` back to `open`.

Request JSON (all responses): `id, type, horse_id, horse_name, horse_color_key, created_by,
creator_name, date (YYYY-MM-DD), date_end (inclusive or null), time_from, time_to (HH:MM),
location, description, tasks[], helpers_needed, status, recurring_rule, series_id,
remind_helper_at, payload, created_at, helpers[{user_id,name,avatar_color,thanked,joined_at}],
helpers_count, spots_left, is_creator, is_helper, can_accept`.

### Types and `payload`

Unknown fields are rejected. The server stores the canonical form.

| Type | Payload | Notes |
| --- | --- | --- |
| `show_helper` | `{show_name, classes:[{name,time?}], tasks[], ride_along}` | `tasks` subset of `hold_horse, warm_up, film, fetch_number, load_trailer`; copied to `tasks` |
| `ride_share` | `{destination, departure_time HH:MM, seats_free 1..8}` | `helpers_needed = seats_free`; `time_from` defaults to the departure |
| `exercise` | `{mode: lunge\|ride, rules_note?}` | horse required |
| `feed_or_turnout` | `{what: feed\|turnout\|bring_in}` | horse required; `date_end` allowed (range) |
| `appointment_companion` | `{with: farrier\|vet\|other, note?}` | horse required |
| `other` | `{}` | |
| `blanket` | any JSON object | generic, defined by the blanket feature |

Other types take a free-text checklist in `tasks` (max 20 entries of 80 characters).

## API

All under `/api/v1`, JSON, errors as in `docs/architecture.md`.

| Route | Purpose |
| --- | --- |
| `GET /requests?status=open,assigned&type=&mine=true&assigned=true&horse_id=&from=&to=&limit=` | List, `{requests, open_count}`. `mine` = created by me, `assigned` = I help, `from` compares with the last day, `to` with the first. Default status excludes `cancelled`. `open_count` = open, not expired, ignores filters. |
| `POST /requests` | Create (`201`). Body: `type, horse_id?, date, date_end?, time_from?, time_to?, location?, description?, tasks?, helpers_needed?, recurring_rule?, remind_helper_at?, payload`. Date must not be in the past. |
| `GET /requests/{id}` | Detail |
| `PATCH /requests/{id}` | Creator or admin, only while open/assigned. Optional `scope: "series"` (see below). `409 too_many_helpers` if fewer helpers than already joined. |
| `POST /requests/{id}/accept` | "Mach ich / Ich komme mit". Locks the row (`FOR UPDATE`). Idempotent. `403 own_request`, `409 request_full`, `409 not_open`, `409 expired`. |
| `POST /requests/{id}/withdraw` | Leave; idempotent; `409 not_open` once done/cancelled |
| `POST /requests/{id}/done` | Creator or helper; idempotent |
| `POST /requests/{id}/cancel` | Creator or admin; optional body `{scope: "one"\|"series"}`; idempotent |
| `POST /requests/{id}/thanks/{userId}` | Creator only, sets `thanked` on that helper (`204`, idempotent, no push) |
| `GET /me/thanks` | `{count}`: how often I was thanked. Private, no ranking |
| `GET /requests/calendar?from=&to=` | My accepted, upcoming requests (open/assigned), by date |
| `GET /requests/calendar.ics`, `GET /requests/{id}.ics` | RFC 5545, see below |
| `GET /requests/options` | `{horses:[{id,name,color_key}], members:[{id,name,avatar_color}]}` for the form (until a horses endpoint exists) |
| `GET/PATCH /requests/notify-settings` | `{new_request: bool}`, the push opt-in |

Accept rules: the creator cannot accept; at most `helpers_needed` helpers; two people racing for
the last seat are serialised by the row lock, the loser gets `409 request_full`.

## Push

Sent through `deps.Notify`:

- New request -> all other members, kind `new_request`. **Opt-in**: `push.OptIn(kind)` marks
  kinds that are only sent to users with a `reminder_settings` row `enabled = true`
  (all other kinds stay opt-out: no row means enabled). Users switch it with
  `PATCH /requests/notify-settings`. Materialized occurrences of a series do not push.
- Accepted / withdrawn -> creator, kind `helper`.
- Cancelled / date, time, place changed -> helpers, kind `helper`.
- Reminder -> helpers, kind `helper`, `data = {request_id, screen}`.

## Recurring requests

`recurring_rule` is a subset of RFC 5545 RRULE, single-day requests only:

```
FREQ=DAILY[;UNTIL=YYYYMMDD]
FREQ=WEEKLY[;BYDAY=MO,WE][;UNTIL=YYYYMMDD]   # without BYDAY: weekday of `date`
```

`INTERVAL`, `COUNT`, other frequencies are rejected. The first request is the template
(`series_id = id`). The job `request-recurring` (daily 03:30 Europe/Berlin, and at start)
creates the occurrences for today..today+14 days (`ON CONFLICT (series_id, date) DO NOTHING`,
so it is idempotent). Occurrences copy the template and start `open`.

- `PATCH {scope:"series"}`: `time_from, time_to, location, description, tasks, helpers_needed,
  payload, recurring_rule` apply to the template and all upcoming open occurrences (an
  occurrence keeps more helpers than the new count). Changing the rule deletes upcoming
  occurrences nobody signed up for; the job recreates them. `recurring_rule: ""` ends the series.
- `POST .../cancel {scope:"series"}`: cancels this and all upcoming occurrences and ends the series.

## Helper reminder

`remind_helper_at` is set by the creator or, on the first accept, defaults to the day before
at 18:00 in the stable's time zone (only if that lies in the future). The job
`request-helper-reminders` (every 5 minutes) sends a push with date, place and checklist to
each helper of an open/assigned request whose time has passed. A row in `reminders`
(`source_table = 'requests'`, unique per request and user) marks it as sent. Someone who accepts
after the time has passed gets no reminder; changing date or reminder time re-arms it.

## Calendar (ICS)

`text/calendar`, CRLF, lines folded at 75 octets, TEXT escaped (`\\ \; \, \n`), one `VEVENT`
per request with `UID <id>@reiterhof.app`. Times are `DTSTART;TZID=Europe/Berlin` (with an
embedded `VTIMEZONE`); requests without a start time are all-day events (`DTEND` exclusive).
The mobile app adds events with the system calendar screen (`expo-calendar`, no permission
needed) and falls back to the share sheet. The ICS URLs need the bearer token, so they cannot be
subscribed to from other calendar apps yet.

## Realtime extension point

`requests.DefaultPublish` (package variable) or `Service.Publish` receives an `Event{StableID,
RequestID, Kind}` after each committed change (`created, updated, accepted, withdrawn, done,
cancelled`). Nil is a no-op; the realtime hub can set it in `cmd/api` before the router is built.
