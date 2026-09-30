# Reminders and settings (Erinnerungen und Einstellungen, M8)

Tickets: JAN-69 (reminder center), JAN-70 (settings), rest of JAN-18 (push: settings, delivery, tap handling).

- Backend: `backend/internal/reminders` (`list.go` reminder center, `settings.go` notification switches and the
  reminder time, `job.go` evening training plan push). Migration `0120_reminders.up.sql`.
- Mobile: `app/reminders/index.tsx`, `app/settings/index.tsx`, `components/reminder-*.tsx`, `lib/api/reminders.ts`,
  pure helpers with tests in `lib/reminders.ts` and `lib/notifications-core.ts`, the tap handler in `lib/notifications.ts`
  (used by `app/_layout.tsx`).
- No new table: the center reads `reminders`, the switches are `reminder_settings`, the time is `stables.reminder_time`.

## Where reminders come from

| Kind | Written by | Row in `reminders` | `source_table` |
| --- | --- | --- | --- |
| `last_person`, `weather_change` | `internal/blankets` | yes, when sent | `blanket_night`, `blanket_checkout`, `blanket_weather` |
| `health_due`, `medication` | `internal/health` job | yes, when sent | `health_items` |
| `reha_checkup` | `internal/reha` job | yes, when sent | `reha_plans` |
| `helper` (reminder) | `internal/requests` job | yes, when sent | `requests` |
| `helper` (accepted, changed, cancelled), `new_request` | `internal/requests` | no, push only | |
| `observation`, `urgent_observation` | `internal/observations` | no, push only (an alert is not a to-do) | |
| `training_plan` | `internal/reminders` job (this milestone) | yes, when sent | `training_plan` |

`training_plan` was not sent by anything before. The job `training-plan-reminders` (every 15 minutes, `Service.RunTrainingPlan`)
sends it from 19:00 to 22:00 stable-local time to everybody with **planned** `week_slots` (`user_id` set, `status = planned`)
for tomorrow: "Training morgen", body "Fanta: Halle, Luna: Ausritt", `data = {screen: "/training/week"}`. One push per user and
evening: the job claims `(user, kind, stable, 19:00 of the evening)` with a `reminders` row (unique index `reminders_training_plan_once`,
migration `0120`); a failed push releases the claim. Users who switched the kind off get no claim row.

## Reminder center (JAN-69)

`GET /api/v1/reminders?range=today|week` (default `week`, `auth.RequireStable`; `400 validation_failed` for anything else).
Times are in the stable's time zone (`stables.timezone`). Response:

```json
{
  "range": "week", "today": "2026-09-30", "timezone": "Europe/Berlin",
  "blanket_check": {"day": "2026-09-30", "time": "20:30", "due_at": "...", "done": 4, "total": 7,
                    "state": "upcoming", "screen": "/blankets"},
  "groups": [
    {"key": "today", "label": "Heute", "items": [Item]},
    {"key": "week", "label": "Diese Woche", "items": [Item]}
  ]
}
```

`range=today` returns only the `today` group. "Diese Woche" are the next six days after today (a rolling window, not the
calendar week). `Item`: `id`, `kind`, `title`, `body`, `due_at`, `all_day`, `sent_at`, `screen`, `computed`, `dismissible`.

`blanket_check` is tonight's check: `time` is the stable's reminder time, `done/total` the horses with a `blanket_states` row for
today's date (the stable-local calendar day is the blanket day of tonight; before noon that is still the coming night),
`state` is `empty` (no horses), `done`, `due` (reminder time reached, horses open) or `upcoming`.

**Stored items**: the user's own `reminders` rows that are not dismissed and whose time (`sent_at`, else `due_at`) lies in the window.
They are placed at `sent_at` when it exists (what the user experienced), otherwise at `due_at`. `dismissible = true`.

**Computed items** (not rows yet, `computed = true`, `dismissible = false`, id `c:<kind>:<source>:<date>`), for the user's own
horses (owner or rider) or requests:

| Kind | Source | Placed at |
| --- | --- | --- |
| `medication` | `health_items` with `daily_time` (course not over) | today at that time |
| `health_due` | `health_items` without `daily_time` due in the window, **overdue ones too** (listed under "Heute") | due date, all day |
| `reha_checkup` | active `reha_plans` with `checkup_date` in the window | checkup date, all day |
| `helper` | open/assigned requests the user helps with (`request_assignees`), date in the window | date and `time_from` (all day without) |
| `training_plan` | planned `week_slots` of the user for day D (tomorrow up to the end of the window) | 19:00 on D-1 |

