# Training (M6)

Tickets: JAN-55 profile, JAN-56/62 logic (pure packages), JAN-57 "Was heute?", JAN-58 exercise
library, JAN-59 week view, JAN-60 "Nur eintragen", JAN-61 finish screen.

- Backend: `backend/internal/trainingapi` (HTTP + SQL). The rules live in the pure packages
  `internal/training`, `training/load` (load score, week segments, assessment) and
  `training/recommend` (recommender); see "Training logic" in [architecture.md](../architecture.md).
- Migration `0060_training_sessions.up.sql`: `sessions.exercise_id` and CHECK constraints for
  activity, feel and focus rating. Tables `training_profiles`, `exercises`, `sessions`, `week_slots`
  and `reha_plans` come from `0003`.
- Mobile: `app/(tabs)/training.tsx` (Was heute?), `app/training/session.tsx` (timer placeholder),
  `app/training/session/finish.tsx`, `app/training/week.tsx`, `app/horses/[id]/training-profile.tsx`;
  pure helpers `lib/training*.ts`, API client and React Query hooks `lib/api/training.ts`.

## Access

All routes are `auth.RequireStable` and stable-scoped. A horse outside the caller's stable is `404`.

| Role | Profile | Today / week / sessions (read) | Log sessions | Take week days |
| --- | --- | --- | --- | --- |
| owner, admin | read, write | yes | yes | any day, any user, rest days |
| rider | read (only the own rider rules) | yes | with `log_sessions` | with `take_week_slots`, only free days, only for self |
| other member | `403` | `403` | `403` | `403` |

### Rider rule mapping

Two sources are combined; a rider without an entry sees nothing (fail closed).

- `horse_riders.rules` (positive list of keys): `log_sessions` (POST sessions),
  `take_week_slots` (PUT week day), `hack_alone` (adds `MayHackAlone`), `shows` (adds
  `MayRideShows`), `report_observations` (not used here). The legacy key `ride` (first seed data,
  `["ride","groom"]`) implies `log_sessions` and `take_week_slots`.
- `training_profiles.rb_rules`: object keyed by rider user id,
  `{allowed_activities: [...], max_intensity: any|light|medium|intense, may_hack_alone, may_ride_shows}`.
  This maps to `training.RiderRules`. **No entry for the rider means `nil` rules**, which the
  recommender treats as "nothing allowed": every activity is hidden and the answer is a rest day.
  The owner edits the entries with `PUT /training-profile` (`rider_rules`).

## API (`/api/v1`)

Errors use the standard envelope (`validation_failed`, `forbidden`, `not_found`, `conflict`).

| Route | Purpose |
| --- | --- |
| `GET /training/horses` | Horse switcher: horses the user owns or rides, `role`, `profile_status`, `has_profile` |
| `GET /horses/{id}/training-profile` | Profile (`can_edit`, `rider_rules`); a horse without row returns the defaults with `exists:false` |
| `PUT /horses/{id}/training-profile` | Owner/admin only. Full replace; `rider_rules` omitted keeps the stored rules |
| `GET /horses/{id}/today?minutes=45` | Recommendations, hidden activities, 7-day dots, context line, active reha phase |
| `POST /horses/{id}/sessions` | Quick log or finished session, returns `{session, next_progression}` |
| `GET /horses/{id}/sessions?limit=&before=` | Newest first; riders see `visible_to_rider` sessions and their own |
| `GET /horses/{id}/week?start=YYYY-MM-DD` | Monday to Sunday (any day of the week works as `start`), load segments, assessment |
| `PUT /horses/{id}/week/{day}` | Claim a day (`{}` = "Ich"), plan, rest day or release (`status: open`); returns the week |
| `GET /exercises?discipline=&level=&tag=` | Global (`stable_id NULL`) and own-stable exercises, easiest first along the progression |
| `GET /exercises/{id}` | With `steps` and the follow-up exercise (`next_exercise_id`, `next`) |

### Profile

`discipline` is one of `dressage, jumping, eventing, leisure, western, young_horse` (required).
`allowed_activities` entries are `{activity, mode: on|off|conditional, note}`; conditional needs a
note; activities missing from the list count as off. `shows` are `{date, name, classes?, helper?}`
(`classes` and `helper` feed the week view). `rhythm`: 1-7 sessions, 0-6 rest days, `max_minutes`
0-240 (0 = no limit), `rest_after_show`. Missing rhythm keys keep the defaults of
`training.DefaultRhythm()` (4-5 sessions, 1-2 rest days, 60 minutes, rest day after a show); when
decoding a stored profile the API always starts from that default. `status` is `fit|reha|pause`.

