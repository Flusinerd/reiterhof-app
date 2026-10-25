# Training (M6)

Tickets: JAN-55 profile, JAN-56/62 logic (pure packages), JAN-57 "Was heute?", JAN-58 exercise
library, JAN-59 week view, JAN-60 "Nur eintragen", JAN-61 finish screen, JAN-89 week plan (rules and AI).

- Backend: `backend/internal/trainingapi` (HTTP + SQL). The rules live in the pure packages
  `internal/training`, `training/load` (load score, week segments, assessment) and
  `training/recommend` (recommender); see "Training logic" in [architecture.md](../architecture.md).
- Migration `0320_session_feeling.up.sql`: `sessions.feeling` (`hard|good|easy`, nullable), see "Sessions".
- Migration `0240_week_slot_focus.up.sql`: `week_slots.focus`, `week_slots.exercise_id` (JAN-92).
- Migration `0060_training_sessions.up.sql`: `sessions.exercise_id` and CHECK constraints for
  activity, feel and focus rating. Tables `training_profiles`, `exercises`, `sessions`, `week_slots`
  and `reha_plans` come from `0003`.
- Planning (M9): focuses, unit plans (Lösen, Arbeiten, Entspannen), four-week blocks, return to work, stable
  equipment and the level goal. The concept and the numbers are in [training-planning.md](training-planning.md), the
  specification; this file says what the backend does (see "Focuses and level goal", "Unit plan and blocks" and
  "Equipment"). The pure rules are in `training/recommend` (`PlanUnit`, `BlockFor`, structure groups 6 to 10) and
  `training/weekplan`; `training/focus.go` and `training/exercise.go` hold the catalog, the phases, the unit split,
  the equipment and the progress.
- Migration `0330_exercise_metadata.up.sql`: exercise `phase`, `minutes`, `intensity`, `requires`, `equipment`, `suits`,
  `summary`, `sources`; `exercises.discipline` now names the library. Migration `0350_training_focus.up.sql`:
  `training_profiles.focuses`, `cycle_start`, `goal_level`, `goal_until`, the table `rider_focuses`,
  `week_slots.focus_key`, `unit_plan`, `equipment_ok`, `sessions.focus_key` and `stables.training_equipment`.
- Mobile screens: `app/(tabs)/training.tsx` (Was heute?), `app/training/session.tsx` (chooser GPS/indoor, see [tracking.md](tracking.md)),
  `app/training/session/finish.tsx` ("Wie war's?", see "Finish step"), `app/training/week.tsx` (week with draft), `app/horses/[id]/training-setup.tsx`
  (profile wizard), `app/horses/[id]/training-profile.tsx` (overview) and the section editors
  `app/horses/[id]/training-profile/{activities,rhythm,structure,shows,riders,focuses}.tsx`, `app/stable/training-equipment.tsx`
  (the stable's equipment, admins, from the Stall tab) and the library `app/training/exercises/{index,[id]}.tsx`.
- Mobile layout (Abendlicht, see [design-system.md](../design-system.md)): the training tab (`app/training/index.tsx`) is the
  horse in the header (with the horse switcher), the recommendation as the one raised block with the primary "Starten", the
  duration pills, alternatives as rows, one week strip (`HorseWeekStrip`, the same as on the horse page) and links as rows.
  Without a profile "Profil einrichten" (`NextStepCard`) is the raised block and "Starten" steps back to an outline; "Woche
  planen" is a quiet row. The week (`app/training/week.tsx`) is a planning list of `WeekDayRow`s (big day number in Fraunces,
  who, activity, exercise; reha and show with time and classes underneath), "Woche planen" is its one primary action. The
  exercise library is `RowList` rows with filter pills; steps are `NumberedSteps` (numbers in Fraunces brass).
- Mobile, planning rebuild (specification: [training-planning.md](training-planning.md), "App"). Every new field is optional on read,
  so an older backend still works, and a new body field is only sent when the data had it (the API rejects unknown fields). Profile:
  the rows "Schwerpunkte" (`training-profile/focuses.tsx`: the catalog grouped "Skala der Ausbildung" and "Praxis" with label,
  description and signs, up to three focuses in priority order, optional "bis" date, the phase hint and the warnings, `turnier`
  starts with the next show of the profile) and "Ziel" (sheet: a class above the current level, optional "bis"; the progress sentence
  with the lections that come next; the `level_up` warning offers "Auf L einstufen" with a confirm sheet). Training tab: the raised
  block shows the focus (label, why) and the unit as three rows "15 Lösen · …" (minutes in Fraunces brass, "statt Trabstangen", a
  tap opens the exercise); the eyebrow names the block ("Aufbauwoche 2 von 4", "Entlastungswoche") or "Wiederantrainieren, Woche
  1"; quiet rows: "Schwerpunkte setzen" (owners, none set), "Mein Schwerpunkt" (sheet, max 2, "Kein eigener Schwerpunkt" clears),
  "Ausstattung im Stall eintragen" (admins while `set` is false) and the way to the level goal. Week: a day row shows the focus as
  a chip, the main exercise and `equipment_note`; the edit sheet has the focus picker ("Eigener Text" keeps the free text), the
  three unit parts (an exercise each, unavailable ones dimmed) and "Aufbau heute" (Standard, Möglich, Nicht möglich =
  `equipment_ok`, saved days only); the header shows the block. Finish ("Wie war's?") sends `focus_key`: the recommendation's
  (handed on through `session` and the trackers, also stored in the snapshot), else the focus of the day's slot when its activity
  is the one ridden. Library: the filters "Schwerpunkt" (matched by the server) and "Phase" (Lösen, Arbeiten, Entspannen), rows with
  library, minutes and "Ausstattung fehlt" (with `?horse=`), a detail with "Wozu", phase, minutes, intensity, the equipment as
  chips, prerequisites, "Alternativen ohne Aufbau", the progression and the sources (they open in the browser). The Heute timeline
  carries the focus label of a slot. PUT bodies: `focus_key` ("" removes), `unit` (`{warmup|main|cooldown: {exercise_id|null,
  minutes}}`), `equipment_ok` (true, false, null = default); saving a day sends them only when the owner changed them. A `unit`
  carries the main exercise in its main part, so no `exercise_id` goes beside it (the backend refuses one that differs). A saved
  planned day has no logged minutes: the editor takes its length from the parts of the unit, else from the note ("Plan: 45 Min.").
  A profile save sends `focuses` and `cycle_start` back as read (the focus editor sets `cycle_start`, the Monday of the current week, with the
  first focuses) and `goal_level`/`goal_until` only from the goal sheet and the level change.
