import assert from "node:assert/strict";
import { test } from "node:test";

import {
  BLOCK_WEEKS,
  FOCUS_LIMIT_OWNER,
  FOCUS_LIMIT_RIDER,
  LEVEL_UP_KEY,
  blockText,
  catalogPhaseOf,
  clearUnitExercises,
  dayFocusLabel,
  dayFocusLines,
  defaultUntil,
  focusChoices,
  focusDraftsFrom,
  focusInputs,
  focusLabel,
  focusLabels,
  focusNote,
  focusSections,
  focusWarnings,
  focusesSummary,
  groupLabel,
  isAbovePhase,
  limitNotice,
  mainExercise,
  moveFocus,
  myFocusSummary,
  nextShowDate,
  phaseFitsPart,
  phaseHint,
  phaseName,
  resplitUnit,
  restartText,
  selectionCaption,
  sessionFocusKey,
  setFocusUntil,
  setUnitExercise,
  toggleFocus,
  toggleKey,
  trainingEyebrow,
  unitBody,
  unitMinutes,
  unitRows,
  unitSplit,
  validateFocuses,
  warningFor,
  type FocusEntry,
  type Unit,
} from "./training-focus.ts";

const entry = (key: string, over: Partial<FocusEntry> = {}): FocusEntry => ({
  key,
  label: key.toUpperCase(),
  group: "scale",
  phase: 1,
  description: "Beschreibung.",
  signs: "Zeichen.",
  fits: ["hall", "arena"],
  ...over,
});

const catalog: FocusEntry[] = [
  entry("takt", { label: "Takt" }),
  entry("losgelassenheit", { label: "Losgelassenheit" }),
  entry("anlehnung", { label: "Anlehnung" }),
  entry("versammlung", { label: "Versammlung", phase: 3 }),
  entry("gelassenheit", { label: "Gelassenheit", group: "practice", phase: 0, fits: ["groundwork", "hack"] }),
  entry("turnier", { label: "Turniervorbereitung", group: "practice", phase: 0, fits: ["hall", "arena", "jumping"] }),
];

// --- catalog ---------------------------------------------------------------------------

test("labels come from the catalog, then the fallback table, then the key", () => {
  assert.equal(focusLabel("takt", catalog), "Takt");
  assert.equal(focusLabel("takt", [entry("takt", { label: "Rhythmus" })]), "Rhythmus", "the catalog wins");
  assert.equal(focusLabel("durchlaessigkeit"), "Durchlässigkeit");
  assert.equal(focusLabel("durchlaessigkeit", catalog), "Durchlässigkeit", "not in this catalog: fallback table");
  assert.equal(focusLabel("geraderichten"), "Geraderichtung");
  assert.equal(focusLabel("mobilitaet"), "Beweglichkeit");
  assert.equal(focusLabel("turnier"), "Turniervorbereitung");
  assert.equal(focusLabel("neue_sache"), "Neue sache");
  assert.equal(focusLabels(["takt", "anlehnung"], catalog), "Takt, Anlehnung");
  assert.equal(focusLabels([]), "");
});

test("the catalog is grouped Skala der Ausbildung, then Praxis", () => {
  assert.equal(groupLabel("scale"), "Skala der Ausbildung");
  assert.equal(groupLabel("practice"), "Praxis");
  assert.equal(groupLabel("extra"), "Extra");
  const sections = focusSections(catalog);
  assert.deepEqual(
    sections.map((s) => [s.label, s.entries.map((e) => e.key)]),
    [
      ["Skala der Ausbildung", ["takt", "losgelassenheit", "anlehnung", "versammlung"]],
      ["Praxis", ["gelassenheit", "turnier"]],
    ],
  );
  // Order of the groups does not depend on the order of the entries; unknown groups come last; empty ones are left out.
  const mixed = focusSections([entry("a", { group: "extra" }), entry("b", { group: "practice" }), entry("c", { group: "scale" })]);
  assert.deepEqual(mixed.map((s) => s.group), ["scale", "practice", "extra"]);
  assert.deepEqual(focusSections([]), []);
  assert.deepEqual(focusSections([entry("b", { group: "practice" })]).map((s) => s.group), ["practice"]);
});