### Today

Input for `recommend.Recommend`: profile, sessions of the last 14 days, the latest
`weather_snapshots` row for today (`Rain` = `will_rain`, `TempC` = forecast **night minimum**, so
the context line shows that temperature), the ground condition of the stable, `minutes` as
available time, and the role: owner/admin as `RoleOwner`, riders as `RoleRider` with the mapped
rules. All times are converted to the stable's timezone (`stables.timezone`) first; "today" and the
session days are calendar dates in that zone.

Context line: `Regen, 6 °C · Boden nass · Turnier in 3 Tagen · Reha: Phase 2`, each part only when
known (weather snapshot, ground not dry, next show within 14 days, active reha phase).

Each recommendation may carry `exercise` (with steps): for hall, arena and lunge the dressage
library (jumping library for jumping horses), for jumping and groundwork the matching library. It is
the first exercise, easiest first along the progression, that the horse has not mastered yet
(no session with focus rating "Sitzt" for it).

Dots (`week`): the last seven days, today last; `kind` is `trained` (a session or done slot),
`rest` (planned rest day or the mandatory rest day after a show) or `nothing`; `intensity` is the
classification of the day's load.

**Reha phases.** `reha_plans.phases` is a JSON list of consecutive phases starting at `start_date`:
`{name, days | weeks | duration_days, activity, min_minutes, max_minutes, conditions}`. The phase
that covers today is passed to the recommender; before the start, after the last phase or without
active plan there is no phase (status `reha` then means light activities only).

### Sessions

Body: `activity`, `minutes` (1-600), optional `canter_share` (0-1), `started_at` (RFC 3339, default
now, at most 14 days back), tracked data `gait_shares` (`walk|trot|canter|halt`, fractions, sum at
most 1), `rein_changes` (ordered segments `{rein: left|right, minutes}`, the number of rein changes
is the segment count minus one), `distance_m`, and `feel` (`fresh|loose|tired|tense`),
`focus_rating` (1 Schwer, 2 Besser, 3 Sitzt), `exercise_id`, `note`, `visible_to_rider` (default
true). `source` is `tracked` when gait shares, rein changes or a distance are sent, else `quick`.

`load_score` is `load.Score(minutes, activity, canterShare)`, stored in `sessions.load_score`; the
canter share is `gait_shares.canter`. Saving marks the week slot of that day (stable-local) as
`done` (created if nobody planned the day; an existing planner stays). With `focus_rating: 3` and an
`exercise_id`, the response contains `next_progression` (the exercise's `next_exercise_id`).

### Week

Each day has a `status`: `done` (a session exists), `today`, `planned` (someone claimed a future
day), `open` ("Niemand eingetragen"), `empty` (past, nothing) or `rest` (`rest_reason`:
`after_show` fixed, `planned`). Show days carry `show {name, classes, helper}`. `segments` are the 7
load levels and `assessment` is the German sentence from `load.Assess`. Hidden sessions (riders) still
count for status and load but reveal neither user nor activity. Days are calendar days in the
stable's timezone, so DST weeks still have seven days.

## Seed

`internal/seed/training.go`: profiles for all seven demo horses (the three minimal rows are filled
only while `discipline` is still empty, so local edits survive), 11 global exercises (dressage
Übergänge, Zirkel verkleinern und vergrößern, Schulterherein, Travers, Traversale; jumping
Stangenarbeit, Cavaletti, Gymnastikreihe; groundwork Führen und Halten, Rückwärtsrichten,
Freiarbeit), a reha plan for Fanta, 20 sessions in the two weeks before the day of seeding and
three planned week slots for Luna.

## Open points

- The tracking screen (`app/training/session.tsx`) is a timer placeholder; the later tracker hands
  `gait` and `rein` (JSON) plus `started_at` to `session/finish` in the same way.
- `/horses/{id}/reha` (link "Reha-Plan") and `/observations/new?horse=` ("Etwas aufgefallen?") are
  built by other work; the reha link only shows for horses in reha.
- The weather temperature is the night minimum of the snapshot; a daytime value would need an
  extension of `weather_snapshots`.
