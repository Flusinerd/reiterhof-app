# Auffälligkeiten (Observations)

Tickets: JAN-50 (melden), JAN-51 (dringend), JAN-52 (Liste je Pferd, Status), JAN-54 (Termine bündeln).
Code: `backend/internal/observations`, `backend/internal/health/bundle.go`, table `observations` (core schema,
no extra migration), seed `internal/seed/observations.go`. App: `mobile/app/observations/`,
`mobile/components/horse-observations.tsx`, `observation-*.tsx`, `mobile/lib/api/observations.ts`,
`mobile/lib/observations.ts`.

All routes need a signed-in user with a stable (`auth.RequireStable`). A horse or observation of another stable
answers `404 not_found`, invalid input `400 validation_failed` (see "Error format" in `architecture.md`).

## Permissions

| Action | Who |
| --- | --- |
| report an observation for any horse of the stable | every member |
| list / read observations | every member |
| change the status (`watch` / `done`) | owner of the horse, admin, or the reporter (`403 forbidden` otherwise) |

Decision: reporting is a member right ("darf melden"), not a rider right. The rider rule `report_observations`
(`docs/domains/horses.md`) is kept in the rule list and in the app but is **not enforced** here: a stranger who sees
a lame horse must be able to say so. Observations cannot be edited or deleted after sending (only the status
changes); a wrong report is set to `done`.

## API (`/api/v1`)

| Route | Notes |
| --- | --- |
| `POST /observations` | body below, `201 {observation, emergency}` |
| `GET /horses/{id}/observations` | array, newest first; `?status=watch\|done`, `?limit=` (1..100, default 50) |
| `GET /observations/{id}` | one observation |
| `PATCH /observations/{id}` | `{status: "watch"\|"done"}` |

Report body (unknown fields are rejected):