test("the focus picker of a day offers what fits the activity, plus the current key", () => {
  assert.deepEqual(focusChoices(catalog, "hall").map((e) => e.key), ["takt", "losgelassenheit", "anlehnung", "versammlung", "turnier"]);
  assert.deepEqual(focusChoices(catalog, "groundwork").map((e) => e.key), ["gelassenheit"]);
  assert.deepEqual(focusChoices(catalog, "hack", "takt").map((e) => e.key), ["takt", "gelassenheit"], "current key stays even when it does not fit");
  assert.deepEqual(focusChoices(catalog, null), []);
  assert.deepEqual(focusChoices(catalog, null, "turnier").map((e) => e.key), ["turnier"]);
  assert.deepEqual(focusChoices([], "hall"), []);
});

// --- phase -----------------------------------------------------------------------------

test("phase names, catalog phases and the above-phase test", () => {
  assert.equal(phaseName("gewoehnung"), "Gewöhnungsphase");
  assert.equal(phaseName("schubkraft"), "Entwicklung der Schubkraft");
  assert.equal(phaseName("tragkraft"), "Entwicklung der Tragkraft");
  assert.equal(phaseName("anders"), null);
  assert.equal(phaseName(undefined), null);
  assert.equal(catalogPhaseOf("gewoehnung"), 1);
  assert.equal(catalogPhaseOf("schubkraft"), 2);
  assert.equal(catalogPhaseOf("tragkraft"), 3);
  assert.equal(catalogPhaseOf(null), null);
  assert.equal(isAbovePhase({ phase: 3 }, 1), true);
  assert.equal(isAbovePhase({ phase: 2 }, 2), false);
  assert.equal(isAbovePhase({ phase: 0 }, 1), false, "phase 0 fits every phase");
  assert.equal(isAbovePhase({ phase: 3 }, null), false, "unknown horse phase: no claim");
  assert.equal(isAbovePhase({ phase: 3 }, undefined), false);
});

test("the phase hint is built from the horse's phase and the scale keys still open", () => {
  assert.equal(
    phaseHint("Luna", "gewoehnung", ["takt", "losgelassenheit", "anlehnung"], catalog),
    "Luna ist in der Gewöhnungsphase: Takt, Losgelassenheit, Anlehnung",
  );
  assert.equal(phaseHint("Luna", "schubkraft", ["geraderichten", "schwung"]), "Luna ist in der Entwicklung der Schubkraft: Geraderichtung, Schwung");
  assert.equal(phaseHint("Luna", "tragkraft", []), "Luna ist in der Entwicklung der Tragkraft.");
  assert.equal(phaseHint("Luna", "tragkraft", undefined), "Luna ist in der Entwicklung der Tragkraft.");
  assert.equal(phaseHint(" ", "gewoehnung", ["takt"]), "Das Pferd ist in der Gewöhnungsphase: Takt");
  assert.equal(phaseHint("Luna", undefined, ["takt"]), null, "older backend: no phase, no hint");
  assert.equal(phaseHint("Luna", "unbekannt", ["takt"]), null);
});

// --- selection -------------------------------------------------------------------------

test("limits: three for the horse, two for one's own focus", () => {
  assert.equal(FOCUS_LIMIT_OWNER, 3);
  assert.equal(FOCUS_LIMIT_RIDER, 2);
  assert.equal(selectionCaption(2, 3), "2 von 3 gewählt");
  assert.equal(limitNotice(2), "Höchstens 2 Schwerpunkte. Nimm erst einen heraus.");
});

test("toggling keys adds at the end, removes, and stops at the limit", () => {
  assert.deepEqual(toggleKey([], "takt", 2), { keys: ["takt"], full: false });
  assert.deepEqual(toggleKey(["takt"], "anlehnung", 2), { keys: ["takt", "anlehnung"], full: false });
  assert.deepEqual(toggleKey(["takt", "anlehnung"], "versammlung", 2), { keys: ["takt", "anlehnung"], full: true });
  assert.deepEqual(toggleKey(["takt", "anlehnung"], "takt", 2), { keys: ["anlehnung"], full: false }, "removing works at the limit");
  const keys = ["takt"];
  toggleKey(keys, "anlehnung", 2);
  assert.deepEqual(keys, ["takt"], "the input stays untouched");
});