- Mobile components: `components/training-next-step.tsx` (profile block, plan row), `training-week-day.tsx` (a day of the week),
  `training-dots.tsx` (`LoadBar`), `exercise-steps.tsx` (numbered steps),
  `training-day-edit-sheet.tsx` (edit one day), `training-activity-modes.tsx` (activity list, wizard and editor),
  `training-week-structure.tsx` (seven weekday rules), `training-rider-rules.tsx` (rider rules), `training-focus-rows.tsx` (catalog
  rows), `training-focus-sheet.tsx` ("Mein Schwerpunkt"), `training-goal-sheet.tsx` and `training-level-up-sheet.tsx` (level goal),
  `training-progress.tsx` (progress rows), `exercise-list-item.tsx` (library row); UI kit parts
  `ListRow`/`RowList`, `Stepper`, `NumberStepper`/`RangeStepper`, `DateField`.
- Mobile logic: `lib/session-finish.ts` (feeling, finish body, note target), pure helpers `lib/training*.ts` (`training-profile.ts`: drafts, per-section validation, discipline
  defaults, summaries; `training-plan.ts`: plan days, day edits and what the sheet does to them, `nextStep`; `training-focus.ts`: catalog
  labels and groups, phase hint, selection limits, warnings, unit split and rows, block text, the show a Turniervorbereitung ends
  with; `training-equipment.ts`: equipment labels, defaults, "Aufbau heute", availability; `training-goal.ts`: level goal, progress
  text, level up), `lib/tracking-exercises.ts` (library and tag labels, filters, sources), API client and React Query hooks
  `lib/api/training.ts`.

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
| `GET /training/focuses` | The focus catalog `{focuses: [{key, label, group, phase, description, signs, fits}]}` (static data, `training.Focuses`) |
| `GET /horses/{id}/training-profile` | Profile (`can_edit`, `rider_rules`); a horse without row returns the defaults with `exists:false`. Also `focuses`, `cycle_start`, `goal_level`, `goal_until`, `phase`, `focus_hints`, `warnings`, `progress`, see [Focuses and level goal](#focuses-and-level-goal) |
| `PUT /horses/{id}/training-profile` | Owner/admin only. Full replace; `rider_rules`, `focuses`, `cycle_start`, `goal_level` and `goal_until` omitted keep their stored value |
| `GET/PUT/DELETE /horses/{id}/my-focus` | The caller's own focuses for this horse, `{keys}` (at most 2); owner, admin and riders with `take_week_slots` or `log_sessions` |
| `GET/PUT /stables/training-equipment` | Facility equipment per location of the stable; every member reads, only admins write, see [Equipment](#equipment) |
| `GET /horses/{id}/today?minutes=45` | Recommendations with `focus` and `unit`, hidden activities, 7-day dots, context line, active reha phase; `phase`, `restart_week`, `block`, `progress` |
| `POST /horses/{id}/sessions` | Quick log or finished session, returns `{session, next_progression}`; optional `focus_key` (what the unit trained) |
| `GET /horses/{id}/sessions?limit=&before=` | Newest first; riders see `visible_to_rider` sessions and their own |
| `GET /horses/{id}/week?start=YYYY-MM-DD` | Monday to Sunday (any day of the week works as `start`), load segments, assessment; days carry `focus_key`, `focus_label`, `unit`, `equipment_ok`, `equipment_note`, the answer `block` |
| `PUT /horses/{id}/week/{day}` | Claim a day (`{}` = "Ich"), plan, rest day or release (`status: open`); optional `focus` (max 80) and `exercise_id` (global or own stable). Fields not sent (or `null`) keep their value, so "Ich" on a planned day keeps activity, note, focus and exercise; `focus: ""` / `exercise_id: ""` clear that field (JAN-95); a rest day clears activity, focus and exercise. Also `focus_key` (`""` clears, unknown is `400`), `unit` (explicit unit plan, validated exercise ids; `{}` clears) and `equipment_ok` (`true`, `false`, `null`); omitted keeps the stored value, and with a `unit` the slot's `exercise_id` is its main exercise. Returns the week (days carry `focus`, `exercise {id, title}`, `focus_key`, `unit`) |
| `POST /horses/{id}/week/plan?start=&day=` | Owner/admin only. Proposal for the open days of the week (rules, with the owner's consent a language model); stores nothing. With `day=YYYY-MM-DD` only that day is planned again; optional body `{draft: [{date, activity or "rest", minutes}], exclude: [activity]}` (empty body and `{}` are fine); entries add `focus_key`, `focus_label`, `focus_why`, `unit`, `equipment_ok`, `equipment_note`, the answer `block`; see [Week plan](#week-plan) |
| `GET /exercises?library=&level=&tag=&phase=&focus=&horse=` | Global (`stable_id NULL`) and own-stable exercises, easiest first along the progression; `library` (alias `discipline`), `phase` (warmup, main and cooldown also list `any`), `focus` (a catalog key: the exercises with its tags), `horse` adds `available` (does the stable have the equipment); items carry `library`, `phase`, `minutes`, `intensity`, `suits`, `summary`, `equipment` |
| `GET /exercises/{id}` | With `steps` and the follow-up exercise (`next_exercise_id`, `next`), `requires [{id, title}]`, `sources [url]` and `alternatives [{id, title, equipment}]` |

### Profile

`discipline` is one of `dressage, jumping, eventing, leisure, western, young_horse` (required).
`allowed_activities` entries are `{activity, mode: on|off|conditional, note}`; conditional needs a
note; activities missing from the list count as off. `shows` are `{date, time?, name, tests?, helper?}`:
`time` is the start (`HH:MM`), `tests` the classes (Prüfungen) the horse starts in, at most 20 `{name, time?}`
(name 1-80 characters, time `HH:MM`). Older app versions sent the classes as one free text `classes` (max 200);
the API still accepts and returns it, the app turns it into `tests` (split at `,`/`;`) and saves only `tests`.
Shows are sorted by date and time. Time, classes and helper feed the week view. `rhythm`: 1-7 sessions, 0-6 rest days, `max_minutes`
0-240 (0 = no limit), `rest_after_show`. Missing rhythm keys keep the defaults of
`training.DefaultRhythm()` (4-5 sessions, 1-2 rest days, 60 minutes, rest day after a show); when
decoding a stored profile the API always starts from that default. `status` is `fit|reha|pause`.

**Week structure** (JAN-93, `rhythm.days`, `rhythm.quotas`): `days` are seven rules, Monday first,
`{kind, activity?}` with `kind` one of `""` (free, the planner decides), `rest`, `recovery` (active recovery),
`light`, `normal`, `demanding` or `activity` (with an allowed `activity`). At most `rest_days_max` days may be
`rest`. `quotas` are units per week (Monday to Sunday): `{demanding, recovery, activities: {hall: 2, ...}}`, each
0-7, activities must be allowed, and all quota units plus `rest_days_min` must fit into seven days. `demanding`
and the activity quotas are also maximums, `recovery` is a minimum only. A horse has at most two demanding and
two jumping units a week: `demanding` is 0 to 2 and the quota of `jumping` at most 2, more is `400`. Like the other rhythm keys, missing
`days` or `quotas` (or `days: []`) mean the default: free days, no quotas. The app edits both in the section
"Feste Tage und Ziele" of the training profile (see "App" below).

**Unit levels** (`recommend.UnitLevel`) come from the load of a unit (`load.Score` without canter, i.e.
minutes × factor): active recovery below 20, light below 30, normal below 45, demanding from 45.

**Structure rules** (`recommend.structureFor`, applied by `Recommend`, `Check` and the week plan) come after
the safety rules (profile, rider rules, reha, show, pause, frozen ground) and turn into a rest day, activities
that are out and a minute window per activity. In order; a group is dropped when nothing would be left after
it, except the load limit, jumping, the load cap of the Entlastungswoche, the first week of the return to work and
the limits of young horses, which turn the day into a rest day or cut the minutes:

1. The owner's rule for the weekday (fixed rule, not a preference).
2. Default rules, only on free days: no demanding unit after a demanding day; after two days with at least
   normal load only a light unit (or less).
3. Needs of the week (week plan only): when the open days left, or the sessions `sessions_max` still allows,
   are needed for missing rest days (`rest_days_min`), quota units or, with three or more allowed activities, a
   third kind of activity, the day fills one of them.
4. Quota maximums: at most two demanding units a week (the quota can only lower it, a horse of 18 or older has one)
   and no more units of an activity than set.
5. Load limit: when the 14 days before the week have at least 3 sessions, the week's load may be at most
   1.2 × their weekly average (at least 90); when no unit fits, the day becomes a rest day.
6. Jumping: at most two units a week and never on two days in a row (also not before a unit fixed for the next
   day); when nothing else is left the day is a rest day ("Springen nicht an zwei Tagen hintereinander – heute
   Ruhetag.").
7. Variety: the activity of the day before, or of a unit fixed for the day after, is out when two other activities
   remain ("Halle war gestern schon dran – heute etwas anderes."); `Recommend` also scores doing it again lower.
8. Block: in the Entlastungswoche no demanding unit ("Entlastungswoche: diese Woche nichts Forderndes.") and the
   week's load at most 0.8 × the weekly average of the three weeks before.
9. Return to work (`Input.RestartFrom`): week 1 light units only, weeks 2 and 3 nothing demanding
   ("Wiederantrainieren, Woche 1: nur leichte Einheiten.").
10. Young horses (under 6): units of at most 30 minutes ("Junge Pferde trainieren höchstens 30 Minuten.") and at most
    4 units a week; this cap wins over a demanding day of the owner. Seniors (18 and older): one demanding unit a week.

**App** (JAN-94). A horse without a profile (`exists: false`) gets the **wizard** `training-setup` (owner and admin
only; everybody else and horses with a profile are sent to the overview). Five steps with a progress `Stepper`,
nothing is saved before the last one: discipline (with level and status), activities, rhythm, fixed weekdays
(skippable) and a summary. Choosing the discipline fills the activities, the rhythm numbers and the limit per unit
with that discipline's defaults (`disciplineDefaults` in `lib/training-profile.ts`; a new profile no longer starts
with every activity off); the later steps adjust them. "Profil anlegen" sends one `PUT` and opens the Training tab
for that horse. A step only continues when it is valid (a discipline, at least one activity, every conditional
activity with a condition, rest days within the maximum).

An existing profile opens the **overview** `training-profile`: the status (Fit, Reha, Pause) is changed with one
tap and saved immediately, below it one row per section with a one-line summary (discipline and level, activities,
rhythm, fixed days and goals, shows, rider rules when there are any). Each row opens an **editor**
(`training-profile/activities`, `rhythm`, `structure`, `shows`, `riders`; discipline and level in a sheet). An editor
works on a copy of the profile, checks only its own section (`validateSection`) and saves the whole profile with
"Speichern"; there is no auto-save because the `PUT` replaces the profile and half-typed values must not land.
Dates of shows and the end of the season are picked with `DateField` (no typing of `YYYY-MM-DD`), the start of a
show and the time of each class (Prüfung, "Prüfung hinzufügen") with `TimeField`. The week view shows the start
("ab 08:30 Uhr") and one line per class ("09:30 Dressur L"); a show day cannot be edited there (only claimed). A rider sees a
read-only text and the own rider rules ("Meine Regeln").

### Focuses and level goal

The focus catalog is static data, `training.Focuses` (15 focuses: the six points of the Skala der Ausbildung and
Durchlässigkeit, practice focuses such as Gelassenheit, Kondition, Springen and Gelände, and Turniervorbereitung
`turnier`); the planning spec lists them with tags and activities. `GET /training/focuses` returns
`{focuses: [{key, label, group: scale|practice, phase: 0-3, description, signs, fits: [activity]}]}`.

- **Horse phase** (`training.Phase(levelClass, ageYears, status)`): `gewoehnung`, `schubkraft` or `tragkraft`. The
  profile answer has `phase`, `focus_hints` (the scale focuses of the phase that are not chosen yet) and `warnings
  [{key, text}]` for chosen focuses above the phase ("Versammlung setzt Schwung und Geraderichtung voraus."); such a
  focus is accepted but planned at most once a week.
- **Focuses of the horse:** `focuses [{key, until}]` in the profile, at most 3, known and distinct keys, `until` a date
  (inclusive; a focus whose date is past is ignored by the planner) or absent. The first focuses that are saved set
  `cycle_start` to the Monday of the current week when it is empty (the start of the four-week blocks); an owner can
  send `cycle_start` (a date, moved to the Monday of its week) or `null` to clear it. A request without these fields keeps them.
- **Own focus** (`/my-focus`): `{keys: [key]}` (at most 2) per person and horse, the owner's and a rider's alike;
  `GET` and `PUT` answer `{keys, catalog_phase, phase}`, `PUT {"keys": []}` and `DELETE` (204) clear it. The planner
  puts the focuses of the person who rides in front of the horse's. They are part of the data export and are deleted
  with the account.
- **Level goal:** `goal_level` (`intermediate|advanced`, `null` = the class above the level of the profile) and
  `goal_until` (a date) are stored. The profile answer has `progress {level, goal_level, goal_until, mastered, total,
  next: [{id, title}], ready, text}` (null without a goal): of the lessons of the goal level in the horse's home
  libraries (phase main or any) `mastered` have a session rated "Sitzt", `next` are up to three unmastered lessons whose
  prerequisites sit, `ready` is at least 80 % of at least three lessons. When ready, `warnings` has `level_up` ("Luna
  hat 4 von 5 M-Lektionen gefestigt – Einstufung auf M anpassen?"). "Was heute?" carries `progress {text, next}`.

### Unit plan and blocks

A unit of hall, arena, lunge, jumping, groundwork or hack (not walker, not rest) has one **focus** (`focus_key`, a
catalog key) and three parts: Lösen (`warmup`), Arbeiten (`main`) and Entspannen (`cooldown`), each with minutes and
at most one exercise. The answers carry it as `unit {warmup, main, cooldown}`, each part `{exercise: {id, title} |
null, minutes, exercise_minutes?, why?, instead_of?: {id, title}}`; a part without minutes and exercise is absent.
`exercise_minutes` is the typical duration of the exercise when it is shorter than the part (the rest is free
work); `why` and `instead_of` say that the planner took another exercise than the one that fitted best, because of the
equipment or for a goal lesson. `week_slots.unit_plan` stores `{warmup: {exercise_id, minutes}, main: {...}, cooldown:
{...}}`; the slot's `exercise_id` stays the main exercise for older clients. `PUT /week/{day}` takes `unit` with
minutes 0-240 per part and 600 in all, exercises the stable can see, in its own shape or the answer's (`exercise
{id, title}`).

**Planner** (`recommend.PlanUnit`, pure; `weekplan.Merge` plans every day with it, "Was heute?" every recommendation):
the minutes of the parts come from `training.UnitSplit` (Lösen about a third, Entspannen a sixth of the unit, seniors
and nights below 5 °C add minutes); the focus from the rotation (the rider's, then the horse's focuses, the one trained
or planned longest ago first; a focus above the phase at most once a week; `turnier` within 14 days before a show;
a fallback per activity when none fits); the exercise of each part by score (tags of the focus, follow-up of a mastered
exercise, retry after "Schwer", used lately, intensity of the unit, goal lessons), among the exercises of the activity's
libraries (`training.ExerciseLibraries`) that fit the part, the horse's level, prerequisites, the day after cavaletti, the
week before a show and the facility equipment. Numbers and texts: [training-planning.md](training-planning.md), the comment
at the top of `training/recommend/unit.go`. Every focus comes with a German reason (`focus.why`, `focus_why`).

**Blocks:** four weeks from `training_profiles.cycle_start`: weeks 1 to 3 build, week 4 is the Entlastungswoche
(`block {week: 1-4, deload}` in `today`, `week` and `week/plan`; null without a start or before it). A block starts
again when one of its completed weeks so far had fewer sessions than `rhythm.sessions_min` (`recommend.BlockFor`).
**Return to work:** after 14 or more days without a session the first week has light units only, the weeks 2 and 3
nothing demanding (`restart_week` 1 to 3 in `today`, else 0; `recommend.RestartAnchor`). Both only change what is
planned, never what was logged.

### Equipment

`GET /stables/training-equipment` (every member) answers `{set, locations: {hall, arena, lunge, jumping, groundwork:
{equipment: [key], shared}}, keys: {facility, personal}, can_edit}`. Facility equipment (`poles`, `cavaletti`, `jumps`,
`obstacles`) is set up in a location: the planner only plans exercises whose facility equipment the location has.
Personal equipment (`lunge`, `side_reins`, `double_lunge`, `kappzaum`, `tarp`, `umbrella`, `cones`, `flag`, `whip`,
`neck_ring`, `trailer`) is always there and only listed on the exercises. A location the stable never saved has its
default: hall, arena and groundwork `poles`, jumping `jumps` and `poles`, lunge nothing, nothing shared; `set` is false
until the first `PUT`. `PUT {locations: {hall: {equipment: [...], shared}, ...}}` (admins only, `403` otherwise)
replaces the stored equipment: a location left out has its default, an unknown location or key is `400`.

`shared` means the location is used together with others: it is planned without set-up ("Halle geteilt: ohne Aufbau
geplant."). `equipment_ok` of a day (`PUT /week/{day}`: `true`, `false`, `null` = the location's default) overrides that:
`true` allows set-up in a shared location, `false` means nothing is set up that day ("Heute ohne Aufbau geplant.");
days and recommendations carry `equipment_ok` and `equipment_note`. When the best exercise needs equipment that is not
available the part carries `instead_of` and a reason ("Ohne Stangen in der Longierhalle: Handwechsel an der Longe statt
Trabstangen."). `GET /exercises?horse={id}` adds `available` to every item, `GET /exercises/{id}` lists `alternatives
[{id, title, equipment}]` (same library, phase equal or any, at least two shared tags, no facility equipment, a level at
or below, at most 5, most shared tags first).

### Today

Input for `recommend.Recommend`: profile, sessions of the last 14 days, the latest
`weather_snapshots` row for today (`Rain` = `will_rain`, `TempC` = forecast **night minimum**, so
the context line shows that temperature), the ground condition of the stable, `minutes` as
available time, and the role: owner/admin as `RoleOwner`, riders as `RoleRider` with the mapped
rules. All times are converted to the stable's timezone (`stables.timezone`) first; "today" and the
session days are calendar dates in that zone.

Context line: `Regen, 6 °C · Boden nass · Turnier in 3 Tagen · Reha: Phase 2`, each part only when
known (weather snapshot, ground not dry, next show within 14 days, active reha phase).

Besides the profile and the sessions the planner gets the age and level of the horse, the block of the week, the
return to work, the library with the horse's ratings, the stable's equipment, the day's `equipment_ok` (from its week
slot) and the own focuses of the person who asks. The sessions reach back to the Monday three weeks before this week
(Entlastungswoche, load limit); a session without a `focus_key` takes the focus and the exercises of the unit plan of
its day's slot.

Each recommendation for an activity with a unit carries `focus {key, label, why}`, `unit` (see "Unit plan and blocks"),
`exercise` (the main exercise of the unit, with steps), `equipment_ok` (only for activities with a location) and
`equipment_note`; all are null or absent for the walker and rest. On the day of a show there is no unit and no
exercise: the show is the work of the day. The answer also has `phase`, `restart_week`, `block` and `progress {text,
next}` (null without a goal level).

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

### Training tab

`app/(tabs)/training.tsx` (JAN-101), in this order: horse switcher (the start horse may come from `?horse=`, else
the first), page header with the last seven days, error card, **next-step card**, recommendation and alternatives,
**"Diese Woche"** strip, time available today, **"Mehr"** list.

- **Next step** (`nextStep` in `lib/training-plan.ts`, no backend field): only for owners and admins (`can_edit`).
  Without profile the card says "Profil einrichten" and opens the wizard; with a profile and at least one open day
  after today ("open" means the same days as the planner: no activity yet, today or later) it says "Woche planen"
  with the number of open days and opens `/training/week?horse=&plan=1`, which plans right away. Otherwise there is
  no card. A rider without profile sees only "Ohne Profil keine Empfehlung. Der Besitzer muss es anlegen."
- **Week strip:** seven circles for the current week (`GET /week`, cached for the week view): done (check), planned
  (activity icon), rest (moon), open (empty), past without training (grey); today has an accent outline. The strip
  and the "Woche" button open the week view.
- **Mehr:** rows "Übungen", "Trainingsprofil" and, only for horses with a reha plan or in status `reha`,
  "Reha-Plan".

### Sessions

Body: `activity`, `minutes` (1-600), optional `canter_share` (0-1), `started_at` (RFC 3339, default
now, at most 14 days back), tracked data `gait_shares` (`walk|trot|canter|halt`, fractions, sum at
most 1), `rein_changes` (ordered segments `{rein: left|right, minutes}`, the number of rein changes
is the segment count minus one), `distance_m`, and `feel` (`fresh|loose|tired|tense`),
`focus_rating` (1 Schwer, 2 Besser, 3 Sitzt), `exercise_id`, `note`, `visible_to_rider` (default
true). `source` is `tracked` when gait shares, rein changes or a distance are sent, else `quick`. `focus_key` (optional,
a key of the focus catalog, else `400`) says what the unit trained; sessions are returned with it (null without one)
and the focus rotation reads it.

`feeling` (`hard|good|easy`, "Schwer", "Gut", "Sehr locker"; optional, `null` for quick logs and older app
versions; anything else is `400`) is how the session felt for the rider, asked by "Wie war's?" after the ride.
It is separate from `feel`, which is how the horse was and what the recommender and the week plan read; the
app no longer asks for `feel` but the API still takes it. Sessions are returned with both.

### Finish step "Wie war's?"

`app/training/session/finish.tsx` (after both trackers, `lib/session-finish.ts` for the logic), top to bottom:
eyebrow "Luna · gerade eben" and the title; the minutes from the tracker as a big number the rider can edit
("Minuten in der Halle, aus dem Tracking"; "Ort ändern" switches the activity), gait and rein summary when
tracked; "Gefühl" chips (Schwer, Gut, Sehr locker, tap again to clear); "Übungen" with the exercise of the
plan as one tick row; the note ("Notiz für Mia …" when someone else owns or rides the horse, else "Notiz …"),
for an owner with a rider the switch "Mia sieht die Einheit" (`visible_to_rider`); the one button "Ins
Tagebuch", and a quiet "Etwas aufgefallen?". Without a next progression the screen closes after saving,
with one it shows "Als Nächstes" first.

- The exercise row starts out ticked when the indoor tracker ticked at least one step (the tracker passes
  `steps_done`, e.g. `0,2`, or `-` for none, in the finish params) and when no tracker data exists. The
  caption says "2 von 4 Schritten erledigt". Unticked: no `exercise_id` is sent.
- The feeling also rates the exercise when it is sent (`focus_rating` 1, 2, 3 for hard, good, easy), so the
  suggestion of the next step ("Sitzt") keeps working without a second question.
- Where the minutes come from: `minutes` of the finish params, at least 1 (`secondsToMinutes`); a typed value
  must be a whole number from 1 to 600, otherwise the button stays disabled.

A non-empty `track` (GPS) needs the `location_tracking` consent, else `403 consent_required`
(`privacy.Has`, JAN-19); see [privacy.md](privacy.md).

`load_score` is `load.Score(minutes, activity, canterShare)`, stored in `sessions.load_score`; the
canter share is `gait_shares.canter`. Saving marks the week slot of that day (stable-local) as
`done` (created if nobody planned the day; an existing planner stays). With `focus_rating: 3` and an
`exercise_id`, the response contains `next_progression` (the exercise's `next_exercise_id`).

### Week

Each day has a `status`: `done` (a session exists), `today`, `planned` (someone claimed a future
day), `open` ("Niemand eingetragen"), `empty` (past, nothing) or `rest` (`rest_reason`:
`after_show` fixed, `planned`). Show days carry `show {name, time, tests, classes, helper}`. A show day that
is not done has no training unit: the week leaves out activity, focus and exercise of its slot (they come back
when the show is removed), the claim (who goes to the show) and the note stay, and `PUT /week/{day}` with an
`activity`, `focus` or `exercise_id` for that day answers `400 validation_failed`; days covered by an active reha plan carry `reha {plan_id, phase,
activity, activity_label, rest, minutes, done}`. `segments` are the 7
load levels and `assessment` is the German sentence from `load.Assess`. Hidden sessions (riders) still
count for status and load but reveal neither user nor activity. Days are calendar days in the
stable's timezone, so DST weeks still have seven days. A planned day carries `focus_key`, `focus_label`, `unit`,
`equipment_ok` and `equipment_note` (null otherwise; a show day, a rest day and a hidden session show none of them);
the answer has `block {week, deload}`.

## Week plan

`POST /horses/{id}/week/plan?start=YYYY-MM-DD` (JAN-89, owner and admins; riders and members `403`). The handler is
`trainingapi/plan.go`, the logic the pure package `training/weekplan`; the Mistral client is `internal/mistral`.

- **Open days:** today or later and no activity planned: status `open`, `today`, or `planned` by someone without an
  activity (the proposal keeps that person, `user`). Done days, show days, rest days (planned or after a show) and
  days with a planned activity are context; a planned activity counts as a session for the following days.
- **Rules:** `weekplan.Rules` asks `recommend.Recommend` day by day with the sessions of the 21 days before plus the
  units planned so far (simulated sessions with their focus and exercises), so variety, recent load, the weekly
  maximum, the focus rotation and the exercise pick apply across the week; every unit is planned with
  `recommend.PlanUnit`. Weather and ground are only known for today and tomorrow. The active reha plan gives each day
  its unit. A day claimed by a person with own focuses is planned with them, days planned by someone else count with
  their focus and exercises.
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
  libraries that fit the allowed activities (`training.ExerciseLibraries`), in progression order, with short keys
  `E1`, `E2`, ..., level, phase, minutes, intensity, title, tags, the keys of the prerequisites, the facility equipment,
  mastered, practised and the key of the follow-up exercise. It also sends the horse's phase, the goal level, the block,
  `restart_week`, the focus catalog (key, tags, activities), the horse's focuses in order (with `above_phase`), the
  focuses of the person who claimed a day (keys), the facility equipment and the equipment that is available per
  activity (`shared`), `equipment_ok` per day and the focus key of the sessions. No names, ids, free text or dates; a test
  checks this.
- **System prompt** (`weekplan.SystemPrompt`, JAN-92): role (riding instructor in the FN system, welfare first),
  the input, every activity with German name, meaning, intensity and sensible duration, then the planning rules:
  unit levels, the week structure (the owner's rules and quotas are binding; on free days no two demanding days in a
  row and a light unit after two days of normal or demanding load; at most two demanding units, one for seniors;
  rest days minimum, active recovery, load limit and the Entlastungswoche, variety, jumping and cavaletti limits,
  return to work, young and senior horses), the focus of a unit (the rotation, fallbacks, the training scale
  Skala der Ausbildung), the unit of three parts with an exercise each (libraries, level, prerequisites, equipment,
  the week before a show), discipline, shows, weather, reha and pause; rules for the German reason (third person,
  concrete cause, no "du") with examples; a checklist before answering. The answer is
  `{"days":[{"day","activity","minutes","focus_key","focus","warmup","main","cooldown","reason"}]}`. `weekplan.Parse`
  drops unknown or closed days and unknown activities; an unknown focus key becomes no focus and an unknown exercise
  key no exercise (the rules pick); the key `exercise` of the earlier format counts as `main`.
- **Checking:** `recommend.Check` tests each proposal against the hard rules (visibility, reha phase, rest after a
  show, pause/reha/show/frozen-ground filters, `rhythm.sessions_max`, the structure rules above) and fits the minutes (reha range, rhythm
  maximum, the activity's sensible range from `recommend.MinutesRange`, e.g. lunge 15-30, hall 30-60, hack 30-120,
  pause 20, before a show 30). A failing proposal is replaced by the rules (`replaced` says why);
  days the model left out come from the rules. The unit of an accepted day is planned by `recommend.PlanUnit`: a
  proposed focus key is taken when it is one of the rotation's candidates, a proposed exercise when the rules allow
  it for its part (library of the activity, phase, level, prerequisites, equipment, the week before a show, and for
  the main part tags that meet the focus, no exercise twice); otherwise the rules pick and `replaced` says why
  ("Übung „Volte“ ersetzt: braucht Stangen."). Days from the rules get the rules' focus and exercises.
- **Response:** `{horse_id, start, end, source: ai|rules, ai_status, owner_is_me, days: [{date, weekday, activity
  (or rest), label, minutes, intensity, intensity_label, reason, note, source, replaced, user, focus, exercise
  {id, title}, focus_key, focus_label, focus_why, unit, equipment_ok, equipment_note, level, level_label}], block}`
  (`level` is the unit level, absent for rest; `exercise` is the main exercise of the unit; `focus_*` and `unit` are
  null for rest and walker; `block` is null without a block start). `ai_status`:
  `used`, `not_configured`, `no_consent`, `owner_under_16`, `limit` (HTTP 429: free credits, rate or capacity limit; the client retries once after
  `Retry-After`, at most 5 s), `failed` (error, timeout 90 s, unusable answer), `nothing_to_plan`. Failures fall back to
  the rules and are logged without content: status plus Mistral's error type, code and (for 401/402/403/429/5xx)
  its own message, e.g. `journalctl -u reiterhof-api | grep "week plan"`.
- **Single day (JAN-95):** with `day=YYYY-MM-DD` exactly that day is open and the answer has one entry in `days`; all
  other days are context, so the week rules (variety, load, quotas) still hold. The day must be in the week, not over,
  not `done`, not a show day and not the rest day after a show, else `400 validation_failed` "day cannot be planned"; a day somebody
  planned or claimed is allowed (`user` stays). The optional body carries the draft the owner sees for the other days
  (`draft`: at most 7 entries, dates in the week, an activity or `"rest"`, `minutes` 0-600; it only applies to days that
  would be open, closed days stay as they are) and `exclude`, the activities rejected for the day. They become
  `weekplan.Day.Minutes` (a planned unit counts with its minutes) and `Day.Exclude`: the prompt gets
  `planned_minutes` and `avoid`, a rejected model proposal is replaced by the rules (`replaced`: "Vom Besitzer
  abgelehnt."), the rules take the first recommendation that is not excluded, else a rest day ("Keine andere Aktivität
  passt heute."). Without `day`, body fields are ignored.
- **App:** the plan is a draft in the week view (`app/training/week.tsx`, only with `can_edit`; JAN-94), not a
  separate sheet. "Woche planen" (or `?plan=1` from the Training tab, once on opening) asks for the proposal and shows
  it in the day rows with the badges "Entwurf" and "KI" or "Regel", level, focus and exercise; nothing is stored
  yet. Week arrows and the horse switcher are locked until the draft is applied or discarded. Each row opens the
  per-day edit sheet (`components/training-day-edit-sheet.tsx`): activity or rest day, minutes, focus, exercise and
  (for saved days) the person. In a draft it changes only the draft ("Fertig"); on a saved, not yet done day it
  changes the week at once ("Speichern") or releases the day ("Freigeben"). "Neu vorschlagen" asks for that one day
  again (`?day=`) with the other draft days as context and the current activity excluded. "Übernehmen (n Tage)"
  stores each day with `PUT /week/{day}` (`planned` with activity, focus, `exercise_id` and the claimed user, or
  `rest`; a rest day proposed for a claimed day is not stored, the claim stays); "Verwerfen" drops the draft. Days
  from the draft or from the edit sheet get a note such as "Plan: 45 Min." or "Plan: Ruhetag" (`lib/training-plan.ts`).
  The week rows show slot notes, the focus and the exercise (opens the exercise). With `no_consent` the owner gets
  "KI-Vorschläge erlauben" (consent sheet, then the plan is asked again).
- **Operator:** Mistral account settings and open contract questions in
  [avv-checkliste.md](../legal/avv-checkliste.md), Teil 1; privacy text section 3.12.

## Seed

`internal/seed/training.go`: profiles for all seven demo horses (the three minimal rows are filled
only while `discipline` is still empty, so local edits survive), a reha plan for Fanta (with abort
criteria), 20 sessions in the two weeks before the day of seeding (all but the walker's with the `focus_key` they
trained) and three planned week slots for Luna, two of them with a focus and a unit plan. Luna has the focuses
`losgelassenheit` and `geraderichten`, Fanta `kondition`, both the current Monday as `cycle_start` (the demo starts
in "Aufbauwoche 1 von 4"); Mia has the own focus `gelassenheit` for Luna; the demo stable has hall `poles, cavaletti`, arena
`poles`, lunge nothing, jumping `jumps, poles` and groundwork `poles, obstacles`, none of them shared. Focuses, block
start, equipment and the unit plans are only filled while they are still empty, so local edits survive; Luna keeps the
default level goal.

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

Migration `0280_leg_yield.up.sql` (first correction): Schenkelweichen (804) and "Viereck verkleinern und
vergrößern" (813) get the aim `seitengaenge` (before, the filter "Seitengänge" and the AI plan's tag
matching skipped them), and the jumping library gets its own Schenkelweichen (846, beginner, followed by
Trabstangen), because hall and arena days of jumping horses use the jumping library and had no leg-yield.
The library now has 42 exercises (9 jumping).

Migration `0330_exercise_metadata.up.sql` (planning, M9): the column `discipline` now names the **library**
(`dressage`, `cavaletti`, `jumping`, `western`, `groundwork`, `lunge`, `hack`; poles and cavaletti are their own
library) and every exercise has a `phase` (`warmup`, `main`, `cooldown` or `any`: the part of the unit), `minutes`
(typical duration), `intensity` (`light`, `normal`, `demanding`), `requires` (prerequisite exercises), `equipment`
(keys of `training.EquipmentKeys`), `suits` (`young`, `senior`), a one-sentence `summary` and its `sources` (urls). Which
libraries an activity uses: `training.ExerciseLibraries(activity, horseDiscipline)` (hall and arena: dressage and
cavaletti, for jumping horses jumping, cavaletti and dressage, for western horses western, cavaletti and dressage;
jumping: jumping and cavaletti; lunge: lunge and cavaletti; groundwork; hack; walker and rest none), the first one is
the home library. Catalog rows added later and their coverage: [exercise-catalog.md](exercise-catalog.md).

## Open points

- The tracker screens (`app/training/track/gps.tsx`, `indoor.tsx`, see [tracking.md](tracking.md)) hand `gait`,
  `rein` (JSON) plus `started_at` to `session/finish`. GPS tracks need the `location_tracking` consent:
  `POST /horses/{id}/sessions` with a non-empty `track` answers `403 consent_required` without it; indoor
  data and quick logs never do.
- `/horses/{id}/reha` (link "Reha-Plan", see [reha.md](reha.md)) and `/observations/new?horse=`
  ("Auffälligkeit melden") are built. The Training tab shows the reha link only for horses in reha and links
  the exercise library ("Übungen") and the profile under "Mehr".
- The weather temperature is the night minimum of the snapshot; a daytime value would need an
  extension of `weather_snapshots`.