| Field | Rule |
| --- | --- |
| `horse_id` | required, a horse of the caller's stable |
| `category` | required: `cough`, `lameness`, `injury`, `not_eating`, `colic`, `behavior`, `blanket_equipment`, `other` |
| `body_part` | optional: `front_left`, `front_right`, `hind_left`, `hind_right`, `head`, `back`, `belly`, `other` |
| `description` | optional, trimmed, at most 2000 characters |
| `media` | optional list of at most 6 photo paths from `POST /files` (`files.Belongs` for the caller's stable; PDFs and duplicates are rejected) |
| `urgency` | `info` (default), `check` ("Bitte ansehen"), `urgent` |

Observation shape: `id, horse_id, horse_name, reporter{id,name,color_key}, category, body_part, description,
media[{path,url,content_type}], urgency, status, created_at, can_change`. New reports start with status `watch`
("Beobachten"); `done` is "Erledigt". `media[].url` is the files API route (send the session token, e.g.
`fileSource()` in the app).

Photos only: `internal/files` accepts JPEG, PNG, WebP, HEIC and PDF. Videos are not supported (they would need
larger limits and streaming, and the Caddy body limit is 25 MB); extending the type table in `files` is the
place to add mp4/mov later. The app offers camera and photo library for images only.

Realtime: every create and every status change publishes `observation.changed` with `{id, horse_id}` (ids only,
see [Realtime](../architecture.md#realtime)); the app invalidates its `observations` queries
(`useObservationEvents`).

## Push (JAN-50, JAN-51)

Sent after the report is saved (a push failure is logged, never fails the request), never to the reporter:

| Urgency | Kind | Recipients |
| --- | --- | --- |
| `info`, `check` | `observation` (new kind, on by default, can be switched off) | owner and riders of the horse |
| `urgent` | `urgent_observation` | owner and riders of the horse **and everybody with an open presence visit** |

Texts (German): title "Dringend: Luna" / "Auffälligkeit: Luna", body "Mia meldet: Kolik. Bitte sofort nach dem Pferd
sehen." (`check` adds "Bitte ansehen."). Data: `observation_id, horse_id, urgency` (plus `kind`). The description
is not part of the push (lock screen). Push only: no SMS, no call.

**Why the presence visibility setting does not apply to the alert.** `presence_visibility` (`hidden`, `only_day`)
controls what *others can see about me*. The alert only *uses* the open visits on the server to decide who to wake up,
and it reveals nothing: the response of `POST /observations` contains no recipient list or count, the message is the
same for everybody, and the reporter cannot tell who was reached. Someone at the stable who has hidden their presence
is exactly the person who can help in the first minutes, so excluding them would defeat the purpose. Users who do not
want this can switch off the kind `urgent_observation` in their reminder settings (`push.Notifier` honors it; that
is the user's choice about their own device, not a server rule). Presence of other stables is never used.

## Emergency card in the response (JAN-51)

For `urgency: "urgent"` the response has `emergency` = the emergency card of the horse (same shape as
`GET /horses/{id}/emergency`, see `horses.LoadEmergencyCard`), otherwise `null`. The app puts it into the query
cache and replaces the report screen with `/horses/{id}/emergency`, so the card is on screen immediately, even
with a slow connection. The report screen asks for confirmation before an urgent report is sent.

## Bundling appointments (JAN-54)

`GET /horses/{id}/health` (package `health`) adds `bundle` to an item (absent otherwise):

```json
{"kind": "vet", "count": 3, "horses": [{"id": "...", "name": "Cookie"}, {"id": "...", "name": "Fanta"}, {"id": "...", "name": "Nala"}]}
```

- Only items of the kinds `vet`, `vaccination`, `dentist` with a due date **today up to 14 days ahead** (overdue
  items are not "approaching"; medication courses with `daily_time` never bundle).
- `horses` are the *other* horses of the stable with an item of the **same kind** due within **+-14 days of this
  item's due date** (an overdue one counts, too). Every horse is counted once. Sorted by name, `count = len(horses)`.
- Same information in `summary`, the four tiles. Other stables are never looked at.
- App: the health screen shows "3 weitere Pferde sind ebenfalls fällig – Termin bündeln?" and the names under the
  item (`bundleText`, `horseNamesText` in `lib/observations.ts`).

## App

- Horse record: `components/horse-observations.tsx` lists the newest 5 observations with badges "Beobachten" /
  "Erledigt" (and the urgency if not info), button **Melden** (`/observations/new?horse=<id>`). Prop
  `renderActions(observation)` is the slot for extra buttons per observation, e.g. "In Reha-Plan umwandeln" of the reha
  feature (rendered inside `ObservationCard`, taps do not open the detail screen).
- `/observations/new?horse=<id>`: horse chips, category chips, horse pictogram (`react-native-svg`, tappable
  regions plus chips as accessible alternative), description, photos (camera / library, upload happens on send and is
  not repeated on retry), urgency "Info" / "Bitte ansehen" / "Dringend". After an urgent report the emergency card opens.
- `/observations/[id]`: details, photos, status button (only if `can_change`), shortcuts to the emergency card and
  the horse record.

## Seed

`ObservationFanta` (Lea, lameness front right, `check`, `watch`, 12 days ago; the reha plan of Fanta refers to it),
`ObservationLunaCough`, `ObservationBaluDone`, `ObservationLunaOldEat`.

## Tests

`go test ./internal/observations ./internal/health` (needs `REITERHOF_TEST_DATABASE_URL`): any member may report,
validation of every field (including foreign-stable and PDF media), cross-stable isolation on all routes, recipients
for info/check/urgent with the fake sender (reporter excluded, owner and rider, present people incl. hidden ones,
left visits, opted-out users, no duplicates, other stables), emergency card only for urgent, list order, filters and
`can_change`, status permissions (reporter, owner, admin, stranger), realtime events, bundling counts and window
edges. Mobile: `lib/observations.test.ts`.