test("drafts: toggle with an end date, order, change the date, PUT shape", () => {
  const a = toggleFocus([], "turnier", 3, "2026-10-28");
  assert.deepEqual(a.drafts, [{ key: "turnier", until: "2026-10-28" }]);
  const b = toggleFocus(a.drafts, "takt", 3);
  assert.deepEqual(b.drafts, [
    { key: "turnier", until: "2026-10-28" },
    { key: "takt", until: "" },
  ]);
  const c = toggleFocus(toggleFocus(b.drafts, "anlehnung", 3).drafts, "versammlung", 3);
  assert.equal(c.full, true);
  assert.equal(c.drafts.length, 3);
  assert.deepEqual(toggleFocus(b.drafts, "turnier", 3).drafts, [{ key: "takt", until: "" }]);

  assert.deepEqual(moveFocus(b.drafts, 1, -1).map((d) => d.key), ["takt", "turnier"]);
  assert.deepEqual(moveFocus(b.drafts, 0, 1).map((d) => d.key), ["takt", "turnier"]);
  assert.deepEqual(moveFocus(b.drafts, 0, -1).map((d) => d.key), ["turnier", "takt"], "the first cannot move up");
  assert.deepEqual(moveFocus(b.drafts, 1, 1).map((d) => d.key), ["turnier", "takt"], "the last cannot move down");
  assert.deepEqual(moveFocus(b.drafts, 5, -1).map((d) => d.key), ["turnier", "takt"]);

  assert.deepEqual(setFocusUntil(b.drafts, "takt", "2026-11-01")[1], { key: "takt", until: "2026-11-01" });
  assert.deepEqual(focusInputs(b.drafts), [{ key: "turnier", until: "2026-10-28" }, { key: "takt" }]);
  assert.deepEqual(focusDraftsFrom([{ key: "takt" }, { key: "turnier", until: "2026-10-28" }, { key: "x", until: null }]), [
    { key: "takt", until: "" },
    { key: "turnier", until: "2026-10-28" },
    { key: "x", until: "" },
  ]);
  assert.deepEqual(focusDraftsFrom(undefined), []);
  assert.deepEqual(focusDraftsFrom(null), []);
});

test("validation of the selection is German", () => {
  const ok = [{ key: "takt", until: "" }, { key: "turnier", until: "2026-10-28" }];
  assert.equal(validateFocuses(ok, 3), null);
  assert.equal(validateFocuses([], 3), null);
  assert.equal(validateFocuses([...ok, { key: "anlehnung", until: "" }], 2), "Höchstens 2 Schwerpunkte.");
  assert.equal(validateFocuses([...ok, { key: "takt", until: "" }], 3), "Jeden Schwerpunkt nur einmal wählen.");
  assert.match(validateFocuses([{ key: "takt", until: "28.10." }], 3) ?? "", /gültiges Datum/);
  assert.match(validateFocuses([{ key: "takt", until: "2026-02-30" }], 3) ?? "", /gültiges Datum/);
});

test("summaries for the profile and the training tab", () => {
  assert.equal(focusesSummary(undefined), "Keine gesetzt");
  assert.equal(focusesSummary([]), "Keine gesetzt");
  assert.equal(focusesSummary([{ key: "takt" }, { key: "turnier", until: "2026-10-28" }], catalog), "Takt, Turniervorbereitung (bis 28.10.)");
  assert.equal(focusesSummary([{ key: "takt", until: "kaputt" }], catalog), "Takt", "an unreadable date is left out");
  assert.equal(myFocusSummary(["gelassenheit"], catalog), "Gelassenheit");
  assert.equal(myFocusSummary(["takt", "anlehnung"], catalog), "Takt, Anlehnung");
  assert.equal(myFocusSummary([]), "Kein eigener Schwerpunkt");
  assert.equal(myFocusSummary(undefined), "Kein eigener Schwerpunkt");
});

// --- the show a Turniervorbereitung ends with ------------------------------------------

