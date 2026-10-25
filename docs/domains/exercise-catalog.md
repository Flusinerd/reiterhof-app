# Exercise catalog

The global exercise library behind the training planner: what a row is, which values are allowed, which coverage
the planner relies on, how to add an exercise and where the content comes from. The planner is specified in
[training-planning.md](training-planning.md) (focuses, unit plan, exercise pick, alternatives, level progression),
the API in [training.md](training.md). The rules below are enforced by `backend/internal/trainingapi/catalog_test.go`,
which reads the rows from a migrated database.

## Where the rows are

The catalog is the set of global rows (`stable_id IS NULL`) of table `exercises`. Stables may add their own rows (free
text, not part of the catalog rules).

| Migration | What it adds |
| --- | --- |
| `0250_exercise_catalog` | 41 rows: dressage 801–817, poles, cavaletti and jumping 820–827, groundwork 830–839, lunge 840–845 |
| `0280_leg_yield` | 846 Schenkelweichen (jumping warm-up) |
| `0330_exercise_metadata` | the columns phase, minutes, intensity, requires, equipment, suits, summary, sources; metadata of the 42 rows; the pole rows 820, 821, 823, 824 move to the library `cavaletti` |
| `0340_exercise_catalog_2` | 127 rows, IDs 850–989 (see the ranges below), progressions, one tag update (834) |

IDs are `00000000-0000-4000-8000-000000000NNN`; the number NNN names the row in tests, docs and tickets.

| Library (`discipline`) | IDs of 0340 | Rows | beginner / intermediate / advanced | without facility equipment | Used for |
| --- | --- | --- | --- | --- | --- |
| `dressage` | 850–889 | 55 | 41 / 7 / 7 | 55 | hall and arena days (home library) |
| `cavaletti` | 890–905 | 14 | 7 / 7 / 0 | 2 | poles and cavaletti on hall, arena, lunge and jumping days |
| `jumping` | 906–919 | 19 | 12 / 5 / 2 | 6 | jumping days (home library) |
| `groundwork` | 920–939 | 27 | 12 / 12 / 3 | 21 | groundwork days (home library) |
| `lunge` | 940–949 | 16 | 10 / 5 / 1 | 15 | lunge days (home library) |
| `hack` | 950–969 | 18 | 15 / 2 / 1 | 18 | hacks (home library) |
| `western` | 970–989 | 20 | 13 / 4 / 3 | 17 | hall and arena days of western horses |

The counts are those after 0340. IDs 990–999 are reserved. The column `discipline` keeps its name, but it means the
library (the API filter `library` is an alias of `discipline`, see the spec).

## Columns

| Column | Meaning |
| --- | --- |
| `title` | German, short, unique within the library. Titles and tags go into the prompt of the language model, so they carry no names of horses or people, no dates and not the word "Turnier" (`plan_test.go` checks this). |
| `summary` | One German sentence: what the exercise is for ("Wozu"). |
| `steps` | JSON list of 3 to 5 German sentences in the imperative/infinitive style of 0250 (wording rules below). |
| `level` | `beginner` (basic training, class E and A), `intermediate` (A* to L*), `advanced` (L** and up). Groundwork and lunge follow the FN badge stages (Reitabzeichen 10–8 beginner, 7–5 and Bodenarbeit Stufe 1 intermediate, Stufe 2 advanced; Longierabzeichen LA 5 beginner, LA 4 intermediate, LA 3 and 2 advanced); western follows the Westernreitlehre (Grundausbildung beginner, Gymnastizierung intermediate, Manöver advanced). |
| `goal_tags` | Tags from the fixed list below. |
| `phase` | `warmup` (Lösen), `main` (Arbeiten), `cooldown` (Entspannen) or `any` (usable in every part). |
| `minutes` | Typical duration, 1 to 120; it caps the part of the unit it is planned in. |
| `intensity` | `light`, `normal` or `demanding` (at most two demanding units a week). |
| `requires` | Prerequisite rows, all mastered before the exercise is planned. Same library, never of a higher level, no circles. |
| `equipment` | Keys from the fixed list below. |
| `suits` | `young` and/or `senior`: the row is a good variant for horses under 6 or 18 and older. Only tagged where a source says so. |
| `sources` | URLs of the pages the row is based on (at least one). The migration repeats them as SQL comments above the row. |
| `next_exercise_id` | The follow-up exercise of the progression. Same library, never easier than the row. |

