# Reha plan (Reha-Plan, M7)

Tickets: JAN-66 data model and screen, JAN-67 integration into training and requests, JAN-42 rules in
`exercise` requests. JAN-68 creates a plan from an observation (`observation_id`).

- Backend: `backend/internal/reha` (pure phase logic in `phase.go`, HTTP in `handler.go`/`view.go`,
  reminder job in `job.go`, queries in `store.go`). Migration `0100_reha.up.sql`.
- Mobile: `app/horses/[id]/reha.tsx` (plan), `app/reha/edit.tsx` (create/edit, phase editor sheet),
  `components/reha-*.tsx`, `lib/api/reha.ts`, pure helpers and tests in `lib/reha.ts`.
- Tables `reha_plans` and `reha_days` come from `0003`; `0100` adds `abort_criteria`, `ended_at`,
  `created_by` and the unique index that claims checkup reminders.

## Model

A plan belongs to one horse: `diagnosis`, `vet`, `start_date`, `phases`, `checkup_date`,
`abort_criteria`, optional `observation_id` (must be an observation of the same horse). **At most one
plan per horse is active** (partial unique index `reha_plans_one_active_per_horse`); creating a plan
ends the previous one in the same transaction (`active = false`, `ended_at`). Ended plans stay as
history and cannot be changed.

`phases` is a JSON list of consecutive phases beginning on `start_date`:

```json
{"name": "Schritt führen", "days": 9, "activity": "groundwork", "min_minutes": 10, "max_minutes": 20, "conditions": "nur Boden fest"}
```

- `activity` is a training activity (`hall, arena, hack, lunge, jumping, groundwork, walker`) or
  `rest` (Boxenruhe: no exercise, minutes must be 0).
- Validation: 1-20 phases, `days` 1-365 (plan at most 730 days), `name` required (80), `min_minutes`
  1..`max_minutes` <= 240, `conditions` 500 characters, `abort_criteria` 1000, `diagnosis` required (200).
  `checkup_date` must not be before `start_date`, `start_date` at most one year ahead.
- Stored phases may also use `weeks` or `duration_days` instead of `days` (data written by hand or by
  the first training code); `reha.ParsePhases` reads all three, everything written by the API uses `days`.
- The spec example: Boxenruhe 5 d, Schritt führen 10→20 min 9 d, Schritt reiten 20→40 min 14 d,
  Trab aufbauen 2→15 min 14 d (used in the tests).

### Dates and the minute ramp

All dates are stable-local calendar dates (`stables.timezone`), handled as UTC-midnight dates, so a
DST change never moves a phase. Phase `n` starts the day after phase `n-1` ends (`end` is inclusive).

"Heute erlaubt" is the current phase's activity plus the minutes of the day: they rise linearly from
`min_minutes` on the first to `max_minutes` on the last day of the phase, rounded half up
(10→20 over 9 days: 10, 11, 13, 14, 15, 16, 18, 19, 20). A one-day phase allows `max_minutes`.
Before the start and after the last phase there is no unit (plan `state` is `upcoming` / `finished`;
a finished plan stays active until somebody ends it).

## Access

All routes are `auth.RequireStable`, stable-scoped; a horse or plan of another stable is `404`.

| Role | See "Heute erlaubt" | See the whole plan and history | Mark a day | Create, change, end |
| --- | --- | --- | --- | --- |
| owner, admin | yes | yes | yes | yes |
| rider | yes | yes | with `log_sessions` (legacy `ride`) | no |
| other member | yes | no | no | no |

Other members can see today's rule because they may be asked to exercise the horse (request type
`exercise`); they do not get diagnosis, vet, dates, abort criteria or history (`view: "today"`).

## API (`/api/v1`)

| Route | Purpose |
| --- | --- |
| `GET /horses/{id}/reha` | `{view: full\|today, date, can_edit, can_mark_done, has_active_plan, plan, today, history}` |
| `POST /horses/{id}/reha-plans` | Owner/admin. Body `{diagnosis, vet?, start_date, phases[], checkup_date?, abort_criteria?, observation_id?}`. `201` with the view. Ends an active plan. |
| `PATCH /reha-plans/{id}` | Owner/admin, active plan only (`409 not_active`). Absent fields stay; `""` clears `vet`, `checkup_date`, `abort_criteria`; `phases` replaces the list. |
| `POST /reha-plans/{id}/end` | Owner/admin, idempotent. |
| `POST /reha-plans/{id}/days/{date}/done` | "Heute erledigt". Owner/admin/rider with `log_sessions`. Not in the future, only inside the phases; idempotent, the first person stays recorded. |
| `DELETE /reha-plans/{id}/days/{date}/done` | Undo: owner/admin or the person who marked the day (`403` otherwise). |