test("the next show is the first date from today on", () => {
  const dates = ["2026-11-14", "2026-10-28", "2026-09-12", "kaputt", "2026-10-25"];
  assert.equal(nextShowDate(dates, "2026-10-25"), "2026-10-25", "today counts");
  assert.equal(nextShowDate(dates, "2026-10-26"), "2026-10-28");
  assert.equal(nextShowDate(dates, "2026-11-15"), null);
  assert.equal(nextShowDate([], "2026-10-25"), null);
});

test("turnier starts with the next show date, everything else without an end", () => {
  const shows = ["2026-09-12", "2026-10-28"];
  assert.equal(defaultUntil("turnier", shows, "2026-10-25"), "2026-10-28");
  assert.equal(defaultUntil("takt", shows, "2026-10-25"), "");
  assert.equal(defaultUntil("turnier", shows, "2026-11-01"), "", "no show ahead: no date (fallback)");
  assert.equal(defaultUntil("turnier", [], "2026-10-25"), "");
});

// --- warnings --------------------------------------------------------------------------

test("warnings: the level goal's is not a focus warning", () => {
  const warnings = [
    { key: "versammlung", text: "Versammlung setzt Schwung und Geraderichtung voraus." },
    { key: LEVEL_UP_KEY, text: "Luna hat 10 von 12 L-Lektionen gefestigt – Einstufung auf L anpassen?" },
  ];
  assert.deepEqual(focusWarnings(warnings).map((w) => w.key), ["versammlung"]);
  assert.deepEqual(focusWarnings(undefined), []);
  assert.equal(warningFor("versammlung", warnings), "Versammlung setzt Schwung und Geraderichtung voraus.");
  assert.equal(warningFor("takt", warnings), null);
  assert.equal(warningFor(LEVEL_UP_KEY, warnings), null);
  assert.equal(warningFor("takt", undefined), null);
});

test("focus note: the server's text, else the above-phase hint before saving", () => {
  const versammlung = catalog.find((e) => e.key === "versammlung")!;
  const takt = catalog.find((e) => e.key === "takt")!;
  const warnings = [{ key: "versammlung", text: "Versammlung setzt Schwung und Geraderichtung voraus." }];
  assert.equal(focusNote(versammlung, warnings, 1, "Luna"), "Versammlung setzt Schwung und Geraderichtung voraus.");
  assert.equal(focusNote(versammlung, [], 1, "Luna"), "Über der Phase von Luna: wird höchstens einmal pro Woche geplant.");
  assert.equal(focusNote(versammlung, undefined, 3, "Luna"), null, "within the phase");
  assert.equal(focusNote(versammlung, undefined, null, "Luna"), null, "phase unknown");
  assert.equal(focusNote(takt, [], 1, "Luna"), null);
  assert.equal(focusNote(undefined, warnings, 1, "Luna"), null);
  assert.match(focusNote(versammlung, [], 1, " ") ?? "", /dem Pferd/);
});

// --- focus of a day --------------------------------------------------------------------

test("a day's chip: the API's label, else the key's, free text is not a chip", () => {
  assert.equal(dayFocusLabel({ focus_key: "takt", focus_label: "Takt" }), "Takt");
  assert.equal(dayFocusLabel({ focus_key: "takt" }, catalog), "Takt");
  assert.equal(dayFocusLabel({ focus_key: "mobilitaet" }), "Beweglichkeit");
  assert.equal(dayFocusLabel({ focus_key: null, focus_label: null }), null);
  assert.equal(dayFocusLabel({}), null);
  assert.deepEqual(dayFocusLines({ focus_key: "takt", focus_label: "Takt", focus: "Übergänge Schritt-Trab" }), {
    chip: "Takt",
    text: "Übergänge Schritt-Trab",
  });
  assert.deepEqual(dayFocusLines({ focus: " Übergänge " }), { chip: null, text: "Schwerpunkt: Übergänge" });
  assert.deepEqual(dayFocusLines({ focus_key: "takt" }, catalog), { chip: "Takt", text: null });
  assert.deepEqual(dayFocusLines({ focus: null }), { chip: null, text: null });
});