**Tags** (`goal_tags`, the fixed list of the spec; unknown tags are refused by the test): `takt`, `rhythmus`,
`losgelassenheit`, `biegung`, `anlehnung`, `schwung`, `geraderichten`, `versammlung`, `durchlaessigkeit`,
`seitengaenge` (the points of the Skala der Ausbildung and lateral work), `balance`, `springgymnastik`,
`aufmerksamkeit`, `vertrauen`, `gelassenheit`, `kondition`, `ausdauer`, `kraft`, `gelaende`, `mobilitaet`, and
`turnier` for rehearsals of a show (the test of the class ridden through, entering and saluting, a course at show
height). The focuses match exercises by these tags.

**Equipment keys** (the fixed list `training.EquipmentKeys`):

- Facility equipment, planned against the stable's equipment of the location: `poles`, `cavaletti`, `jumps`,
  `obstacles` (fixed calmness or trail stations: bridge, narrow passage, flutter curtain, gate).
- Personal equipment, always available: `lunge`, `side_reins`, `double_lunge`, `kappzaum`, `tarp`, `umbrella`,
  `cones`, `flag`, `whip`, `neck_ring`, `trailer`.

A hack has no location with set-up, so no row of the library `hack` may name facility equipment (the test checks
this): trail obstacles in the terrain are simply there.

## Wording of the steps

