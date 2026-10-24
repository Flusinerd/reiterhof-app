# Requests (Anfragen, M4)

Help requests to the whole stable ("Turniertrottel", trailer seat, "Bewegen", ...).
Backend: `backend/internal/requests/`, migration `0050_requests.up.sql`. Mobile: screens
`app/requests/{index,new,[id],calendar}.tsx` (see "Mobile" below), helpers `lib/requests*.ts`,
`lib/stable-tab.ts`, `lib/api/requests.ts`, `components/request-*.tsx`, `components/start-requests.tsx`.

Requests go to **everyone** in the stable (not only people who are present). Everything is
scoped by `stable_id`; all routes use `auth.RequireStable`. Other stables get `404`.

## Model

`requests` (see `0002_care.up.sql`) plus `request_assignees(request_id, user_id, thanked)`.
`0050` adds `requests.series_id` (unique with `date`) and a unique index on `reminders` for
`source_table = 'requests'`.

Status: `open` -> `assigned` (when helpers == `helpers_needed`) -> `done`; `cancelled` from
`open`/`assigned`. A withdrawal moves `assigned` back to `open`.

Request JSON (all responses): `id, type, horse_id, horse_name, horse_color_key, created_by,
creator_name, creator_color (avatar color key of the creator, may be null), date (YYYY-MM-DD), date_end (inclusive or null), time_from, time_to (HH:MM),
location, description, tasks[], helpers_needed, status, recurring_rule, series_id,
payload, created_at, helpers[{user_id,name,avatar_color,thanked,joined_at}],
helpers_count, spots_left, is_creator, is_helper, can_accept, my_remind_at` (the viewer's own
reminder as helper; other helpers' reminders are never exposed).

### Types and `payload`

Unknown fields are rejected. The server stores the canonical form.

| Type | Payload | Notes |
| --- | --- | --- |
| `show_helper` | `{show_name, classes:[{name,time?}], tasks[]}` | `tasks` subset of `hold_horse, warm_up, film, fetch_number, load_trailer`; copied to `tasks` |
| `ride_share` | `{destination, departure_time HH:MM, seats_free 1..8}` | `helpers_needed = seats_free`; `time_from` defaults to the departure |
| `exercise` | `{mode: lunge\|ride, rules_note?}` | horse required; `rules_note` (max 800) gets the horse's reha rule as first line, filled by the server, see [reha.md](reha.md) |
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
| `POST /requests` | Create (`201`). Body: `type, horse_id?, date, date_end?, time_from?, time_to?, location?, description?, tasks?, helpers_needed?, recurring_rule?, payload`. Date must not be in the past. |
| `GET /requests/{id}` | Detail |
| `PATCH /requests/{id}` | Creator or admin, only while open/assigned. Optional `scope: "series"` (see below). `409 too_many_helpers` if fewer helpers than already joined. |
| `POST /requests/{id}/accept` | "Mach ich / Ich komme mit". Optional body `{remind_at}` (RFC 3339, future; default see Helper reminder). Locks the row (`FOR UPDATE`). Idempotent. `403 own_request`, `409 request_full`, `409 not_open`, `409 expired`. |
| `PUT /requests/{id}/reminder` | Helper only: `{remind_at: RFC 3339 \| null}` sets or switches off my own reminder (`400` if in the past, `403 not_helper`, `409 not_open`) |
| `POST /requests/{id}/withdraw` | Leave; idempotent; `409 not_open` once done/cancelled |
| `POST /requests/{id}/done` | Creator or helper; idempotent |
| `POST /requests/{id}/cancel` | Creator or admin; optional body `{scope: "one"\|"series"}`; idempotent |
| `POST /requests/{id}/thanks/{userId}` | Creator only, sets `thanked` on that helper (`204`, idempotent, no push) |
| `GET /me/thanks` | `{count}`: how often I was thanked. Private, no ranking |
| `GET /requests/calendar?from=&to=` | My accepted, upcoming requests (open/assigned), by date |
| `GET /requests/calendar.ics`, `GET /requests/{id}.ics` | RFC 5545, see below |
| `GET /requests/options` | `{horses:[{id,name,color_key}], members:[{id,name,avatar_color}]}` for the form (until a horses endpoint exists) |
| `GET/PATCH /requests/notify-settings` | `{new_request: bool}`, the push opt-in (same row as `PUT /settings/notifications/new_request`, see [reminders](reminders.md); the app uses the central settings) |

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

Each helper chooses their own reminder (`request_assignees.remind_at`); the creator does not
set one. On accept it defaults to the day before at 18:00 in the stable's time zone (only if
that lies in the future), or the app sends `remind_at`. The helper changes it in the detail
screen ("Meine Erinnerung": Vortag 18:00, Am Morgen 07:00, 2 Stunden vorher, Keine) via
`PUT /requests/{id}/reminder`. The job `request-helper-reminders` (every 5 minutes) sends a push
with date, place and checklist to each helper of an open/assigned request whose own time has
passed. A row in `reminders` (`source_table = 'requests'`, unique per request and user) marks it
as sent; changing the reminder re-arms it, as does withdrawing. Changing the request's date resets
all helpers to the default for the new day.