test("a session is recorded with the focus it was started from, else the day's slot of the same activity", () => {
  const days = [
    { date: "2026-10-24", is_today: false, activity: "hall", focus_key: "anlehnung" },
    { date: "2026-10-25", is_today: true, activity: "hall", focus_key: "takt" },
    { date: "2026-10-26", is_today: false, activity: "hack", focus_key: "kondition" },
  ];
  assert.equal(sessionFocusKey({ requested: "schwung", activity: "hall", days }), "schwung", "the recommendation's focus wins");
  assert.equal(sessionFocusKey({ activity: "hall", days }), "takt", "today's slot");
  assert.equal(sessionFocusKey({ activity: "hall", date: "2026-10-24", days }), "anlehnung", "the day of the session");
  assert.equal(sessionFocusKey({ activity: "hack", days }), undefined, "the slot is another activity");
  assert.equal(sessionFocusKey({ activity: "hack", date: "2026-10-26", days }), "kondition");
  assert.equal(sessionFocusKey({ activity: "hall", date: "2026-10-30", days }), undefined, "no slot that day");
  assert.equal(sessionFocusKey({ activity: "hall" }), undefined, "week not loaded");
  assert.equal(sessionFocusKey({ requested: "", activity: "hall", days: [{ date: "2026-10-25", is_today: true, activity: "hall", focus_key: null }] }), undefined);
});

// --- the unit --------------------------------------------------------------------------

const unit: Unit = {
  warmup: { exercise: { id: "e1", title: "Cavaletti auf dem Zirkel" }, minutes: 15 },
  main: { exercise: { id: "e2", title: "Schulterherein" }, minutes: 27 },
  cooldown: { exercise: null, minutes: 8 },
};

test("unit rows: minutes, part, exercise in the order Lösen, Arbeiten, Entspannen", () => {
  const rows = unitRows(unit);
  assert.deepEqual(
    rows.map((r) => [r.part, r.label, r.minutes, r.exercise?.title ?? null]),
    [
      ["warmup", "Lösen", 15, "Cavaletti auf dem Zirkel"],
      ["main", "Arbeiten", 27, "Schulterherein"],
      ["cooldown", "Entspannen", 8, null],
    ],
  );
  assert.equal(rows[0]!.text, "15 Lösen · Cavaletti auf dem Zirkel");
  assert.equal(rows[1]!.text, "27 Arbeiten · Schulterherein");
  assert.equal(rows[2]!.text, "8 Entspannen", "a part without an exercise is free work");
  assert.equal(rows[0]!.accessibilityLabel, "Lösen, 15 Minuten: Cavaletti auf dem Zirkel");
  assert.equal(rows[2]!.accessibilityLabel, "Entspannen, 8 Minuten: ohne feste Übung");
  assert.equal(unitMinutes(unit), 50);
});

test("unit rows leave out absent parts and show what the equipment replaced", () => {
  assert.deepEqual(unitRows(undefined), []);
  assert.deepEqual(unitRows(null), []);
  assert.deepEqual(unitRows({}), []);
  assert.deepEqual(unitRows({ main: null, warmup: undefined }), []);
  const only = unitRows({ main: { exercise: { id: "e2", title: "Schulterherein" }, minutes: 30 } });
  assert.deepEqual(only.map((r) => r.part), ["main"]);
  assert.deepEqual(unitRows({ warmup: { exercise: null, minutes: 0 } }), [], "no minutes and no exercise: nothing to show");
  const instead = unitRows({
    warmup: { exercise: { id: "e3", title: "Tempiwechsel an der Longe" }, minutes: 10, instead_of: { id: "e4", title: "Trabstangen" } },
  })[0]!;
  assert.equal(instead.insteadOf, "statt Trabstangen");
  assert.equal(instead.text, "10 Lösen · Tempiwechsel an der Longe");
  assert.equal(instead.accessibilityLabel, "Lösen, 10 Minuten: Tempiwechsel an der Longe, statt Trabstangen");
  assert.equal(unitRows({ main: { exercise: null, minutes: 20, instead_of: null } })[0]!.insteadOf, null);
  assert.equal(unitMinutes(undefined), 0);
});