The steps follow the ISES learning principles that the FN guidelines use: one thing at a time, a light signal first
and then more if needed, the release named ("Sobald das Pferd angehalten hat, gleich wieder die Hände nach vorne
nehmen"), praise instead of punishment. A step never says "muss" (the test checks the new rows), never asks to
punish a refusal or a startle, and never quotes a method that works with escalating pressure to the bone, with
jerks on the halter or with a "reprimand"; where a source mixes methods only its gentle part is used.

## Sources: nothing is made up

Every row names the pages it is based on. Every figure in a step (distances, counts, minutes, heights, ratios) stands
on one of the row's sources; a step without a figure in the sources leaves it out. The rows of 0340 were written
against the downloaded text of each page and checked by a script that looks up every number and number word of a
summary or step in the texts of the row's sources. Figures of different sources are not merged into one claim: where
two sources differ (for example minimum walk before work: ten minutes in one article, 15 in another), the row says
which one it follows or gives both.

Limits of the sources: the FN member magazine pages of 2025/2026 beyond `10-tipps-01..04-2025` are paywalled stubs
and were not used; the FN Aufgabenheft (test texts) and the EWU Merkblatt Westernreitabzeichen could not be read and
are not used; the Westernreitlehre is only available as table of contents, so the western rows rest on the EWU
Regelbuch 2024 (definitions of manoeuvres, trail dimensions), the Kreinberg pages and the aids page of
pferdchen.org; the Kondition chain rests on FN articles and on conditioning pages for eventing horses (Burger,
Equisense), which are labelled where a figure comes from them.

## Coverage the planner relies on

All of these are checked by `catalog_test.go` against the migrated database.

1. **Every focus has beginner exercises.** For every focus of the spec and every library an activity of its `fits`
   list draws from as home library (hall and arena: `dressage`; jumping: `jumping`; lunge: `lunge`; groundwork:
   `groundwork`; hack: `hack`) there are at least 3 beginner exercises with phase `main` that share a tag with the
   focus.
2. **Coverage without set-up.** At least 2 of those exercises need no facility equipment. A shared riding hall often
   allows no set-up, a lunge hall rarely has poles: the lunge library therefore has a solid pole-free set (tempo
   changes, halting and starting on voice, long-and-low, side reins, kappzaum, direction changes, circle size,
   double lunge).
3. **Alternatives.** Every exercise with facility equipment has at least one alternative: same library, phase equal
   or `any`, at least 2 shared tags, no facility equipment, level at or below. The cavaletti library has two rows
   without poles for this purpose (898 rhythm without poles, 899 lines with cones); the pole rows of `jumping`
   have 906 and 907 (seat without a jump, imaginary course); the stations of `groundwork` have 925 and 928.
4. **Every library has a warm-up and a cool-down:** at least 2 exercises with phase `warmup` or `any` and at least
   1 with `cooldown` or `any`.
5. **hack** has the Kondition chain 953 walk, 954 uphill walk, 955 steady trot stretch, 956 trot and canter
   stretches, 957 intervals, linked by `next`, from beginner to advanced.
6. **cavaletti** has walk (820, 890), trot straight (821, 891, 892) and bent (823, 893), canter (824, 895) and lunge
   (896) variants.
7. **groundwork** has the calmness stations tarp (839), umbrella (928), flutter curtain (929), bridge (930), narrow
   passage (931) and pole labyrinth (932), leading (830, 920, 924) and trailer loading (934).
8. **western** has the basics of the Kreinberg/EWU training: stop (973), back up (974), turns on the forehand (975)
   and the haunches (983), sidepass (984), circles with speed control (976), lope transitions (979) and trail
   obstacles (980–982).
9. **turnier:** at least 4 exercises in `dressage` (arriving and saluting, a warm-up as on the show ground, the tests
   of class E, A and L ridden through, a rehearsal) and at least 3 in `jumping` (course walk, an imaginary course, a
   related distance, courses at E, A, L and M height). All are `main`; none is `demanding` except a full course in
   jumping.
10. **Young and senior:** every library has at least one row that suits young horses and one that suits seniors.

Two links of the first catalog leave their library and are allowed by the test: 846 (jumping) leads to 821
(cavaletti) and 824 (cavaletti) leads to 825 (jumping); the pole rows moved to their own library in 0330 and their
follow-ups stayed.

## Adding an exercise

1. Pick the library and a free ID in its range; read the sources first and decide level, phase, minutes and
   intensity from them.
2. Write the row: title, summary, 3 to 5 steps, tags from the fixed list, equipment keys, `suits` only where a
   source says so, `requires` and `next` inside the library. Name every source in `sources`.
3. A row with facility equipment needs an alternative without (3 above): add one or extend the tags of an existing
   row, and say so in the migration comment.
4. Add the row to a **new** migration (the next free number; never edit a migration that has been released) as an
   `INSERT ... ON CONFLICT (id) DO UPDATE ... WHERE exercises.stable_id IS NULL` with the same column list as 0340,
   and set progressions with `UPDATE exercises e SET next_exercise_id = ...` after all rows exist.
5. Run `cd backend && go vet ./... && go test ./...`. `catalog_test.go` names the row and the rule when something is
   missing. Update the counts in `exercises_test.go` and `seed_test.go`.
6. Update this file (inventory table, coverage lists) and the ticket of the milestone in Linear.

Two things to know about the first planner. Until `suggestExercise` is replaced by the exercise pick of the spec, it
orders a library by level, then by the number of predecessors in the `next` chain, then by title, and
`today_test.go` and `sessions_test.go` assert which exercise comes first (`Rückwärtsrichten` in dressage, `Aufwärmen
im Schritt` and then `Handwechsel an der Longe` in lunge). The beginner rows of 0340 that have no predecessor
therefore have titles that sort after those; a new beginner row without predecessor should too, or sit behind
another row in a chain. And the prompt of the language model contains titles and tags of the global library (see
Columns).

## Sources used by 0340

Rows cite their sources by URL; the lists give the page, the number of rows of 0340 that cite it and the URL.

### FN member magazine "Pferd und Mensch" (magazin.pferdesport-deutschland.de), series "10 Tipps"

- 10 Tipps für eine wirkungsvolle Lösungsphase, 07/2019 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2019/>
- 10 Tipps für mehr Punkte im Viereck, 08/2019 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2019/>
- 10 Tipps für gelungene Übergänge, 11-12/2019 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2019/>
- 10 Tipps für eine feinere Anlehnung, 01/2020 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2020/>
- 10 Tipps für abwechslungsreiche Winterarbeit, 02/2020 (4 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2020/>
- 10 Tipps für schwungvolle Verstärkungen, 03/2020 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2020/>
- 10 Tipps für erfolgreiches Parcoursspringen, 04/2020 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2020/>
- 10 Tipps für gehorsames Rückwärtsrichten, 05/2020 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2020/>
- 10 Tipps für fleißiges Schreiten, 06/2020 (7 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2020/>
- 10 Tipps für gelungenes Geraderichten, 07/2020 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2020/>
- 10 Tipps für abwechslungsreiche Ausritte, 08/2020 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2020/>
- 10 Tipps für punktgenaues Halten, 09/2020 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2020/>
- 10 Tipps für einen kräftigen Pferderücken, 10/2020 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2020/>
- 10 Tipps für effektives Longieren, 11-12/2020 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2020/>
- 10 Tipps für einen großen Galopp, 01/2021 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2021/>
- 10 Tipps für die Bodenarbeit, 02/2021 (6 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2021/>
- 10 Tipps für einen sicheren leichten Sitz, 03/2021 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2021/>
- 10 Tipps für das Schulterherein, 06/2021 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2021/>
- 10 Tipps für den Vorbereitungsplatz, 07/2021 (4 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2021/>
- 10 Tipps für motivierte Pferde, 08/2021 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2021/>
- 10 Tipps für erste Sprünge im Gelände, 09/2021 (4 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2021/>
- 10 Tipps für den Ausritt in der Gruppe, 10/2021 (4 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2021/>
- 10 Tipps für das Freispringen, 02/2022 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2022/>
- 10 Tipps für kreisrunde Volten, 03/2022 (6 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2022/>
- 10 Tipps für mehr Kondition, 04/2022 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2022/>
- 10 Tipps für das Abteilungsreiten, 05/2022 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2022/>
- 10 Tipps für die Parcoursbesichtigung, 06/2022 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2022/>
- 10 Tipps für mehr Kadenz, 07/2022 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2022/>
- 10 Tipps für das Schenkelweichen, 08/2022 (6 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2022/>
- 10 Tipps für effektive Springgymnastik, 09/2022 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2022/>
- 10 Tipps für mehr Gleichgewicht, 01/2023 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2023/>
- 10 Tipps für kalte Tage, 02/2023 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2023/>
- 10 Tipps für die Arbeit mit Cavaletti, 03/2023 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2023/>
- 10 Tipps für die Doppellonge, 04/2023 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2023/>
- 10 Tipps für den einfachen Galoppwechsel, 05/2023 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-05-2023/>
- 10 Tipps für mehr Losgelassenheit, 06/2023 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2023/>
- 10 Tipps für einen gelungenen Außengalopp, 08/2023 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2023/>
- 10 Tipps für einen klaren Takt, 10/2023 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-10-2023/>
- 10 Tipps für eine gute Anlehnung, 11-12/2023 (3 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-11-12-2023/>
- 10 Tipps für bessere Bahnfiguren, 02/2024 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2024/>
- 10 Tipps für pferdegerechtes Reiten, 03/2024 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2024/>
- 10 Tipps für das Reiten über feste Hindernisse, 06/2024 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-06-2024/>
- 10 Tipps für mehr Trittsicherheit, 07/2024 (8 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-07-2024/>
- 10 Tipps für den fliegenden Galoppwechsel, 08/2024 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-08-2024/>
- 10 Tipps für gemeinsames Ausreiten, 09/2024 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-09-2024/>
- 10 Tipps für mehr Losgelassenheit, 01/2025 (1 row): <https://magazin.pferdesport-deutschland.de/project/10-tipps-01-2025/>
- 10 Tipps für eine feine Verbindung, 02/2025 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-02-2025/>
- 10 Tipps für ein aktives Hinterbein, 03/2025 (2 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-03-2025/>
- 10 Tipps für gemeinsame Ausritte, 04/2025 (5 rows): <https://magazin.pferdesport-deutschland.de/project/10-tipps-04-2025/>

### FN member magazine, "Lektion im Fokus" and articles

- Lektion im Fokus: Fliegende Galoppwechsel, 05/2023 (1 row): <https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-05-2023/>
- Lektion im Fokus: Galopp-Pirouette, 11-12/2023 (1 row): <https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-11-12-2023/>
- Lektion im Fokus: Überstreichen, 05/2021 (1 row): <https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-ueberstreichen-05-2021/>
- Abwechslungsreiches Training, 11-12/2024 (6 rows): <https://magazin.pferdesport-deutschland.de/project/abwechslungsreiches-training-11-12-2024/>
- Ausbildung: Das Cavaletti, der heimliche Superstar im Training, 08/2024 (6 rows): <https://magazin.pferdesport-deutschland.de/project/ausbildung-cavaletti-08-2024/>
- Cavaletti-Training: Gold wert, 01/2021 (5 rows): <https://magazin.pferdesport-deutschland.de/project/cavaletti-training-01-2021/>
- Der Wert des Aufgabenreitens, 06/2021 (5 rows): <https://magazin.pferdesport-deutschland.de/project/der-weg-des-aufgabenreitens-06-2021/>
- Ausbildungstipp: Ein Sprung, viele Möglichkeiten, 11-12/2021 (2 rows): <https://magazin.pferdesport-deutschland.de/project/ein-sprung-viele-moeglichkeiten-11-12-2021/>
- Einstieg in das Springen, 06/2021 (7 rows): <https://magazin.pferdesport-deutschland.de/project/einstieg-in-das-springen-06-2021/>
- Wissenschaft: Das Lernverhalten von Pferden, 07/2020 (6 rows): <https://magazin.pferdesport-deutschland.de/project/lernverhalten-von-pferden-07-2020/>
- Ausreiten: Sicher unterwegs im Gelände, 10/2019 (13 rows): <https://magazin.pferdesport-deutschland.de/project/sicher-unterwegs-im-gelaende-10-2019/>
- Young PM: So klappt es mit dem Abteilungsreiten, 06/2019 (1 row): <https://magazin.pferdesport-deutschland.de/project/so-klappt-es-mit-dem-abteilungsreiten-06-2019/>
- Springausbildung für Reiter und Pferd, Teil 3: Springgymnastik und Springreihen, 06/2017 (2 rows): <https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-3-springgymnastik-und-springreihen-06-2017/>
- Springausbildung für Reiter und Pferd, Teil 4: Distanzen und Kombinationen, 07/2017 (3 rows): <https://magazin.pferdesport-deutschland.de/project/springausbildung-fuer-reiter-und-pferd-teil-4-07-2017/>
- Titelthema: Blick in den Springparcours, 04/2025 (6 rows): <https://magazin.pferdesport-deutschland.de/project/springparcours-04-2025/>
- Sprünge passend anreiten: Der Weg ist das Ziel, 05/2022 (5 rows): <https://magazin.pferdesport-deutschland.de/project/spruenge-passend-anreiten-05-2022/>
- Turnierreiten: Der erste Eindruck zählt, 03/2025 (11 rows): <https://magazin.pferdesport-deutschland.de/project/turnierreiten-03-2025/>
- Young PM: Tipps zum Führen mit Halfter, 03/2019 (5 rows): <https://magazin.pferdesport-deutschland.de/project/young-pm-tipps-zum-fuehren-03-2019/>

### Merkblätter, rule books, training articles and science

- Burger, Universität Bern / Agroscope: Vielseitig trainiert (Brennpunkt Pferd, HAFL, 2019) (2 rows): <https://alumni.bfh.ch/dam/jcr:7dc00f6f-7387-4071-9e88-38b1406834a0/burger-vielseitig-trainiert-brennpunkt-pferd-2019.pdf>
- Cavallo: Geführte GHP für Einsteiger (stations of the Gelassenheitsprüfung; only the gentle methods are used) (6 rows): <https://www.cavallo.de/pferdeverhalten/gefuehrte-ghp-fuer-einsteiger/>
- Cavallo: Gymnastik an der Longe (1 row): <https://www.cavallo.de/reittraining/gymnastik-an-der-longe/>
- Cavallo: Vier Übungen für besseres Seitwärts am Boden (3 rows): <https://www.cavallo.de/reittraining/vier-uebungen-fuer-besseres-seitwaerts-am-boden/>
- EWU Regelbuch 2024 (definitions of the manoeuvres, trail dimensions, rules for the trail class) (19 rows): <https://berlinbrandenburg.ewu-bund.com/wp-content/uploads/2024/02/EWU-Regelbuch_2024.pdf>
- Equisense: So planst du das Training deines Pferdes über eine ganze Saison (2 rows): <https://blog.equisense.com/de/training-pferd-plan-ganze-saison/>
- FN Merkblatt Abzeichen Longieren LA 5 bis LA 1V, Stand Oktober 2025 (3 rows): <https://www.pferdesportwestfalen.de/fileadmin/Media/Dateisammlung/Dokumente_Abzeichen/Abzeichen_Merkblaetter/merkblatt_longierabzeichen_2026.pdf>
- FN Merkblatt Bodenarbeit, Stand Oktober 2019 (Reitabzeichen 10 to 5, Bodenarbeit Stufe 1 and 2) (16 rows): <https://www.lpbb.de/files/lpbb/images/Ausbildung/Abzeichen/Merkblatt_Bodenarbeit_Stand_Oktober_2019_APO2020.pdf>
- Hooforia: Longieren: sinnvoll eingesetzt hilft es Pferd und Reiter (9 rows): <https://www.hooforia.com/reitausbildung/bodenarbeit/longieren-sinnvoll-eingesetzt-hilft-es-pferd-und-reiter/>
- Horses and People, Dr Kate Fenner: Easy Trailer Loading in 10 Easy Steps (1 row): <https://horsesandpeople.com.au/10-easy-steps-to-easy-trailer-loading-rq/>
- Kreinberg College: Western & Dressage, 53 Übungen für feineres Reiten (titles of the exercises) (2 rows): <https://www.kreinberg.college/lp/westerndressage>
- Land Oberösterreich: Wenn Pferde in die Jahre kommen (Volksblattartikel, 2022) (2 rows): <https://www.land-oberoesterreich.gv.at/Mediendateien/Formulare/Dokumente%20SGD%20Abt_ESV/Volkblattartikel_Wenn%20Pferde%20in%20die%20Jahre%20kommen_19.11.2022.pdf>
- Pferdchen.org: Hilfengebung beim Westernreiten, Basics (9 rows): <https://www.pferdchen.org/Reiten/Reitstile/Westernreiten/Hilfengebung.html>
- The Gentle Touch (Kreinberg): Rückwärtsrichten mit Leichtigkeit (1 row): <https://www.thegentletouch.de/post/r%C3%BCckw%C3%A4rtsrichten-mit-leichtigkeit>
- The Gentle Touch (Kreinberg): Versammlung in Ruhe vorbereiten, Quarter Horse Journal (5 rows): <https://www.thegentletouch.de/post/kreinberg-artikel-im-quarter-horse-journal>
- Tierspital Zürich: Arthrose beim Pferd, Informationsblatt für Studienteilnehmer (2023) (4 rows): <https://www.tierspital.uzh.ch/wp-content/uploads/2023/03/Infoblatt_Arthrosestudie_230123.pdf>
- Wikipedia: Longieren (2 rows): <https://de.wikipedia.org/wiki/Longieren>
- wehorse: Westernreiten, alles zur Disziplin (8 rows): <https://www.wehorse.com/de/blog/westernreiten>

The sources of the 42 older rows are in the `sources` column of those rows.
