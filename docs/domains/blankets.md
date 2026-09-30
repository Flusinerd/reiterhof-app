# Blankets (Decken und Wetter, M3)

Who blankets which horse tonight, and the reminder for the last person. Tickets JAN-29
(blankets), JAN-30 (plan and rules), JAN-31 (day state), JAN-32 (tab "Decken"), JAN-33
("Letzte Person"), JAN-34 (weather change), JAN-35 (start screen), JAN-39 (requests close).

Backend: `backend/internal/blankets/` (API, store, jobs), pure rule evaluation in
`backend/internal/blanketplan`, forecast from `backend/internal/weather`, migration
`0070_blankets.up.sql` (unique index for reminder claims). Mobile: tab
`app/(tabs)/blankets.tsx`, start screen `app/(tabs)/index.tsx`, plan
`app/horses/[id]/blanket-plan.tsx`, history `app/blankets/history/[horseId].tsx`,
`lib/api/blankets.ts`, `lib/blankets*.ts`, `components/blanket-*.tsx`, `components/start-*.tsx`.

## Model

Tables (core schema): `blankets` (per horse: `name, fill_g, color, location, photo_path`),
`blanket_rules` (per horse, ordered by `position`, unique per horse), `blanket_states`
(history: every state change is a row; `automatic` marks rows of the auto-uncover job), `weather_snapshots`, `stables.reminder_time`. The
helper note is `horses.helper_note`; it is changed through `PATCH /horses/{id}` (horses API),
this domain only returns it.

Roles: everyone in the stable reads and sets day states. Blankets and rules can be changed by
the owner and admins (`auth.CanManageHorse`); riders and other members get `403 forbidden`.
A horse of another stable is `404 not_found`.

### The blanket day