Every write answers with the new view. `plan` has `phases[]` with computed `start_date`, `end_date`,
`status` (`past|current|upcoming`), `current_phase` (1-based, 0 = none), `day_index`, `total_days`,
`checkup_in_days`, `done_days[]`. `today` has `phase, phase_index, phases, day_in_phase, days_in_phase,
activity, activity_label, rest, minutes, min_minutes, max_minutes, conditions, text, done, done_by`.
Writes publish the realtime event `reha.changed {horse_id}`.

## Training status

Creating a plan sets `training_profiles.status = 'reha'` (the profile row is created if the horse has
none). Ending the plan sets it back to `fit`, **but only if it is still `reha`**: an owner who switched the
horse to `pause` meanwhile keeps that choice. Replacing a plan leaves the status `reha`. The owner can
still change the status in the training profile; with an active plan that has a unit today, the plan wins
in "Was heute?".

## Integration (JAN-67)

- **Was heute?** (`GET /horses/{id}/today`): while a phase covers today, the recommender returns only that
  unit (`recommend.RehaPhase`). The upper bound is today's ramp value, the lower bound the phase minimum, so
  less available time shortens the unit but never below the phase minimum. A rest phase recommends a rest day.
  The `reha` block carries `minutes`, `rest`, `done` and `text`. Riders whose rules do not allow the activity
  see a rest day (fail closed).
- **Week** (`GET /horses/{id}/week`): every day of the plan has `reha {plan_id, phase, activity, activity_label,
  rest, minutes, done}`; `done` is the "Heute erledigt" mark. The day `status` still comes from sessions and slots.
  Marking a day does not create a training session (no load is booked); log a session for that.
- **Requests** (JAN-42): for `exercise` requests with a horse, the server writes the plan's rule for the
  request's date into the first line of `payload.rules_note` (`Reha: Schritt führen 14 min, nur Boden fest`,
  `Reha: Boxenruhe – heute keine Bewegung`, or `Reha-Plan aktiv: für diesen Tag ist keine Einheit vorgesehen`).
  Lines the requester typed stay below. Editing a request refreshes the line, and it is removed once the plan
  has ended. Lines starting with `Reha:` typed by the requester are replaced. Occurrences of a recurring series
  copy the template's payload and keep the rule of the template's day.

## Checkup reminder

Scheduler job `reha-checkup-reminders` (every 15 minutes, `reha.Reminders`): for active plans with
`checkup_date`, from 08:00 stable-local time it sends push kind `reha_checkup` to the owner and all riders
**2 days before** and **on the day**. Each reminder is claimed by a `reminders` row (`source_table =
'reha_plans'`, unique per user, kind, plan and scheduled time), so any number of runs sends it once; a failing
push releases the claim and the next run retries. Moving `checkup_date` creates a new scheduled time and
reminds again. Users can switch the kind off in `reminder_settings`. The push data carries
`route` and `screen`: `/horses/{id}/reha`.

## Mobile

`/horses/[id]/reha`: hero "Heute erlaubt" (activity, minutes, conditions) with "Heute erledigt" (and "Rückgängig"),
plan card with progress, phase timeline (current phase highlighted), abort criteria, vet checkup with
date, link "Auffälligkeit melden" (`/observations/new?horse=<id>`), history. Owner and admins get
"Reha-Plan anlegen" / "Plan bearbeiten" / "Plan beenden". `/reha/edit?horse=<id>[&plan=<id>][&observation=<id>]`
is the form with the phase editor sheet (add, edit, remove, reorder). With `observation` (JAN-68, from "In Reha-Plan
umwandeln" in the horse record) a new plan starts with the diagnosis taken from the observation
(`diagnosisFromObservation`: category, body part and description, at most 200 characters) and posts `observation_id`.
The plan screen shows the source observation ("Ausgelöst durch", opens its detail); the observation detail links back. The training week screen shows the reha
line of each day.

## Open points

- The general training rules of a horse (profile activities and conditions) are not yet part of `rules_note`,
  only the reha plan.
- The tracker screen does not yet offer "Heute erledigt" after a session; the button is on the reha screen.