## Calendar (ICS)

`text/calendar`, CRLF, lines folded at 75 octets, TEXT escaped (`\\ \; \, \n`), one `VEVENT`
per request with `UID <id>@reiterhof.app`. Times are `DTSTART;TZID=Europe/Berlin` (with an
embedded `VTIMEZONE`); requests without a start time are all-day events (`DTEND` exclusive).
The mobile app adds events with the system calendar screen (`expo-calendar`, no permission
needed) and falls back to the share sheet. The ICS URLs need the bearer token, so they cannot be
subscribed to from other calendar apps yet.

## Mobile

Requests are shown as messages, never as cards ([design system](../design-system.md)): `components/request-row.tsx`
(`RequestRow`) has the avatar of who asked (`creator_color`), the first name or "Du" with the time in brass
("Anna · morgen 07:00"), the text (the description, else the title; ride shares "Mitfahrgelegenheit nach Haltern"),
a quiet line with the type, place and repeat, and the answer: the outlined accept button of the type ("Mach ich",
"Ich komme mit") and who helps ("Noch niemand", "Jonas hilft", "Du und Jonas helfen · 2 von 3 Plätzen"). The texts
are pure helpers in `lib/stable-tab.ts` (`requestRow`, `requestWhen`, `helpersLine`, `groupRequestsByDay`).

- `/requests`: header with "Neu" (the page's one primary button) and the calendar icon, the filter pills (Offen, Meine, Ich
  helfe, Erledigt), the requests grouped by day with a heading ("Morgen", "Montag, 26. Okt") and hairlines. Done and
  cancelled ones fade. `/requests/calendar` uses the same rows.
- `/requests/[id]`: one primary action by role (accept for everybody who can, else "Erledigt" for the creator and the
  helpers), the description as an italic quote, the facts as rows (Wo, Pferd, Ziel, Wiederholung ...), "Meine Erinnerung" pills
  for helpers, the checklist and the helpers as rows ("Danke" for the creator), and the quiet actions as rows
  (Abmelden, In Kalender, Anfrage absagen in red, which asks in a sheet).
- `/requests/new`: the type as icon pills with its hint, the fields with upper-case labels (`FieldRow`), one primary
  button "Senden".
- The tab "Stall" shows the three newest open requests with the same rows and renders nothing when there are none.

## Blanket requests close automatically

When someone records the blanket state of a horse ([blankets](blankets.md#requests-close-automatically-jan-39)),
`requests.CompleteBlanketRequests` marks the open or assigned `blanket` requests of that horse whose
date range covers the blanket night as `done` and adds `payload.feedback` (German text with the
blanket and the person). Other types, horses and days are not touched.

## Realtime extension point

`requests.DefaultPublish` (package variable) or `Service.Publish` receives an `Event{StableID,
RequestID, Kind}` after each committed change (`created, updated, accepted, withdrawn, done,
cancelled`). Nil is a no-op; the realtime hub can set it in `cmd/api` before the router is built.