A "day" is the night that starts on a stable-local date. The night belongs to that date until
**04:00 the next morning** (`blankets.RolloverHour`, `blankets.NightDay`): at 00:30 on
1 October the day is still 30 September (late checks count for the running night); from
04:00 on the app shows the coming night of 1 October (forecast, recommendation, progress),
because the past night is settled by then. One exception (`blankets.StateDay`):
"uncovered" recorded before **15:00** (`blankets.UncoverUntilHour`) still belongs to the past
night (the horses are out from 08:30 to 12:30 and the blankets come off when they are brought
in), so taking the blankets off does not mark the horse done for the coming one. Covering or checking in the morning already counts for the coming night. The forecast of a day is the newest snapshot with
`valid_for = day` (the summary of the time the horse is covered, the horse's cover window, 18:00 to 12:30 the next day by default, see `docs/architecture.md`).

### Recommendation

`blanketplan.Recommend(rules, forecast)`: the first rule by position that matches (`temp_min
<= night min < temp_max`, `NULL` = open, `rain NULL` = any; a rain rule can also require an amount,
`rain_min_mm <= rain_mm < rain_max_mm` over the cover window, JAN-84) wins. Result shape (`recommendation`):

```json
{"status": "blanket|none|no_rule|no_weather", "rule_index": 3, "blanket": {...} , "note": "Wunsch des Besitzers"}
```

- `blanket`: the matched rule names a blanket (`blanket` is set, with `photo_url` and `location`).
- `none`: the matched rule has no blanket ("Keine Decke").
- `no_rule`: there is a forecast but no rule matches.
- `no_weather`: no snapshot for the night yet (graceful, no error). `rule_index` is null.
- `rule_index` is the 0-based index into the `rules` array; `note` is the owner's wish (rule note).

### Cover window (Deckenzeitraum)

Every horse has its own window in which it is covered: `horses.cover_start` and `horses.cover_end` (`time`,
defaults 18:00 and 12:30, migration `0170`). The weather of a horse (its plan, the recommendation, the weather
change push) is the forecast summarised over that window (`blankets.weatherOf`, from the hourly forecast kept in
the snapshot). The weather of `GET /blankets/today` (hero) stays the default window. Owners and admins change it
with `PUT /horses/{id}/cover-window` `{cover_start, cover_end}` ("HH:MM"): start 15:00 to 23:00, end 04:00 to 15:00
(after the day rollover, not past `UncoverUntilHour`), else `400 validation_failed`; riders and members get `403`.
`GET blanket-plan` returns `cover_start` and `cover_end`. The change applies on the next read (no refetch needed);
`blanket_plan.changed` is published.

## API (`/api/v1`, all `auth.RequireStable`)

Blanket JSON: `{id, horse_id, name, fill_g, color, location, photo_path, photo_url}`;
`photo_url` is `/api/v1/files/<photo_path>` (load with the bearer token, `fileSource()` in the app).

| Route | Who | Purpose |
| --- | --- | --- |
| `GET /horses/{id}/blankets` | member | `{blankets}` ordered by fill weight |
| `POST /horses/{id}/blankets` | owner, admin | `{name*, fill_g (0-2000), color, location, photo_path}`, `201`. `photo_path` must come from `POST /files` (`files.Belongs`) |
| `PATCH /horses/{id}/blankets/{blanketId}` | owner, admin | absent = unchanged, `""` clears color, location, photo |
| `DELETE /horses/{id}/blankets/{blanketId}` | owner, admin | `204`; `409 in_use` while a rule names the blanket. The photo file is kept |
| `GET /horses/{id}/blanket-rules` | member | `{rules:[{id, position, temp_min, temp_max, rain, rain_min_mm, rain_max_mm, blanket_id, note}]}` |
| `PUT /horses/{id}/cover-window` | owner, admin | `{cover_start, cover_end}`, see "Cover window" |
| `PUT /horses/{id}/blanket-rules` | owner, admin | `{rules:[{temp_min?, temp_max?, rain?, rain_min_mm?, rain_max_mm?, blanket_id?, note?}]}` full replace in one transaction; array order = priority (positions 1..n). Validation (`400`): max 30 rules, temperatures in -60..60, `temp_min < temp_max`, rain amounts only with `rain: true`, in 0..500 mm and `rain_min_mm < rain_max_mm`, `blanket_id` is a blanket of this horse, note up to 200 characters. `[]` removes all rules |
| `GET /horses/{id}/blanket-plan` | member | `{horse, day, weather, recommendation, rules, blankets, helper_note, state, can_manage, cover_start, cover_end}` (`weather` null without snapshot; `state` = newest state tonight) |
| `POST /horses/{id}/blanket-state` | member | `{action: covered\|uncovered\|checked, covered_with?}` -> `{state, closed_requests}` |
| `GET /horses/{id}/blanket-states?days=14` | member | `{today, states}` newest first, 1..90 days (default 14) |
| `GET /blankets/today` | member | overview below |

`POST blanket-state`: `checked` is for horses that need no blanket according to the plan so they
count as done (it is not enforced). `covered_with` is only allowed with `covered` and must be a
blanket of the horse; if omitted and the plan recommends a blanket, that blanket is stored.
Repeating the newest state of the night (same action and blanket) changes nothing, so double
taps do not clutter the history; any other combination adds a row (e.g. `covered` in the
evening, `uncovered` in the morning). "Done" means: at least one state exists for the day.

`GET /blankets/today`:

```json
{
  "day": "2026-09-30",
  "weather": {"night_min_c": 3, "temp_max_c": 9, "will_rain": true, "rain_probability": 60, "rain_mm": 1.5,
    "rain_peak_mm": 0.8, "rain_hours": 3, "rain_from": "...", "rain_until": "...", "wind_kmh": 12,
    "window_start": "...", "window_end": "...", "timeline": [{"time": "...", "temp_c": 8, "rain_mm": 0.4, "rain_prob": 60, "wind_kmh": 9}], "fetched_at": "..."},
  "reminder_time": "20:30",
  "progress": {"done": 4, "total": 7},
  "horses": [{"horse": {"id","name","box","color_key"}, "recommendation": {...}, "state": {...}|null, "done": false, "is_mine": true}]
}
```

Open horses first, then done ones, each group by name. `total` = all horses of the stable.

## Realtime

After commit: `blanket_state.changed` `{horse_id, day}` (state recorded), `blanket_plan.changed`
`{horse_id}` (blankets or rules changed), and `request.changed` `{id, kind: "done"}` for closed
requests. The app refetches (`useInvalidateOnEvents`).

## Requests close automatically (JAN-39)

In the same transaction as the state, `requests.CompleteBlanketRequests(tx, stable, horse, day,
feedback)` marks open or assigned requests of type `blanket` for that horse whose date range
covers the blanket day as `done` and stores a German `payload.feedback` ("Eingedeckt mit Decke
100 g (Mia)", "Abgedeckt (Mia)", "Geprüft, keine Decke nötig (Mia)"). Other requests, other
horses, other days, cancelled and done requests are untouched. The response lists
`closed_requests`; the app invalidates the requests queries then.

## Automatic uncovering (JAN-78)

On weekdays the farm staff, who do not use the app, take the blankets off at about 12:30 when they
bring the horses in from the paddock; on weekends the owners do it in the app. So the stable can
have the app do it for the staff.

Settings (columns of `stables`, set with `stallfunk-admin stable create|update`, read-only in
`GET /stables/reminder-time` as `auto_uncover_time` and `auto_uncover_days`):

- `auto_uncover_time` (`--auto-uncover HH:MM|off`): stable-local time; NULL = off (default). The
  CLI only accepts times from `RolloverHour:00` up to before `UncoverUntilHour:00` (04:00 to 14:59).
- `auto_uncover_days` (`--auto-uncover-days`): ISO weekdays, 1 = Monday .. 7 = Sunday, default
  Monday to Friday (`mon-fri`); a CHECK allows only 1 to 7.

Job `blanket-auto-uncover` (`Service.RunAutoUncover`, polled every minute): for each stable with
`auto_uncover_time`, when the stable-local weekday is in `auto_uncover_days`, the local time is
at or after `auto_uncover_time` and the local hour is before `UncoverUntilHour` (15:00), the day is
the previous calendar date, i.e. the past night (`StateDay(now, loc, uncovered)`). Every horse of
the stable whose newest state for that day is `covered` gets an `uncovered` state with
`automatic = true`, `changed_by` NULL, `changed_at` = now, `covered_with` NULL. Horses that are
already uncovered or checked, or have no state, are left alone. Each horse runs in its own
transaction under the same advisory lock as `setState` and re-checks the newest state inside it,
so repeated runs, a concurrent tap or a manual "uncovered" never produce a second row. Like
`setState` it closes the open blanket requests of the night (feedback "Automatisch abgedeckt
(Hof)") and publishes `request.changed` and `blanket_state.changed`. It logs a summary at info
level only when something was uncovered. After 15:00 the job does nothing any more: a horse still
covered stays covered (a missed window is not caught up), and the overview of the coming night is
never affected.

`State` in the API has `automatic` (bool). The app shows such a state as "Abgedeckt (Hof,
automatisch)" and "Hof" as the author.

## Last person reminder (JAN-33)

`Service.RunLastPerson` (scheduler job `blanket-last-person`, polled every minute) runs at each
stable's `reminder_time` (default 20:30, stable time zone) and then every 15 minutes until
22:00 (slots R, R+15, ... < 22:00; a job run counts for a slot in its first two minutes). It
looks at the horses without a state tonight:

| People present (open presence visits) | Action |
| --- | --- |
| exactly 1 | push `last_person` to that person with the open horses |
| 0 | push `last_person` to the owners of the open horses (each owner: their own horses) |
| more than 1 | nothing now; the next slot checks again |
| all horses done | nothing |

Presence visibility is irrelevant here: people who hide from others still count, and only the
recipient learns anything. Each user is reminded at most once per night: the job claims
`(user, kind, source, night)` by inserting a `reminders` row (`source_table = blanket_night`,
`source_id` = stable, `due_at` = 04:00 (`RolloverHour`) stable-local of the day; unique index in migration
`0070`); a failed push releases the claim so the next slot retries. Users switch it off with
`reminder_settings` kind `last_person` (honored by the notifier).

### Check-out hook

`presence.AfterCheckOut` (package variable, set in `cmd/api`, nil is a no-op) is called by
`POST /presence/check-out` after a visit was actually closed, so it also covers the geofence
exit. `Service.OnCheckOut` then pushes `last_person` to that user when nobody else is present,
it is 17:00 or later stable-local (`CheckoutFrom`; check-outs during the day are ordinary),
and horses are still open tonight. One per user and night (`source_table = blanket_checkout`).
Chosen over an extra endpoint because the geofence exit runs in the background without the
app calling anything else, and the change in `presence` is four lines.

## Weather change (JAN-34)

`Service.CheckWeatherChange` runs right after every weather refresh (wrapped around
`weatherSvc.Refresh` in `cmd/api`). For each horse that already has a state tonight it compares
the recommendation of the newest snapshot of the night with the one of the snapshot that was
current when the state was set (the forecast the decision was based on;
`blanketplan.Changed`: another blanket, or blanket vs. none). On a change the owner and the
riders get a `weather_change` push ("Luna: jetzt Decke 100 g statt keine Decke."), once per
horse and night (claim `blanket_weather`, `source_id` = horse). Horses without a state, states
set before any forecast existed and snapshots of other days are ignored.

## Mobile

- Weather card (`components/weather-card.tsx`, on the tab and the plan): temperature range, rain
  amount with intensity word, rain timing, peak per hour, probability, wind and an hourly strip
  for the cover window (default 18:00 to 12:30, set per horse by owners and admins in the plan). Snapshots without details show the facts only.
- Tab "Decken": hero with date, night line ("Heute Nacht 3 bis 9 °C, 4,8 mm Regen"), progress "4/7" and
  "Erinnerung um 20:30"; open horses as large cards (avatar, recommendation with blanket photo
  and location, owner wish, buttons "Eingedeckt"/"Abgedeckt" or "Geprüft"), done horses as
  compact rows (tap to correct). Realtime refresh.
- Deckenplan (`/horses/{id}/blanket-plan`): "Heute Nacht" hero, card "Deckenzeitraum" (`components/cover-window-section.tsx`, quarter-hour steps), rule list with the active rule
  highlighted, blanket grid with photos, helper note, history. Owners and admins edit rules
  (sheet with reorder, warning for rules that can never apply; with "Regen" a second choice
  "Wie stark?": Egal, Leicht (under 2 mm), Mäßig (2 to under 8 mm), Stark (from 8 mm), the steps of
  the weather card, or Genau with own "Ab (mm)" / "Unter (mm)"), blankets (sheet with photo
  upload through `lib/upload.ts`) and the helper note.
- Verlauf (`/blankets/history/{horseId}`): the last 60 days.
- Start screen: greeting, night temperature with the recommendations for my horses (owned and
  ridden), presence tile, blanket progress tile, the two newest open requests.
- Pure helpers with tests: `lib/blankets.ts` (texts, progress, greeting), `lib/blankets-rules.ts`
  (rule drafts, validation, reorder, unreachable rules).

## Seed

`seed/blankets.go`: locations for the existing blankets and one blanket plus two rules for every
demo horse that had none (Fanta, Cookie, Merlin, Pepe, Nala). Luna and Balu keep their rules.

## Tests

`go test ./internal/blankets ./internal/requests` (needs `REITERHOF_TEST_DATABASE_URL`):
permissions and validation, stable isolation, recommendation wiring with fixed snapshots, day
boundary, progress and ordering, history, realtime events, request auto-close, automatic uncovering
(weekday/weekend/disabled/window, idempotent, request closing, schema), last person
(0/1/2 present, re-check until 22:00, idempotent, opt-out, retry after a failed push), check-out
hook, weather change. Time is a fake clock.
