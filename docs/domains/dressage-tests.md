# Practice tests and freestyles (Aufgaben und Küren)

Specification for milestone 0.4.0. Owners and riders generate their own dressage tests to ride at home
(Übungsaufgabe) and plan a freestyle (Kür) for a show, both drawn in the arena with letters and timed.
English for code and this document, German for every UI text.

## Copyright

FN test texts (Aufgabenheft), score sheets and Leitfäden are © FNverlag ("Nachdruck oder sonstige
Vervielfältigung zu kommerziellen Zwecken verboten"). The app ships none of them: no test text, no
score sheet layout, no FN test name as the title of a generated test. It ships **facts** that are not
protected as such: arena geometry, the names of movements, the rules of the Kür leaflet (salute,
walk lines, time windows, deductions) written in our own words, and per class the list of required
movements with the source named. Generated tests are titled "Übungsaufgabe E · 20×40" or by the user.
The app links to the official sources: the FN Regelwerke app and fnverlag.de for the Aufgabenheft,
pferdesport-deutschland.de for the Kür leaflet. A user may keep the text of an official test as a
private note on a show class (`shows[].tests[].note`, own words or a photo, visible only to the horse's
owner and riders); the app never distributes it.

Sources (every figure below comes from one of them; the implementing agent re-checks each and adds
more where the spec says "research"):

- FN/DRV „Merkblatt Dressur-Kür“, gültig ab 01/2024, and its „Anlage“ (pferdesport-deutschland.de shop
  downloads, sample ids 162 and 161).
- FN score sheets and Leitfäden are not used as a source for shipped data (see "Requirements are
  entered by the user").
- Arena geometry: de.wikipedia.org „Dressurviereck“, FEI dressage rules (arena annex).

## Arena

`dressage.Arena` (pure Go, `backend/internal/dressage`), two sizes, coordinates in metres with A at
(10, 0) and C at (10, L) where L is 40 or 60, x across the arena (0 = left long side seen from A).

| Size | Long side letters (from A, left side / right side) | Centre line |
| --- | --- | --- |
| 20×40 | K, E, H / F, B, M; K, F, H, M 6 m from the corners, E and B at 20 m | D, X, G (D 6 m from A, X 20 m, G 34 m) |
| 20×60 | K, V, E, S, H / F, P, B, R, M; 6 m from the corners, then every 12 m | D, L, X, I, G (in line with K/F, V/P, E/B, S/R, H/M) |

Track = 0.5 m inside the fence is not modelled; paths run on the fence line, a quarter line is 5 m, the
centre line x = 10. Corners are quarter circles of radius 3 m (class E/A), 2 m above ("ganze Bahn"
corners ridden according to the class). Research: confirm the corner radii per class from the FN
Richtlinien Band 1 or leave one radius 3 m for all and say so.

## Figures

A **figure** is a path with a start and end point, a length and the rein it ends on. The figure library
(`dressage.Figures`) is static data; each figure has a key, a German label template, the arena sizes it
fits, its points, and a path builder returning the polyline (for the length and the drawing):

- track segments between two letters (`track`), corners included;
- change of rein: long diagonal (K–X–M, F–X–H), in 20×60 also the short diagonals between a corner
  letter and the middle letter of the other long side (research which ones FN tests use), through the circle („durch den Zirkel wechseln“), out of the corner;
- centre line A–C (with halt at X or G), quarter lines;
- circles (`zirkel`, 20 m at A, C, and in 20×60 at E/B), voltes 10 m and 8 m at a letter on the track or
  at X, double voltes (Doppelvolte), half voltes back to the track;
- serpentines: 3 and 4 loops through the whole arena, single and double loops on the long side
  (einfache/doppelte Schlangenlinie, 5 m / 2.5 m deep);
- „Viereck verkleinern/vergrößern“, turn on the haunches / short turn (Kurzkehrt) at a letter,
  half pirouette in walk;
- lateral work along a line: leg yield (Schenkelweichen) from the quarter line or centre line to the
  track, shoulder-in and travers on the long side (≥ 12 m), half pass from the centre line to the track
  (≥ 5 m sideways);
- straight-line tempo changes: lengthening on the long side or the diagonal (never on a clearly curved
  line), transitions at a letter;
- halt and salute on the centre line facing C, rein back (a number of steps).

## Movements

A **movement** (`dressage.Movements`) is what the rider shows: key, German label, gait (`walk`,
`trot`, `canter`, `halt`), tempo (`working`, `medium`, `extended`, `collected`, `free`), the lowest class
that asks for it, the figures it can be ridden on (and per figure the constraints, e.g. walk only on
straight or curved lines, lengthening only on straight lines, canter voltes in true canter), its rein
(`left`, `right`, `either`, `both` = must be shown on both reins) and the linked catalog exercises
(`exercise_ids`, so mastery comes from the training sessions: a movement is mastered when a linked
exercise has a session rated `focus_rating` 3). The lowest class per movement comes from the FN
Richtlinien für Reiten und Fahren (Band 1/2) as described in FN magazine articles and Wikipedia, not from
test texts or score sheets; every entry names its source, and a movement without a source has no class
(it is offered from L** up and marked "Klasse nicht belegt").

## Speeds and duration

Duration = path length / speed of the gait and tempo, plus fixed times for halts (salute 6 s, halt 4 s),
transitions (1 s) and rein back (1 s per step). The speeds are constants in metres per second with a
source each (research: published GPS/biomechanics measurements of dressage horses; if none are found,
derive from strides per minute and stride length in a named source). The generator reports the total
and per segment; the app shows mm:ss.

## Practice test (Übungsaufgabe)

`POST /api/v1/horses/{id}/dressage-tests` with `{kind: "practice", class: "E"|"A"|"L"|"L**",
arena: "20x40"|"20x60", focus_key?: string, length: "short"|"normal"}`.

Generator (pure, deterministic for a given seed, `seed` returned and accepted for "Neu würfeln"):

1. Movements of the class whose lowest class ≤ class; when the horse has sessions, only mastered
   movements plus at most 2 that are the `next` of the level progression (0.3.0 `Progress`); without
   sessions all movements of the class.
2. Structure (general dressage convention, own wording, never modelled on a specific FN test): enter at A, halt and salute at X or G,
   work on both reins in balance (±20 % path length per rein), each gait on both reins, a free walk on
   a long rein in classes E/A, a halt at the end facing C, leave at A on a long rein. Order of gaits:
   trot on both reins, walk (with the free walk on a diagonal), canter on the first rein, trot with the change
   of rein, canter on the second rein, trot; each canter is one block that starts after a trot line.
3. A focus prefers movements whose linked exercises carry the focus's tags.
4. Chain figures so each starts where the last ended (connectors along the track count as "ganze
   Bahn"). Search: depth-first with backtracking over the figure options, scoring variety (no figure twice
   in a row, not only standard lines), balance of reins, and the target duration (`PracticeWindow`): short
   3:30–5:00, normal 5:00–6:00 (E/A) or 5:30–7:00 (L, L**). These are the app's own targets, not FN figures.
5. Output: numbered lines like an FN test but in our own words: `{n, letters: "A–X", text: "Einreiten
   im Arbeitstrab, bei X halten, grüßen", gait, tempo, rein, seconds, figure, path: [[x,y],…]}`.

## Freestyle (Kür)

`POST /api/v1/horses/{id}/dressage-tests` with `{kind: "freestyle", class: "A"|"L*"|"L**"|…, arena,
requirements?: [...], seed?}`.

- **Requirements are entered by the user.** The required movements and the time window of a freestyle
  class are part of FN's score sheets (Leitfäden/Notenbögen, © FNverlag) and are **not shipped**, not
  even as a list. "Anforderungen" is an editor: the user picks the required movements from the app's own
  movement list (with count and rein) and enters the time window from their own copy of the
  Aufgabenheft; the editor says so ("Aus deinem Aufgabenheft übernehmen"). Saved requirement sets are
  per stable and can be reused for the next freestyle of the same class (`freestyle_requirements`
  table: stable_id, class, label, movements jsonb, min_seconds, max_seconds, created_by).
- **Rules** from the Kür leaflet and its annex, enforced by `dressage.CheckFreestyle` and explained in
  the result: salute at the start and the end on the centre line facing C; required walk only on
  straight and/or curved lines (not as lateral work or „Viereck verkleinern“), at least the required
  distance; lengthenings in trot not on clearly curved lines or through two corners; canter voltes in
  true canter; Kurzkehrt not more than 180°; no movement of a higher class (annex per class, e.g. no
  flying changes and no lateral work in canter in L*/L**, no lateral work in L*); allowed extras per
  class from the annex (A: serpentines of any kind in trot, 10 m double voltes in trot, „Viereck
  verkleinern/vergrößern“ in walk/trot, canter change via ≥ 6 strides medium walk; L*: all serpentines in
  trot and canter, double voltes 8–10 m in trot and canter; L**: zig-zag half pass in trot, travers and
  renvers in addition); time within the window (more than 10 s off costs points).
- **Choreography** (B-note criteria of the leaflet, used as scoring of the generator, never as a
  promise of a score): use the whole arena, balance left and right, present highlights towards C, not
  only standard lines, no lesson always on the same line, alternate demanding passages with calm ones
  and build to a finale, each gait balanced, few gait changes. The result lists which criteria the
  plan meets in plain words ("Beide Hände ausgewogen: links 48 %, rechts 52 %").
- **Music** (no files, no licences): per segment the gait so the user can cut music; the result groups
  consecutive segments into "Musikteile" (Schritt, Trab, Galopp) with their durations. Research tempo
  ranges (beats per minute) per gait from a named source; leave them out if none is found.

## Data and API

Migration `0370_dressage_tests.up.sql`: `dressage_tests (id, stable_id, horse_id, created_by, kind
text CHECK (kind IN ('practice','freestyle')), class text, arena text CHECK (arena IN
('20x40','20x60')), title text, focus_key text, seed bigint, requirements jsonb, plan jsonb, created_at,
updated_at)`; index on (horse_id, created_at).

| Route | |
| --- | --- |
| `GET /api/v1/dressage/catalog` | arena sizes with letters, movements (key, label, gait, tempo, lowest class), freestyle classes (key, label) and the stable's saved requirement sets |
| `POST /api/v1/horses/{id}/dressage-tests` | generate; `{save: false}` returns the plan without storing (preview), `save: true` stores |
| `GET /api/v1/horses/{id}/dressage-tests` | saved tests, newest first |
| `GET/PUT/DELETE /api/v1/dressage-tests/{id}` | read; PUT `{title, plan}` stores the user's edits (reordered or removed lines) after `CheckFreestyle`/`CheckPractice` (400 with the German reasons when a rule breaks); delete |

Access as the training profile: owner and admin full, riders with `log_sessions` read and generate.
The plan JSON: `{kind, class, arena, seed, lines: [...], seconds, reins: {left, right}, gaits: {walk,
trot, canter}, music: [{gait, seconds}], checks: [{key, ok, text}], requirements: [{key, label, count,
met}]}`.

The week planner (0.3.0) gets one addition: when a show day has a class with a saved freestyle or
practice test of that horse, the `turnier` focus picks "Aufgabe reiten" with that test linked
(`unit.main.test_id`).

## Backend as built (decisions marked **Decision**)

Package `backend/internal/dressage` (pure), routes in `backend/internal/trainingapi/dressage.go`, migration
`0370_dressage_tests.up.sql`.

- Routes added to the table above: `GET/POST /api/v1/dressage/requirements` and
  `GET/PUT/DELETE /api/v1/dressage/requirements/{id}` (stable members read; owners, admins and riders with
  `log_sessions` or the legacy `ride` write). `GET /dressage/catalog` returns `arenas` (letters with positions),
  `classes` (`key`, `label`, `practice`), `movements`, `speeds` with `speed_source`, `links` and `requirement_sets`.
  The POST body: `{kind, class, arena, length?, focus_key?, seed?, save?, title?, requirements_id? |
  requirements?: {label, movements: [{key, count, rein, min_meters?}], min_seconds, max_seconds}}`; `save`
  defaults to false (preview, 200), true stores (201). A freestyle without a set has no required movements and
  a target of 4 minutes. A rule that breaks on PUT answers `400 {"error": {"code": "rules_broken", ...},
  "reasons": [German texts]}`; a client may send back the whole plan, only `option`, `gait` and `tempo` of each
  line count, the rest is rebuilt from the figure library.
- **Decision**: `freestyle_requirements` has `updated_at` (PUT). A saved test keeps a copy of its set in
  `dressage_tests.requirements`, so deleting or editing the set changes no stored test.
- **Decision**: access to the tests of a horse (read, generate, edit) needs owner, admin, or a rider with
  `log_sessions` (or the legacy `ride`); delete needs owner, admin or the author.
- Plan JSON as built: `{kind, class, arena, seed, lines, seconds, reins: {left, right}, gaits: {walk, trot,
  canter}, music: [{gait, seconds, from, to, cadence}], checks: [{key, ok, rule, text}], requirements: [{key,
  label, rein, count, shown, met}]}`. `reins` and `gaits` are seconds. A line is `{n, option, letters, text,
  gait, tempo, rein, seconds, meters, figure, movements, straight, lateral, counter?, degrees?, start, end, path}`.
  `checks` with `rule: true` are rules (a failing one is an error, PUT answers 400), the others are advice
  (choreography, duration of a practice test).
- Arena: coordinates and letters as in "Arena"; **Decision**: one corner radius of 3 m for all classes (no FN
  source per class). The track is the rounded rectangle, a letter on it is 6 m from the corner.
- Figures (`dressage.Library`, per arena size): track pieces between letters (up to 4 letters), lateral work on
  a long side (shoulder-in, travers, renvers, at least 12 m), long diagonals K–X–M and F–X–H and across the
  arena E–X–B (20×60 also V–P and S–R), centre line A–C and C–A, serpentines through the whole arena (3 and 4
  bows), single and double serpentines on the long side, circles of 20 m at A, C, E and B, voltes of 10 and 8 m
  at every letter of the track, through the circle (20×40 only), Viereck verkleinern, leg yield from the
  quarter line, half pass from the centre line to the track (at least 5 m sideways), Kurzkehrt (180°), and in
  canter a diagonal with a simple change of lead and a diagonal with counter canter and a simple change.
  Not modelled: short diagonals, double voltes, half pirouette, rein back as a separate figure (it is part of
  the centre line variant with a halt at X). Lengthenings are shown on straight lines only.
  **Decision**: "durch die halbe Bahn wechseln" is the straight line across the arena at E–B.
- Speeds: Back & Clayton, Equine Locomotion (2nd ed. 2013), as quoted by Equisense: walk 84–108, trot 192–296,
  canter 196–358 m/min for dressage horses; cadence 52–56, 77–83, 99–105 strides/min. **Decision**: the source
  gives the range per gait only, a tempo is placed in it by our own fraction (collected 0, working 0.25, free
  walk 0.4, medium 0.6, extended 1). Halt with salute 6 s (includes the transitions), transition 1 s, rein
  back 1 s per step, simple change of lead 6 s. Music: only the cadence of each gait, no beats per minute (the
  source has none).
- Lowest class per movement (`lowest_class`, `class_note`, `source`): sourced from FN magazine articles,
  Wikipedia and the general rules of the Kür leaflet, never from a test text or score sheet: schenkelweichen E
  (FN Pferd und Mensch 08/2022), mitteltrab A (07/2022), Rückwärtsrichten A (11/12 2021), Volte 10 m in trot A
  (Wikipedia „Volte“), Volte 8 m in trot and canter L (FN 07/2023, Wikipedia), Außengalopp L (FN 08/2023,
  Wikipedia), einfacher Galoppwechsel L and Kurzkehrt L (Merkblatt Kür 2024), serpentines in trot and Viereck
  verkleinern A, serpentines in canter L (Anlage 2024), shoulder-in, travers, renvers and half pass in trot L**
  (Merkblatt and Anlage 2024), flying change M* (Wikipedia „Galoppwechsel“). **Decision**: the gaits in the
  working tempo, halt and salute, the circle and the changes of rein (diagonal, across the arena, through the
  circle, centre line) have no class of their own in any source; they are set to E with the PM-Forum article on
  Hufschlagfiguren (02/2023) as the source and marked `basis`. Every other movement without a source (collected,
  medium and extended tempi except Mitteltrab, canter voltes of 10 m, serpentines in walk, half walk pirouette)
  has no class and is offered from L** with "Klasse nicht belegt". **Decision**: a movement that the user
  requires in a freestyle is allowed whatever its class (the class rule is for the extras).
- Practice generator (`practice.go`): ten blocks in this order, each filled by a depth-first search with
  backtracking over the figures of the library: trot on the first rein, trot with the change of rein and
  work on the other rein (figures only after the change), medium walk, free walk on a diagonal (from L also
  across the arena) and medium walk again, one trot track line to the letter of the canter, canter on the first
  rein as one block (entry at a letter, one or two figures, back to trot), trot with the change of rein, canter
  on the second rein as one block, a last trot line to A, then the salute and the exit. No gait changes inside a
  canter block. From class L the second canter block shows a simple change of lead (diagonal or counter canter
  to the next letter) in about half of the seeds; it ends on the first rein and the last trot follows on it.
  The search builds the plan line by line; a plan is accepted when `CheckPractice` has no failing rule, the
  duration is inside `PracticeWindow`, both reins have 40–60 % of the time and the line limit holds (120
  attempts with other random choices per seed, the same seed always gives the same plan).
  **Decision** (own quality rules, not from a source): short tests have at most 18 lines, normal tests at most
  26 (entry and exit included); two track lines of one gait never follow each other (one track line covers up to
  four letters) and a track line is at least 6 s of riding; a figure is not repeated on the same letters and a
  volte does not follow a circle at the same letter; each movement shows at most twice per gait (a circle in
  trot, in walk and in canter counts separately; plain gaits and the salute are not counted), at most two
  lateral lines per rein, at most one serpentine per test; the rein back is only a walk figure, centre lines are
  only changes of rein and rein back is no change of rein. A canter block has one or two figures (a change of
  lead counts as one). Short tests show a subset: per gait the core movements of the class
  (`CoreMovements`: for example A in trot circle, volte 10 m, medium trot and leg yield) plus at most two other
  movements, normal tests plus at most four; the extras are drawn at random, the focus movements always come
  first and mastered movements are twice as likely. The core lists are the app's own choice from the movements
  of the class (E: circle and leg yield in trot, circle in walk and canter; L adds volte 8 m, serpentine, Viereck
  and the change of lead; L** the lateral work). **Decision**: the generator no longer uses the freestyle
  engine; PracticeWindow of the normal test in L and L** starts at 5:30 (before 6:00) because 26 lines in the
  20×40 arena last about 14 s each. With sessions only the movements with a mastered exercise, plus at most two
  whose exercise is among the next lessons of the level progression, plus the movements without a linked
  exercise (gaits, halt, the common figures); a gait without any such figure still rides circles. Kurzkehrt and
  Kurzkehrt-based changes are not placed by the practice generator; they are in the library, the catalog and the
  freestyle.
- Freestyle generator: unchanged search; afterwards track lines that follow each other in one gait and tempo are
  merged into one line when the library has a track line of that length (up to four letters) and the merged plan
  breaks no rule.
- Rules of `CheckFreestyle` (own words, from the Kür leaflet): `chain`, `opening` (entry and salute on the centre
  line), `closing`, `class`, `gaits` (all three shown), `walk` (required walk on straight and curved lines only,
  as long as the user's minimum), `lengthening` (trot lengthenings on straight lines), `canter_voltes` (true
  canter), `kurzkehrt` (at most 180°), `time` (inside the window ±10 s), `requirements` (counts and reins).
  Advice: `arena_use`, `balance`, `highlights_c`, `lines`, `same_line`, `alternation`, `finale`, `gait_changes`,
  `gait_balance` (the thresholds are the app's own). `CheckPractice`: `chain`, `opening`, `closing`, `class`,
  `gaits_both_reins`, `balance` (each rein at least 40 %), advice `duration`.
- **Decision**: the duration of a freestyle with many required movements can exceed a tight window; the plan
  is returned with the failing `time` check and the user edits it or loosens the window.
- Wire format as the app uses it (checked end to end): `GET /horses/{id}/dressage-tests` answers `{tests: [{id, kind,
  class, arena, title, focus_key, seconds, lines, created_by, created_at}]}`; a preview (`save: false`) and a stored
  test answer with the same envelope `{id (null for a preview), horse_id, kind, class, arena, title, focus_key, seed,
  requirements, plan, saved, ...}` (the plan is under `plan`, not the body itself); `GET /dressage/requirements`
  answers `{requirements: [...]}`; an inline `requirements` object needs no `label`. A `music` part counts lines:
  `from` and `to` are line numbers, not seconds (the app adds up the seconds itself). Every line carries `option`,
  and a PUT must send it back (without it the answer is `400 "Zeile 1: unbekannte Figur"`). Errors answer
  `{error: {code, message}, reasons?: [...]}` with `message` = the reasons joined by a space; 403 for a horse the caller
  may not use (list, generate, read, edit) and for deleting the test of someone else (code `forbidden`, English text,
  the app shows its own German one).
- **Decision**: a freestyle needs at least about 5 minutes for five required movements (all gaits on both reins,
  salute, free walk); a shorter window always fails the `time` check. The app's editor starts with 5:00 to 6:00.
- Open: the `turnier` link (`unit.main.test_id`) is not built. Shows have no class or test list yet
  (`shows[].tests` of "Copyright" does not exist), the unit plan stores no test id and the planner is the language
  model prompt; it needs those three together.

## App

- **Horse → Training → "Aufgaben und Küren"** (`app/training/tests/index.tsx`): saved tests as rows
  (title, class, arena, duration), primary action "Neue Aufgabe".
- **New** (`app/training/tests/new.tsx`): Segmented "Übungsaufgabe / Kür", class picker (default from
  the profile level; classes above the goal level are hidden), arena 20×40 / 20×60, for practice
  "kurz / normal" and an optional focus, for Kür the requirements editor (nothing is prefilled, see "Requirements are
  entered by the user": movements from the catalog with count and rein, time window in steps of 10 s, saved sets of the
  class as pills, "Set für den Stall speichern"). "Erstellen" shows the preview, "Speichern" there stores the test with
  the seed of the preview and returns to the list.
- **Plan view** (`app/training/tests/[id].tsx`, also used for the preview): the arena drawn as SVG
  (react-native-svg)
  with letters, the path of the selected line highlighted, the full path faint; below the numbered
  lines (letters in Fraunces, text, mm:ss); for Kür the requirement checklist, "Regeln" (checks with `rule: true`, the
  failing ones first) and "Hinweise" (advice about the choreography) and the music parts; while lines are edited the
  server's checks are hidden (they would be stale) and the reasons of a refused save stand above "Speichern"; actions: "Speichern" (preview), "Neu würfeln", "Zeilen bearbeiten" (reorder/remove with
  re-check), "Teilen" (plain text via the share sheet), "Abspielen" (animates a dot along the path at
  real speed or 2× for learning the test).
- The official tests: a quiet row "Offizielle Aufgaben" linking to the FN Regelwerke app/fnverlag.de
  (Linking.openURL), never their text.
- Design system "Abendlicht" and the Feedback section (pending states, success in place, double-tap
  guard, effects map: new effect `dressageTests {horseId}`).

## Tests

- Go, pure: arena coordinates and distances (letters, centre line points, lengths of figures with a
  tolerance), figure chaining (every line starts where the previous ended), practice generator for each
  class and size (structure rules, line limits, phases in order, canter blocks, limits per movement, core plus
  extras, balance, duration window, determinism and variety by seed, only mastered plus next), freestyle generator and `CheckFreestyle` for every rule above with a failing and a passing
  case, catalog test (every movement has a source, every class requirement maps to movements, no FN
  test name in titles).
- API tests: generate preview and save, access (owner, rider, stranger), PUT with a rule break → 400.
- Mobile node tests: arena drawing helpers (scale, letter positions), line formatting, duration
  formatting, requirement checklist, music parts.
- Docs: this file stays the spec; `docs/domains/training.md` links it; CHANGELOG "Unreleased".
