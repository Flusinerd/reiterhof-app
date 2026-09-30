# Training (M6)

Tickets: JAN-55 profile, JAN-56/62 logic (pure packages), JAN-57 "Was heute?", JAN-58 exercise
library, JAN-59 week view, JAN-60 "Nur eintragen", JAN-61 finish screen, JAN-89 week plan (rules and AI).

- Backend: `backend/internal/trainingapi` (HTTP + SQL). The rules live in the pure packages
  `internal/training`, `training/load` (load score, week segments, assessment) and
  `training/recommend` (recommender); see "Training logic" in [architecture.md](../architecture.md).
- Migration `0240_week_slot_focus.up.sql`: `week_slots.focus`, `week_slots.exercise_id` (JAN-92).
- Migration `0060_training_sessions.up.sql`: `sessions.exercise_id` and CHECK constraints for
  activity, feel and focus rating. Tables `training_profiles`, `exercises`, `sessions`, `week_slots`
  and `reha_plans` come from `0003`.
- Mobile: `app/(tabs)/training.tsx` (Was heute?), `app/training/session.tsx` (chooser GPS/indoor, see [tracking.md](tracking.md)),
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
| `PUT /horses/{id}/week/{day}` | Claim a day (`{}` = "Ich"), plan, rest day or release (`status: open`); optional `focus` (max 80) and `exercise_id` (global or own stable). Fields not sent (or `null`) keep their value, so "Ich" on a planned day keeps activity, note, focus and exercise; `focus: ""` / `exercise_id: ""` clear that field (JAN-95); a rest day clears activity, focus and exercise. Returns the week (days carry `focus`, `exercise {id, title}`) |
| `POST /horses/{id}/week/plan?start=&day=` | Owner/admin only. Proposal for the open days of the week (rules, with the owner's consent a language model); stores nothing. With `day=YYYY-MM-DD` only that day is planned again; optional body `{draft: [{date, activity or "rest", minutes}], exclude: [activity]}` (empty body and `{}` are fine), see [Week plan](#week-plan) |
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

**Week structure** (JAN-93, `rhythm.days`, `rhythm.quotas`): `days` are seven rules, Monday first,
`{kind, activity?}` with `kind` one of `""` (free, the planner decides), `rest`, `recovery` (active recovery),
`light`, `normal`, `demanding` or `activity` (with an allowed `activity`). At most `rest_days_max` days may be
`rest`. `quotas` are units per week (Monday to Sunday): `{demanding, recovery, activities: {hall: 2, ...}}`, each
0-7, activities must be allowed, and all quota units plus `rest_days_min` must fit into seven days. `demanding`
and the activity quotas are also maximums, `recovery` is a minimum only. Like the other rhythm keys, missing
`days` or `quotas` (or `days: []`) mean the default: free days, no quotas. The app edits both in "Wochenstruktur"
and "Wochenziele" of the training profile.

**Unit levels** (`recommend.UnitLevel`) come from the load of a unit (`load.Score` without canter, i.e.
minutes × factor): active recovery below 20, light below 30, normal below 45, demanding from 45.

**Structure rules** (`recommend.structureFor`, applied by `Recommend`, `Check` and the week plan) come after
the safety rules (profile, rider rules, reha, show, pause, frozen ground) and turn into a rest day, activities
that are out and a minute window per activity. In order; a group is dropped when nothing would be left after
it, except the load limit:

1. The owner's rule for the weekday (fixed rule, not a preference).
2. Default rules, only on free days: no demanding unit after a demanding day; after two days with at least
   normal load only a light unit (or less).
3. Needs of the week (week plan only): when the open days left, or the sessions `sessions_max` still allows,
   are needed for missing rest days (`rest_days_min`) or quota units, the day fills one of them.
4. Quota maximums: no more demanding units and no more units of an activity than set.
5. Load limit: when the 14 days before the week have at least 3 sessions, the week's load may be at most
   1.2 × their weekly average (at least 90); when no unit fits, the day becomes a rest day.

### Today

Input for `recommend.Recommend`: profile, sessions of the last 14 days, the latest
`weather_snapshots` row for today (`Rain` = `will_rain`, `TempC` = forecast **night minimum**, so
the context line shows that temperature), the ground condition of the stable, `minutes` as
available time, and the role: owner/admin as `RoleOwner`, riders as `RoleRider` with the mapped
rules. All times are converted to the stable's timezone (`stables.timezone`) first; "today" and the
session days are calendar dates in that zone.

Context line: `Regen, 6 °C · Boden nass · Turnier in 3 Tagen · Reha: Phase 2`, each part only when
known (weather snapshot, ground not dry, next show within 14 days, active reha phase).

Each recommendation may carry `exercise` (with steps): for hall and arena the dressage
library (jumping library for jumping horses), for jumping, groundwork and lunge the matching library. It is
the first exercise, easiest first along the progression, that the horse has not mastered yet
(no session with focus rating "Sitzt" for it).

Dots (`week`): the last seven days, today last; `kind` is `trained` (a session or done slot),
`rest` (planned rest day or the mandatory rest day after a show) or `nothing`; `intensity` is the
classification of the day's load.

**Reha phases.** `reha_plans.phases` is a JSON list of consecutive phases starting at `start_date`
(`{name, days | weeks | duration_days, activity, min_minutes, max_minutes, conditions}`). Parsing, the
calendar layout and the minute ramp live in `internal/reha` (see [reha.md](reha.md)). The phase that covers
today is passed to the recommender with today's ramp value as upper and the phase minimum as lower bound; a
`rest` phase (Boxenruhe) recommends a rest day. Before the start, after the last phase or without active
plan there is no phase (status `reha` then means light activities only). The `reha` block of the response
also has `minutes`, `rest`, `done` (the "Heute erledigt" mark) and `text`.

### Sessions

Body: `activity`, `minutes` (1-600), optional `canter_share` (0-1), `started_at` (RFC 3339, default
now, at most 14 days back), tracked data `gait_shares` (`walk|trot|canter|halt`, fractions, sum at
most 1), `rein_changes` (ordered segments `{rein: left|right, minutes}`, the number of rein changes
is the segment count minus one), `distance_m`, and `feel` (`fresh|loose|tired|tense`),
`focus_rating` (1 Schwer, 2 Besser, 3 Sitzt), `exercise_id`, `note`, `visible_to_rider` (default
true). `source` is `tracked` when gait shares, rein changes or a distance are sent, else `quick`.

A non-empty `track` (GPS) needs the `location_tracking` consent, else `403 consent_required`
(`privacy.Has`, JAN-19); see [privacy.md](privacy.md).

`load_score` is `load.Score(minutes, activity, canterShare)`, stored in `sessions.load_score`; the
canter share is `gait_shares.canter`. Saving marks the week slot of that day (stable-local) as
`done` (created if nobody planned the day; an existing planner stays). With `focus_rating: 3` and an
`exercise_id`, the response contains `next_progression` (the exercise's `next_exercise_id`).

### Week

Each day has a `status`: `done` (a session exists), `today`, `planned` (someone claimed a future
day), `open` ("Niemand eingetragen"), `empty` (past, nothing) or `rest` (`rest_reason`:
`after_show` fixed, `planned`). Show days carry `show {name, classes, helper}`; days covered by an active reha plan carry `reha {plan_id, phase,
activity, activity_label, rest, minutes, done}`. `segments` are the 7
load levels and `assessment` is the German sentence from `load.Assess`. Hidden sessions (riders) still
count for status and load but reveal neither user nor activity. Days are calendar days in the
stable's timezone, so DST weeks still have seven days.

## Week plan

`POST /horses/{id}/week/plan?start=YYYY-MM-DD` (JAN-89, owner and admins; riders and members `403`). The handler is
`trainingapi/plan.go`, the logic the pure package `training/weekplan`; the Mistral client is `internal/mistral`.

- **Open days:** today or later and no activity planned: status `open`, `today`, or `planned` by someone without an
  activity (the proposal keeps that person, `user`). Done days, rest days (planned or after a show) and days with a
  planned activity are context; a planned activity counts as a session for the following days.
- **Rules:** `weekplan.Rules` asks `recommend.Recommend` day by day with the sessions of the 14 days before plus the
  units planned so far, so variety, recent load and the weekly maximum apply across the week. Weather and ground are
  only known for today. The active reha plan gives each day its unit.
- **Language model:** used when `REITERHOF_MISTRAL_API_KEY` is set and the horse's **owner** has the `ai_training`
  consent (whoever plans; an admin planning someone else's horse needs that owner's consent) and stated to be 16 or
  older (`users.age_confirmed_at`; Mistral's terms forbid personal data of children below the age of digital
  consent, so a parental consent is not enough). `weekplan.Prompt` sends discipline, the level only as a library
  level (`training.LevelClass`: E/A beginner, A*-L* intermediate, L** and up advanced, anything else nothing), the
  horse's age in years, status, rhythm with the week structure (`days[].rule`), the quotas and the load limit
  (`load.previous_weeks_average`, `load.week_limit`), allowed activities (without notes), the sessions of the last 14 days as days
  ago, activity, minutes, load, canter share, feel and the library exercise with its rating (hard, better, solid),
  shows as days from today, the reha unit per day, the weather of today and tomorrow and today's ground, and the
  candidate exercises: the **global** library (own-stable exercises are free text and stay on the server) in the
  libraries that fit the allowed activities (`training.ExerciseLibrary`), in progression order, with short keys
  `E1`, `E2`, ..., level, title, tags, mastered and the key of the follow-up exercise. No names, ids, free text or
  dates; a test checks this.
- **System prompt** (`weekplan.SystemPrompt`, JAN-92): role (riding instructor in the FN system, welfare first),
  the input, every activity with German name, meaning, intensity and sensible duration, then the planning rules:
  unit levels, the week structure (the owner's rules and quotas are binding; on free days no two demanding days in a
  row and a light unit after two days of normal or demanding load; rest days minimum, active recovery, load limit,
  rhythm),
  content by the training scale (Takt and Losgelassenheit first; Anlehnung, Schwung, Durchlässigkeit for
  intermediate; Geraderichtung and Versammlung only for advanced, fit horses, at most twice a week), discipline,
  age, shows, weather, reha and pause; how to pick focus and exercise; rules for the German reason (third person,
  concrete cause, no "du") with examples; a checklist before answering. The answer is
  `{"days":[{"day","activity","minutes","focus","exercise","reason"}]}`. `weekplan.Parse` drops unknown or closed
  days and unknown activities; an unknown exercise key becomes no exercise.
- **Checking:** `recommend.Check` tests each proposal against the hard rules (visibility, reha phase, rest after a
  show, pause/reha/show/frozen-ground filters, `rhythm.sessions_max`, the structure rules above) and fits the minutes (reha range, rhythm
  maximum, the activity's sensible range from `recommend.MinutesRange`, e.g. lunge 15-30, hall 30-60, hack 30-120,
  pause 20, before a show 30). A failing proposal is replaced by the rules (`replaced` says why);
  days the model left out come from the rules. A proposed exercise stays only when its library fits the activity;
  days from the rules get the first exercise of their library the horse has not mastered, and no focus.
- **Response:** `{horse_id, start, end, source: ai|rules, ai_status, owner_is_me, days: [{date, weekday, activity
  (or rest), label, minutes, intensity, intensity_label, reason, note, source, replaced, user, focus, exercise
  {id, title}, level, level_label}]}` (`level` is the unit level, absent for rest). `ai_status`:
  `used`, `not_configured`, `no_consent`, `owner_under_16`, `limit` (HTTP 429: free credits, rate or capacity limit; the client retries once after
  `Retry-After`, at most 5 s), `failed` (error, timeout 90 s, unusable answer), `nothing_to_plan`. Failures fall back to
  the rules and are logged without content: status plus Mistral's error type, code and (for 401/402/403/429/5xx)
  its own message, e.g. `journalctl -u reiterhof-api | grep "week plan"`.
- **Single day (JAN-95):** with `day=YYYY-MM-DD` exactly that day is open and the answer has one entry in `days`; all
  other days are context, so the week rules (variety, load, quotas) still hold. The day must be in the week, not over,
  not `done` and not the rest day after a show, else `400 validation_failed` "day cannot be planned"; a day somebody
  planned or claimed is allowed (`user` stays). The optional body carries the draft the owner sees for the other days
  (`draft`: at most 7 entries, dates in the week, an activity or `"rest"`, `minutes` 0-600; it only applies to days that
  would be open, closed days stay as they are) and `exclude`, the activities rejected for the day. They become
  `weekplan.Day.Minutes` (a planned unit counts with its minutes) and `Day.Exclude`: the prompt gets
  `planned_minutes` and `avoid`, a rejected model proposal is replaced by the rules (`replaced`: "Vom Besitzer
  abgelehnt."), the rules take the first recommendation that is not excluded, else a rest day ("Keine andere Aktivität
  passt heute."). Without `day`, body fields are ignored.
- **App:** "Woche planen" in `app/training/week.tsx` (only with `can_edit`), `components/training-plan-sheet.tsx`
  shows the days with the badge "KI-Vorschlag" or "Regel", level, focus and exercise; "Übernehmen" stores each day with
  `PUT /week/{day}` (`planned` with activity, focus, `exercise_id` and the claimed user, or `rest`; a rest day
  proposed for a claimed day is not stored, the claim stays) and a note such as
  "KI-Vorschlag: 45 Min. · …" (`lib/training-plan.ts`). The week rows show slot notes, the focus and the exercise
  (opens the exercise). With `no_consent` the owner gets
  "KI-Vorschläge erlauben" (consent sheet, then the plan is asked again).
- **Operator:** Mistral account settings and open contract questions in
  [avv-checkliste.md](../legal/avv-checkliste.md), Teil 1; privacy text section 3.12.

## Seed

`internal/seed/training.go`: profiles for all seven demo horses (the three minimal rows are filled
only while `discipline` is still empty, so local edits survive), a reha plan for Fanta (with abort
criteria), 20 sessions in the two weeks before the day of seeding and three planned week slots for Luna.

## Exercise library

Reference data, not demo data: migration `0250_exercise_catalog.up.sql` inserts the global library
(stable_id NULL, IDs `…0008YY`) on every installation, production included: 41 exercises, 17 dressage,
8 jumping, 10 groundwork, 6 lunging. Every row names the pages it is based on (SQL comments); nothing is
made up, a step without a figure in the sources leaves it out. Main sources: the FN member magazine
"Pferd und Mensch" (series "Lektion im Fokus", "10 Tipps", Springausbildung with figures from Richtlinien
Band 1/2), the FN Merkblätter for the badges Bodenarbeit, Vormustern and Longieren, de.wikipedia and
established riding magazines. Levels: beginner = basic training and Klasse E/A, intermediate = A* to L*,
advanced = L** and M (groundwork and lunging: the FN badge stages). Distances are for horses
(Großpferde). The app has no editor for global rows; corrections or new exercises go into a new
migration (`UPDATE`/`INSERT … ON CONFLICT` on the fixed IDs), never into 0250.

## Open points

- The tracker screens (`app/training/track/gps.tsx`, `indoor.tsx`, see [tracking.md](tracking.md)) hand `gait`,
  `rein` (JSON) plus `started_at` to `session/finish`. GPS tracks need the `location_tracking` consent:
  `POST /horses/{id}/sessions` with a non-empty `track` answers `403 consent_required` without it; indoor
  data and quick logs never do.
- `/horses/{id}/reha` (link "Reha-Plan", see [reha.md](reha.md)) and `/observations/new?horse=`
  ("Auffälligkeit melden") are built. The reha link only shows for horses in reha. The Training tab links
  the exercise library ("Übungsbibliothek").
- The weather temperature is the night minimum of the snapshot; a daytime value would need an
  extension of `weather_snapshots`.