`training_plan` is optional: users who switched the kind off do not see it. Computed items are dropped when a stored row for the same
thing is in the window (same kind and source, e.g. the medication push of 08:00 already made its row), so nothing shows up twice.
Computed items cannot be dismissed; they are completed where they live (health "erledigt", request done, ...). At most 100 items.

`screen` follows the convention of the features (`data.screen` of their pushes): `/blankets`, `/horses/{id}/blanket-plan`,
`/horses/{id}/health`, `/horses/{id}/reha`, `/requests/{id}`, `/training/week`, or `""` (none; for example when the source row is gone).

`POST /api/v1/reminders/{id}/dismiss` -> `204` (sets `dismissed_at`, idempotent). Only the user's own stored reminders:
`404 not_found` for other users' rows, unknown and malformed ids.

## Notification settings (JAN-70)

`GET /api/v1/settings/notifications` -> `{items: [{kind, label, description, enabled, opt_in}], push_consent}` with **every**
`push.Kinds()` entry in display order (a test checks the catalog is complete). `label`/`description` are German UI copy in
`settings.go`. `enabled` follows the notifier: opt-out kinds are on unless a `reminder_settings` row says `enabled = false`, the opt-in
kind `new_request` (`push.OptIn`) is off unless a row says `enabled = true`. `push_consent` is the privacy consent `push`.

`PUT /api/v1/settings/notifications/{kind}` `{enabled: bool}` -> the changed item (`404` unknown kind, `400` without `enabled`).

`GET/PATCH /api/v1/requests/notify-settings` stay (same `reminder_settings` row of kind `new_request`, so both are always consistent);
the app no longer uses them: the requests tab links to the settings screen.

### Push consent

`push.Notifier.NotifyUsers` only sends to users with a current consent `push` (row in `consents` with `revoked_at IS NULL`); a user
who never consented, or revoked, gets nothing, whatever the switches say. Revoking still deletes the tokens (`privacy.Set`); the
check also covers tokens registered without consent. Tests that register tokens call `pushtest.GrantConsent`.

## Reminder time of the stable

`GET /api/v1/stables/reminder-time` -> `{reminder_time: "20:30", can_edit}` (every member, read only for non-admins).
`PUT` `{reminder_time: "HH:MM"}` is **admin only** (`403 forbidden`), `400 validation_failed` unless `HH:MM` between `16:00` and `22:00`
(the last-person job starts at that time and re-checks every 15 minutes until 22:00; the blanket day rolls over at noon). The blankets
job (`RunLastPerson`) reads `stables.reminder_time` in every run, so the new time applies the same evening (test
`TestLastPersonJobUsesNewReminderTime`). Presence visibility is not duplicated: the settings screen uses `PATCH /me`
(`presence_visibility`) like the presence screen.

## Mobile

- `/reminders`: hero "Deckencheck" (time of the reminder, progress "4/7", button "Zu den Decken"), "Heute", "Diese Woche" with a
  heading per day. Tapping a row opens its `screen` (only internal whitelisted routes), the check icon "Erledigt" dismisses a stored
  reminder. Refreshes on `blanket_state.changed` and `request.changed`.
- `/settings`: "Benachrichtigungen" (switch per kind, banner "Mitteilungen erlauben" without push consent, which opens the consent
  sheet), "Stallgasse" (reminder time in quarter-hour steps with Speichern, admins only; members see the time), "Anwesenheit"
  (visibility), "Datenschutz" (link to `/settings/privacy`), "Konto" (Abmelden). Entry points: bell and gear icons on the start
  screen, "Alle Einstellungen" on the presence screen, link on the requests tab.
- Notification taps (`lib/notifications.ts`, `useNotificationNavigation` in the root layout): the tap while the app runs (listener) and
  the tap that started it (`getLastNotificationResponse`, cleared after handling) open `data.screen`, else `data.route` (health and reha
  pushes still use that key), else `request_id` / `observation_id` (`notificationRoute`). Only routes starting with `/blankets`,
  `/horses/`, `/requests/`, `/observations/`, `/reminders` or `/training` (whole segments, no `..`, no scheme) are opened; anything
  else is ignored. A tap before sign-in waits for sign-in. Foreground notifications are shown as banner and in the list.

## Open points

- The reminder center does not list observation alerts and pushes that are sent without a `reminders` row (new request, request
  changes); they exist as notifications only.
- Computed items cannot be dismissed.
- The notification tap handling and the foreground presentation are not tested on a device (needs a real device and an EAS project id).
- `lib/api/requests.ts` still has `getNotify`/`setNotify` although no screen uses them.