test("the main exercise: the unit's, else the single exercise of older backends", () => {
  assert.deepEqual(mainExercise({ unit, exercise: { id: "old", title: "Alt" } }), { id: "e2", title: "Schulterherein" });
  assert.deepEqual(mainExercise({ exercise: { id: "old", title: "Alt" } }), { id: "old", title: "Alt" });
  assert.deepEqual(mainExercise({ unit: { main: null }, exercise: { id: "old", title: "Alt" } }), { id: "old", title: "Alt" });
  assert.deepEqual(mainExercise({ unit: { main: { exercise: null, minutes: 20 } }, exercise: null }), null);
  assert.equal(mainExercise({}), null);
});

test("the body of a unit names exercises by id and lets a part have none", () => {
  assert.deepEqual(unitBody(unit), {
    warmup: { exercise_id: "e1", minutes: 15 },
    main: { exercise_id: "e2", minutes: 27 },
    cooldown: { exercise_id: null, minutes: 8 },
  });
  assert.deepEqual(unitBody({ main: { exercise: { id: "e2", title: "x" }, minutes: 29.6 } }), { main: { exercise_id: "e2", minutes: 30 } });
  assert.deepEqual(unitBody({}), {});
});

test("the split of a unit follows the table of the planning spec", () => {
  // hall, arena, jumping: Lösen 33 % (10 to 25), Entspannen 17 % (5 to 10), the rest is Arbeiten (at least 10)
  assert.deepEqual(unitSplit("hall", 45), { warmup: 15, main: 22, cooldown: 8 });
  assert.deepEqual(unitSplit("arena", 60), { warmup: 20, main: 30, cooldown: 10 });
  assert.deepEqual(unitSplit("hall", 30), { warmup: 10, main: 15, cooldown: 5 });
  assert.deepEqual(unitSplit("hall", 90), { warmup: 25, main: 55, cooldown: 10 }, "both are capped");
  assert.deepEqual(unitSplit("jumping", 30), { warmup: 15, main: 10, cooldown: 5 }, "jumping warms up at least 15");
  assert.deepEqual(unitSplit("jumping", 40), { warmup: 15, main: 18, cooldown: 7 });
  // lunge, groundwork, hack: fixed Lösen and Entspannen
  assert.deepEqual(unitSplit("lunge", 25), { warmup: 10, main: 10, cooldown: 5 });
  assert.deepEqual(unitSplit("groundwork", 30), { warmup: 5, main: 20, cooldown: 5 });
  assert.deepEqual(unitSplit("hack", 60), { warmup: 10, main: 40, cooldown: 10 });
  // walker and rest have no unit plan
  assert.equal(unitSplit("walker", 30), null);
  assert.equal(unitSplit("rest", 0), null);
  assert.equal(unitSplit("hall", 0), null);
  // seniors and cold nights: five more minutes of Lösen (capped at 30), the time comes off Arbeiten
  assert.deepEqual(unitSplit("hall", 45, 5), { warmup: 20, main: 17, cooldown: 8 });
  assert.deepEqual(unitSplit("hall", 90, 10), { warmup: 30, main: 50, cooldown: 10 });
  for (const [activity, minutes] of [["hall", 45], ["arena", 60], ["lunge", 25], ["hack", 90], ["groundwork", 30]] as const) {
    const s = unitSplit(activity, minutes)!;
    assert.equal(s.warmup + s.main + s.cooldown, minutes, `${activity} ${minutes}`);
  }
});

test("a new split keeps the chosen exercises and drops what no longer fits", () => {
  const split = resplitUnit(unit, "hall", 60)!;
  assert.deepEqual(unitRows(split).map((r) => [r.part, r.minutes, r.exercise?.id ?? null]), [
    ["warmup", 20, "e1"],
    ["main", 30, "e2"],
    ["cooldown", 10, null],
  ]);
  assert.equal(resplitUnit(unit, "walker", 30), undefined);
  assert.equal(resplitUnit(undefined, "hall", 45), undefined);
  assert.equal(resplitUnit(null, "hall", 45), undefined);
  const withInstead = resplitUnit({ main: { exercise: null, minutes: 20, instead_of: { id: "x", title: "Stangen" } } }, "hall", 45)!;
  assert.equal(withInstead.main?.instead_of, undefined, "the replaced exercise belongs to the old plan");

  const cleared = clearUnitExercises(unit, "lunge", 25)!;
  assert.deepEqual(unitRows(cleared).map((r) => [r.part, r.minutes, r.exercise]), [
    ["warmup", 10, null],
    ["main", 10, null],
    ["cooldown", 5, null],
  ]);
  assert.equal(clearUnitExercises(undefined, "lunge", 25), undefined);
  assert.equal(clearUnitExercises(unit, "walker", 25), undefined);
});

