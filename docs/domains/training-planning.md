# Training planning: focuses, unit plans, catalog (M9)

The rebuild of the training planning decided with the product owner in October 2026 (research and
decision: the concept page "Stallfunk Trainingslehre"; this file is the specification). It extends
[training.md](training.md); where the two disagree, this file wins until training.md is updated.

## Basis

Four layers, each from a source that is the accepted standard for its question:

1. **Content: Skala der Ausbildung** (FN Richtlinien Band 1; the FEI and the EWU western guidelines
   use the same points). Takt, Losgelassenheit, Anlehnung, Schwung, Geraderichtung, Versammlung;
   Durchlässigkeit is the goal over all of them. Three overlapping phases: *Gewöhnungsphase* (Takt,
   Losgelassenheit, Anlehnung), *Entwicklung der Schubkraft* (Losgelassenheit to Geraderichtung),
   *Entwicklung der Tragkraft* (Schwung to Versammlung). Every unit has exactly one focus.
2. **Mix of the week: vielseitige Grundausbildung** (FN Band 1, Klimke): dressage, poles and
   cavaletti, gymnastic jumping and hacking for every horse the profile allows it for; never two
   identical units in a row.
3. **Load: training science** (reviews in Equine Veterinary Education 2026 and JEVS 2017, Clayton,
   Burger): overload and recovery, progressive loading, periodisation, individuality. Heart and lungs
   adapt in weeks, muscles in months, tendons, ligaments and bone in one to two years.
4. **Unit: the three parts of a riding lesson** (Lösen, Arbeiten, Entspannen) and the ISES learning
   principles for the wording of exercise steps (one thing at a time, a light signal before pressure,
   immediate release, no punishment).

### Rules with numbers

Rules marked *(exists)* are already implemented (training.md); the others are new.

Week:

- Rest days per the rhythm *(exists)*; after two days with at least normal load a light unit *(exists)*;
  no demanding day after a demanding day *(exists)*.
