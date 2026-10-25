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
- „Leitfaden Dressurprüfung der Klasse L* – Kür“ (FNverlag score sheet, Stand 1.1.2018, as published by
  clubs): required movements and time windows of L*.
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
exercise has a session rated `focus_rating` 3). Research: the movements per class E, A, L*, L** from
FN sources (Richtlinien, published Leitfäden, FN magazine articles, reiter1x1/ehorses/zuegel-und-buegel
as secondary sources only where they match a primary one); every class entry names its source.

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
2. Required structure as in every FN test (facts, no text): enter at A, halt and salute at X or G,
   work on both reins in balance (±20 % path length per rein), each gait on both reins, a free walk on
   a long rein in classes E/A, a halt at the end facing C, leave at A on a long rein. Order of gaits:
   trot, walk, canter, trot, as common in tests; canter work on each rein starts after a trot phase.
3. A focus prefers movements whose linked exercises carry the focus's tags.
4. Chain figures so each starts where the last ended (connectors along the track count as "ganze
   Bahn"). Search: greedy with backtracking over the figure options, scoring variety (no figure twice
   in a row, not only standard lines), balance of reins, and the target duration: short 3–4 min,
   normal 5–6 min (classes E/A) or 6–7 min (L, L**) — research and source the typical durations.
5. Output: numbered lines like an FN test but in our own words: `{n, letters: "A–X", text: "Einreiten
   im Arbeitstrab, bei X halten, grüßen", gait, tempo, rein, seconds, figure, path: [[x,y],…]}`.

## Freestyle (Kür)

`POST /api/v1/horses/{id}/dressage-tests` with `{kind: "freestyle", class: "A"|"L*"|"L**"|…, arena,
requirements?: [...], seed?}`.

- **Requirements per class** (`dressage.FreestyleClasses`): required movements with counts and reins,
  time window per arena size, allowed extras and forbidden "higher class" movements. Shipped only for
  classes with a source; L* from the Leitfaden: Mittelschritt (≥ 40 m continuous), Kurzkehrt (right
  and/or left), versammelter Trab, Volte im Trab (right and/or left), Mitteltrab, versammelter Galopp,
  Volte im Galopp right, Volte im Galopp left, Mittelgalopp, Außengalopp right, Außengalopp left,
  einfacher Galoppwechsel right–left and left–right; time 3½–4 min in 20×40, 4½–5 min in 20×60.
  Research the other classes (A, L**, M*) from published Leitfäden; a class without a source is offered
  with "Anforderungen selbst eintragen": the user picks the required movements from the movement list
  and enters the time window.
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
| `GET /api/v1/dressage/catalog` | arena sizes with letters, movements (key, label, gait, tempo, lowest class), freestyle classes with requirements, time windows and sources |
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

## App

- **Horse → Training → "Aufgaben und Küren"** (`app/training/tests/index.tsx`): saved tests as rows
  (title, class, arena, duration), primary action "Neue Aufgabe".
- **New** (`app/training/tests/new.tsx`): Segmented "Übungsaufgabe / Kür", class picker (default from
  the profile level; classes above the goal level are hidden), arena 20×40 / 20×60, for practice
  "kurz / normal" and an optional focus, for Kür the requirements (prefilled from the class, editable
  when the class has no source). "Erstellen" shows the preview.
- **Plan view** (`app/training/tests/[id].tsx`, also used for the preview): the arena drawn as SVG
  (react-native-svg, already a dependency? check; if not, a pinned addition via `npx expo install`)
  with letters, the path of the selected line highlighted, the full path faint; below the numbered
  lines (letters in Fraunces, text, mm:ss); for Kür the requirement checklist, rule checks and the
  music parts; actions: "Speichern" (preview), "Neu würfeln", "Zeilen bearbeiten" (reorder/remove with
  re-check), "Teilen" (plain text via the share sheet), "Abspielen" (animates a dot along the path at
  real speed or 2× for learning the test).
- The official tests: a quiet row "Offizielle Aufgaben" linking to the FN Regelwerke app/fnverlag.de
  (Linking.openURL), never their text.
- Design system "Abendlicht" and the Feedback section (pending states, success in place, double-tap
  guard, effects map: new effect `dressageTests {horseId}`).

## Tests

- Go, pure: arena coordinates and distances (letters, centre line points, lengths of figures with a
  tolerance), figure chaining (every line starts where the previous ended), practice generator for each
  class and size (structure rules, balance, duration window, determinism by seed, only mastered plus
  next), freestyle generator and `CheckFreestyle` for every rule above with a failing and a passing
  case, catalog test (every movement has a source, every class requirement maps to movements, no FN
  test name in titles).
- API tests: generate preview and save, access (owner, rider, stranger), PUT with a rule break → 400.
- Mobile node tests: arena drawing helpers (scale, letter positions), line formatting, duration
  formatting, requirement checklist, music parts.
- Docs: this file stays the spec; `docs/domains/training.md` links it; CHANGELOG "Unreleased".