test("choosing an exercise for a part leaves the minutes, and makes a unit when there is none", () => {
  const ex = { id: "e9", title: "Neue Übung" };
  const changed = setUnitExercise(unit, "cooldown", ex, "hall", 50)!;
  assert.deepEqual(changed.cooldown, { exercise: ex, minutes: 8 });
  assert.deepEqual(changed.warmup, unit.warmup);
  assert.deepEqual(setUnitExercise(unit, "warmup", null, "hall", 50)!.warmup, { exercise: null, minutes: 15 });
  const fresh = setUnitExercise(undefined, "main", ex, "hall", 45)!;
  assert.deepEqual(fresh, {
    warmup: { exercise: null, minutes: 15 },
    main: { exercise: ex, minutes: 22 },
    cooldown: { exercise: null, minutes: 8 },
  });
  assert.equal(setUnitExercise(undefined, "main", ex, "walker", 30), undefined);
  const missing = setUnitExercise({ main: unit.main }, "warmup", ex, "hall", 45)!;
  assert.deepEqual(missing.warmup, { exercise: ex, minutes: 15 }, "an absent part takes its share of the split");
  assert.equal(unit.cooldown?.exercise, null, "the input stays untouched");
});

test("which exercises fit which part", () => {
  assert.equal(phaseFitsPart("warmup", "warmup"), true);
  assert.equal(phaseFitsPart("warmup", "main"), false);
  assert.equal(phaseFitsPart("main", "main"), true);
  assert.equal(phaseFitsPart("cooldown", "cooldown"), true);
  assert.equal(phaseFitsPart("cooldown", "warmup"), false);
  for (const part of ["warmup", "main", "cooldown"] as const) assert.equal(phaseFitsPart("any", part), true);
  assert.equal(phaseFitsPart(undefined, "main"), true, "older backends: an exercise is a main exercise");
  assert.equal(phaseFitsPart(undefined, "warmup"), false);
  assert.equal(phaseFitsPart("", "main"), true);
});

// --- the block of the week -------------------------------------------------------------

test("block text: build weeks, deload week, nothing for an older backend", () => {
  assert.equal(BLOCK_WEEKS, 4);
  assert.equal(blockText({ week: 2, deload: false }), "Aufbauwoche 2 von 4");
  assert.equal(blockText({ week: 1, deload: false }), "Aufbauwoche 1 von 4");
  assert.equal(blockText({ week: 4, deload: true }), "Entlastungswoche");
  assert.equal(blockText({ week: 9, deload: false }), null);
  assert.equal(blockText({ week: 0, deload: false }), null);
  assert.equal(blockText(undefined), null);
  assert.equal(blockText(null), null);
  assert.equal(restartText(1), "Wiederantrainieren, Woche 1");
  assert.equal(restartText(0), null);
  assert.equal(restartText(undefined), null);
});

test("eyebrow of the training tab", () => {
  assert.equal(trainingEyebrow("Fit", 0, { week: 2, deload: false }), "Training · Fit · Aufbauwoche 2 von 4");
  assert.equal(trainingEyebrow("Fit", 0, { week: 4, deload: true }), "Training · Fit · Entlastungswoche");
  assert.equal(trainingEyebrow("Fit", 2, { week: 2, deload: false }), "Training · Fit · Wiederantrainieren, Woche 2");
  assert.equal(trainingEyebrow("Fit", undefined, undefined), "Training · Fit");
  assert.equal(trainingEyebrow(null, undefined, undefined), "Training");
});