- **Demanding units at most 2 per week** by default (an owner's quota may set 0 to 2; more than 2 is refused).
- **Variety**: the same activity never on two consecutive days when at least two other allowed activities
  fit that day (hard rule in `Check`, strong penalty in `Recommend`); when three or more activities are
  allowed, the week needs at least three different ones (a need like the quotas).
- **Hack**: when `hack` is allowed and the week has none yet, an open day prefers it (bonus).
- **Poles and cavaletti** 1 to 2 units per week, never on consecutive days (handled by the exercise pick:
  the `cavaletti` library is excluded the day after it was used).
- **Jumping** at most 2 per week, never consecutive (hard rule; default activity quota 2 for `jumping`).
- **Load limit** 1.2 × the average of the two weeks before *(exists)*.
- **Blocks of four weeks** from `cycle_start`: weeks 1 to 3 build, week 4 is the *Entlastungswoche*: no
  demanding unit and the load capped at 0.8 × the average of the three weeks before. The block is reset
  (week 1 again, `cycle_start` moved) when one of the three weeks before had fewer than
  `sessions_min` sessions.
- **Return to work** after 14 or more days without a session: the first week only light and recovery
  units, the second at most normal, demanding from the fourth week on.
- **Shows** *(exists: last demanding unit 2 to 3 days before, light the day before, rest after)*; new: in
  the 7 days before a show no new exercise (only exercises the horse has practised: a session rated
  anything), the focus is the horse's first focus.
- **Young horses** (under 6): units at most 30 minutes, at most 4 units per week, focus phase
  Gewöhnung only, groundwork, lunge and calm hacks preferred (already in the prompt).
- **Seniors** (18 and older): warm-up and cool-down 5 minutes longer, at most 1 demanding unit per week.

Unit (the "Stundenbild"): for a unit of `T` minutes of activity `a`:

| Activity | Lösen | Arbeiten | Entspannen |
| --- | --- | --- | --- |
| hall, arena, jumping | clamp(round(T × 0.33), 10, 25), jumping at least 15 | the rest, at least 10 | clamp(round(T × 0.17), 5, 10) |
| lunge | 10 (walk, no side reins) | the rest | 5 |
| groundwork | 5 | the rest | 5 |
| hack | 10 (walk) | the rest | 10 (walk) |
| walker, rest | no unit plan | | |

Seniors and a night minimum below 5 °C add 5 minutes to Lösen (capped at 30); the time comes off
Arbeiten. Each part gets at most one exercise of the matching phase (see "Exercise pick").

## Focus catalog

Static Go data (`training.Focuses`, `backend/internal/training/focus.go`) served by
`GET /api/v1/training/focuses`. Keys are stable identifiers (ASCII), labels and texts German.

| Key | Label | Group | Phase | Tags matched | Fits activities |
| --- | --- | --- | --- | --- | --- |
| `takt` | Takt | scale | 1 | takt, rhythmus | hall, arena, lunge, hack |
| `losgelassenheit` | Losgelassenheit | scale | 1 | losgelassenheit, biegung | hall, arena, lunge, hack |
| `anlehnung` | Anlehnung | scale | 1 | anlehnung | hall, arena, lunge |
| `schwung` | Schwung | scale | 2 | schwung | hall, arena, hack |
| `geraderichten` | Geraderichtung | scale | 2 | geraderichten, biegung, seitengaenge | hall, arena |
| `versammlung` | Versammlung | scale | 3 | versammlung | hall, arena |
| `durchlaessigkeit` | Durchlässigkeit | scale | 2 | durchlaessigkeit | hall, arena, lunge |
| `seitengaenge` | Seitengänge | practice | 2 | seitengaenge | hall, arena, groundwork |
| `gelassenheit` | Gelassenheit | practice | 0 | gelassenheit, vertrauen | groundwork, hack, arena |
| `kondition` | Kondition | practice | 0 | kondition, ausdauer | hack, arena, hall |
| `springen` | Springen | practice | 0 | springgymnastik, rhythmus | jumping, hall, arena |
| `gelaende` | Gelände | practice | 0 | gelaende, gelassenheit, kondition | hack |
| `vertrauen` | Vertrauen | practice | 0 | vertrauen, aufmerksamkeit | groundwork, lunge |
| `mobilitaet` | Beweglichkeit | practice | 0 | mobilitaet, losgelassenheit | hall, arena, lunge, groundwork, hack |
| `turnier` | Turniervorbereitung | practice | 0 | turnier, durchlaessigkeit, aufmerksamkeit | hall, arena, jumping |

Phase 0 = fits every phase. Each entry also has `description` (what it is, one sentence) and `signs`
("Woran du es merkst", one sentence), written from the FN definitions.

**Turniervorbereitung** (`turnier`) is the one focus with date logic: an owner picks it with `until` set to
the show date (the editor prefills the next show from the week plans), and the planner adds it by itself
as the last candidate in the 14 days before a show when it is not set (see "Focus rotation"). Its
exercises are the rehearsals (tag `turnier`): riding the dressage test of the class, warming up as on
the show ground, entering, halting and saluting, course sections and a full course at show height.

**Horse phase** (`training.Phase(levelClass, ageYears, status)`): `gewoehnung` when the level class is
beginner or unknown, the horse is under 6, or the status is not `fit`; `schubkraft` for intermediate;
`tragkraft` for advanced. Focuses allowed by phase: gewoehnung = phase 0 and 1; schubkraft adds 2;
tragkraft adds 3. The API accepts a focus above the horse's phase but returns a `warning` for it
("Versammlung setzt Schwung und Geraderichtung voraus."), and the planner schedules such a focus at most
once per week.

**Exercise tags** (`exercises.goal_tags`) are the fixed list: takt, rhythmus, losgelassenheit, biegung,
anlehnung, schwung, geraderichten, versammlung, durchlaessigkeit, seitengaenge, balance, springgymnastik,
aufmerksamkeit, vertrauen, gelassenheit, kondition, ausdauer, kraft, gelaende, mobilitaet, turnier. Unknown
tags are refused by a test over the catalog.

## Data model

### Migration `0330_exercise_metadata.up.sql` (done, on the branch)

`exercises` gets `phase` (`warmup|main|cooldown|any`, default `main`), `minutes` (typical duration,
nullable), `intensity` (`light|normal|demanding`, default `normal`), `requires uuid[]` (prerequisite
exercises, all must be mastered before the exercise is planned), `equipment text[]`, `suits text[]`
(`young`, `senior`), `summary text` (one sentence "Wozu") and `sources text[]` (URLs or titles). The
`discipline` column keeps its name but now means the **library**: `dressage`, `jumping`, `cavaletti`,
`groundwork`, `lunge`, `hack`, `western`. The pole rows 820, 821, 823, 824 moved to `cavaletti`. The 42
existing rows got their metadata and their sources (from the SQL comments of 0250/0280).

### Migration `0340_exercise_catalog_2.up.sql` (catalog agent)

About 110 new global exercises, IDs `00000000-0000-4000-8000-0000000008YY` / `…0009YY` in these ranges:
dressage 850–889, cavaletti 890–905, jumping 906–919, groundwork 920–939, lunge 940–949, hack 950–969,
western 970–989, 990–999 reserved. Every row: library, level, goal_tags (fixed list), title, summary,
steps (3 to 5 German sentences in the imperative style of 0250), phase, minutes, intensity, requires,
equipment, suits, sources. Progressions with `UPDATE … next_exercise_id` as in 0250. Rules:

- Nothing is made up: every figure (distances, counts, minutes) comes from a named source; a step
  without a figure in the sources leaves the figure out. Sources go into the `sources` column (and may
  stay as SQL comments).
- Levels as in 0250 (beginner E/A, intermediate A*–L*, advanced L** and up; badge stages for groundwork
  and lunge). Western levels by the Kreinberg/EWU stages (Grundausbildung, Gymnastizierung, Manöver).
- Wording follows the ISES principles: one aid at a time, release named, no punishment, no "muss".
- Required coverage: every focus key has at least 3 main exercises per phase-fitting library at beginner
  level; every library has at least 2 warm-up and 1 cool-down exercise; `hack` has a progression chain for
  Kondition (walk, uphill walk, steady trot stretch, trot and canter stretches, intervals); `cavaletti` has
  walk, trot (straight and bent), canter and lunge variants; `groundwork` has the calmness stations
  (tarp, flutter curtain, umbrella, bridge, narrow passage, pole labyrinth) and leading/trailer loading;
  `western` has the Kreinberg basics (stop, back-up, turns on fore- and hindquarters, sidepass, circles with
  speed control, lope transitions, trail obstacles) for western horses on hall and arena days; young and
  senior variants are tagged in `suits`; `turnier` has at least 4 exercises in `dressage` (the E, A and L
  tests ridden through from the FN Aufgabenheft, halting and saluting, warming up as on the show ground)
  and at least 3 in `jumping` (course sections, a full course at E/A height, a related distance ridden as
  in a course), all `main`, none `demanding` except the full course.

### Migration `0350_training_focus.up.sql` (backend agent)

- `training_profiles.focuses jsonb NOT NULL DEFAULT '[]'`: `[{key, until?: "YYYY-MM-DD"}]`, at most 3,
  priority = order. `training_profiles.cycle_start date` (Monday of the first block week).
- `rider_focuses (stable_id, horse_id, user_id, keys text[], updated_at, UNIQUE (horse_id, user_id))`:
  the rider's own focuses, at most 2.
- `week_slots.focus_key text`, `week_slots.unit_plan jsonb` (`{warmup: {exercise_id, minutes}, main: {…},
  cooldown: {…}}`; `exercise_id` stays the main exercise for older clients).
- `sessions.focus_key text` (what was trained; drives the rotation).

## API

All under `/api/v1`, access as in training.md.

| Route | Change |
| --- | --- |
| `GET /training/focuses` | `{focuses: [{key, label, group, phase, description, signs, fits: [activity]}]}` |
| `GET /horses/{id}/training-profile` | adds `focuses [{key, until}]`, `cycle_start`, `phase` (`gewoehnung|schubkraft|tragkraft`), `focus_hints [key]` (the scale keys of the phase that are not chosen yet), `warnings [{key, text}]` |
| `PUT /horses/{id}/training-profile` | accepts `focuses` (max 3, known keys, distinct, `until` a date or absent) and `cycle_start` (optional; set to the Monday of the current week when focuses are first set and it is empty) |
| `GET/PUT/DELETE /horses/{id}/my-focus` | `{keys: [key]}` (max 2) for owner, admin and riders with `take_week_slots` or `log_sessions`; GET returns `{keys, catalog_phase}` |
| `GET /horses/{id}/week` | days add `focus_key`, `focus_label`, `unit {warmup, main, cooldown}` (each `{exercise: {id, title} \| null, minutes}` or the part absent), `block {week: 1..4, deload: bool}` on the response |
| `PUT /horses/{id}/week/{day}` | accepts `focus_key` ("" clears; unknown key 400) and `unit` (optional explicit plan; validated exercise ids); `focus` free text stays |
| `POST /horses/{id}/week/plan` | entries add `focus_key`, `focus_label`, `unit`, `block`; `focus` (model's words) stays |
| `GET /horses/{id}/today` | recommendations add `focus {key, label, why}` and `unit`; response adds `phase`, `restart_week` (0 when not returning to work), `block` |
| `POST /horses/{id}/sessions` | accepts `focus_key` (optional, known key); stored, returned |
| `GET /exercises` | filters `library` (alias of `discipline`), `level`, `tag`, `phase`, `focus` (= the focus's tags); items carry `library`, `phase`, `minutes`, `intensity`, `suits`, `summary`, `equipment` |
| `GET /exercises/{id}` | adds `requires [{id, title}]`, `sources [string]` |

`training.ExerciseLibraries(activity, horseDiscipline) []string` replaces `ExerciseLibrary`: hall and
arena → dressage + cavaletti (jumping horses: jumping + cavaletti + dressage; western horses: western +
cavaletti + dressage); jumping → jumping + cavaletti; lunge → lunge + cavaletti; groundwork → groundwork;
hack → hack; walker and rest → none. The first library is the "home" library for the main exercise.

## Planner

### Focus rotation (`training/recommend`, `training/weekplan`, pure)

Input per day: the horse's focuses (priority order), the rider's focuses when the day is claimed by a
rider who set some, the sessions of the last 14 days with `focus_key`, the units planned so far in the
week with their focus. For every unit that is not rest or walker:

1. Candidates = rider focuses (in order) then horse focuses (in order), filtered to those whose `fits`
   contains the activity; a focus above the horse's phase is dropped when it was planned or trained in the
   last 7 days. When a show is planned within the next 14 days and `turnier` is not among the focuses, it
   is appended as the last candidate for hall, arena and jumping units, at most twice a week.
2. Pick the candidate with the most days since it was last trained or planned (never = infinity); ties by
   list order.
3. No candidate: fallback by activity: hall, arena, lunge → `losgelassenheit`; groundwork → `vertrauen`;
   hack → `gelaende` when the horse has it, else `kondition`; jumping → `springen`.
4. The `why` text names the cause: "Schwerpunkt Takt: zuletzt am Montag." / "Mias Schwerpunkt Gelassenheit."
   / "Ohne Schwerpunkt im Profil: Losgelassenheit kommt immer zuerst." / "Turnier am 28.10.: Aufgabe und
   Parcours üben."

### Exercise pick (per phase)

Candidates: the libraries of the activity; phase `warmup` or `any` for Lösen, `main` or `any` for
Arbeiten, `cooldown` or `any` for Entspannen; level at or below the horse's level class (one above only
when every easier exercise of that library is mastered); `requires` all mastered; `cavaletti` rows
excluded when a cavaletti exercise was used the day before; in the 7 days before a show only exercises
with any session rating or tagged `turnier`. Score (highest wins, ties by progression order, then title):

- Arbeiten: +3 per tag shared with the focus's tags (0 shared = not a candidate unless nothing else fits);
  +2 when it is the `next` of a mastered exercise; +2 when rated "Schwer" 3 to 7 days ago; +1.5 when rated
  "Besser" in the last 14 days; −2 when mastered; −3 when used in the last 3 days; −2 when `intensity`
  is `demanding` and the unit level is light or recovery, or `light` and the unit is demanding; +1 when
  `suits` matches young or senior.
- Lösen: +2 per tag in {losgelassenheit, takt, mobilitaet}; −3 when used yesterday; +1 for `suits`.
- Entspannen: +2 per tag in {losgelassenheit, mobilitaet}; prefer `cooldown` over `any` (+1).

Mastered = at least one session with `focus_rating` 3 *(exists)*. Minutes per part from the table
above; an exercise's own `minutes` caps its part (the rest is free work in that part).

### Week structure additions (`recommend/structure.go`)

Groups in this order after the existing 1 to 5: 6. default jumping quota 2 and jumping never on
consecutive days; 7. variety (consecutive same activity out when two other fits remain; "three different
activities" as a need); 8. block: deload week → demanding out, load cap 0.8 × average of the three weeks
before; 9. return to work: week 1 light/recovery only, week 2 at most normal, week 3 at most normal,
demanding from week 4; 10. young: minutes ≤ 30, units ≤ 4; senior: demanding quota 1. Reasons in German
as the existing ones ("Entlastungswoche: diese Woche nichts Forderndes."; "Wiederantrainieren, Woche 1:
nur leichte Einheiten.").

### Language model (`weekplan/prompt.go`)

The prompt gets: `phase`, `block {week, deload}`, `restart_week`, `focus_catalog` (key, tags, fits),
`horse_focuses` (keys in order, with `above_phase` flags), per day `rider_focuses` (keys) when the day is
claimed by a rider with focuses, exercises with `phase`, `minutes`, `intensity`, `requires` (keys) and
`practised` (any rating). The system prompt gets the four layers and the rules above (replace the
"HOW TO PLAN" and "FOCUS AND EXERCISE" sections). The answer per day: `focus_key` (catalog key or null),
`warmup`, `main`, `cooldown` (exercise keys or null), `focus` (2 to 5 German words, optional), `reason`.
`Parse` maps unknown focus keys to none (the rules pick) and unknown exercise keys to null;
`Merge` fills missing parts with the exercise pick, and `Check` still validates activity and minutes.
A proposed main exercise is dropped when its tags do not meet the focus or its prerequisites are not
mastered (rules pick instead, `replaced` names why).

## App

- **Profile** (owner): new section "Schwerpunkte" (`app/horses/[id]/training-profile/focuses.tsx`,
  row in the overview): the catalog as rows grouped "Skala der Ausbildung" / "Praxis", each with label,
  description and signs; tap selects (max 3), order by drag or arrows, optional "bis" `DateField`; a hint
  shows the phase ("Luna ist in der Gewöhnungsphase: Takt, Losgelassenheit, Anlehnung") and the warning
  for focuses above it; picking `turnier` prefills "bis" with the next show date. The wizard gets no new step; the training tab shows a quiet row "Schwerpunkte
  setzen" while none are set.
- **My focus** (owner and riders): training tab row "Mein Schwerpunkt" opens a sheet
  (`components/training-focus-sheet.tsx`) with the same rows, max 2, "Kein eigener Schwerpunkt" clears.
- **Today** (`app/training/index.tsx`): the raised block shows the focus (label, why) and the unit plan as
  three hairline rows "15 Lösen · Cavaletti auf dem Zirkel", "27 Arbeiten · …", "8 Entspannen · …"
  (minutes in Fraunces brass); tapping a row opens the exercise.
- **Week** (`app/training/week.tsx`, `components/training-week-day.tsx`): the day row shows the focus as a
  chip and the main exercise; the edit sheet gets a focus picker (catalog chips, "Eigener Text" keeps the
  free text) and the three unit parts (exercise per part, optional); "Übernehmen" sends `focus_key` and
  `unit`. The header shows the block ("Aufbauwoche 2 von 4" / "Entlastungswoche").
- **Finish** ("Wie war's?"): sends `focus_key` of the day's slot.
- **Library**: filter pills "Schwerpunkt" (catalog) and "Phase" (Lösen, Arbeiten, Entspannen); rows show
  library, level, minutes; the detail shows summary ("Wozu"), phase, minutes, intensity, equipment,
  prerequisites (links), sources (links open externally), and the progression.
- **Heute timeline**: training entries carry the focus label in the detail when the slot has one.
- Pure helpers with node:test tests: `lib/training-focus.ts` (labels, phase texts, selection limits,
  warnings, unit split display, block text), extended `lib/training-plan.ts` and `lib/tracking-exercises.ts`
  (new tag labels: Kondition, Ausdauer, Kraft, Gelände, Beweglichkeit; library labels Cavaletti, Gelände,
  Western).

## Tests and docs

- Backend: table tests for the rotation, the exercise pick, the block and return-to-work rules, variety,
  jumping limit, young and senior caps; API tests for focuses (validation, warnings, rider access), week
  and plan responses, sessions with `focus_key`; prompt test that no name, id or free text leaks; catalog
  test that every exercise uses known tags, phases, libraries and that `requires` points to the same
  library; coverage test for the required coverage above.
- Mobile: node tests for every pure helper.
- Docs: training.md gets the new routes and screens and links here; this file stays the planning spec.
- Seed: Luna gets focuses `losgelassenheit`, `geraderichten`; Mia (rider) `gelassenheit`; Fanta
  `kondition`; a `cycle_start`; sessions of the demo get `focus_key`s.
