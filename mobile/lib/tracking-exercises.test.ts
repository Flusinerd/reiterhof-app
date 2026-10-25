import assert from "node:assert/strict";
import { test } from "node:test";

import {
  NO_FILTER,
  applyFilter,
  checklistLabel,
  disciplineLabel,
  exerciseIntensityLabel,
  exercisePhaseText,
  exerciseQueryString,
  exerciseRowDetail,
  filterOptions,
  hasFilter,
  levelLabel,
  libraryLabel,
  libraryOf,
  minutesText,
  phaseLabel,
  phaseOptions,
  sourceItems,
  stepCountLabel,
  tagLabel,
  toggleFilter,
} from "./tracking-exercises.ts";

const list = [
  { id: "1", title: "Übergänge", discipline: "dressage", level: "beginner", goal_tags: ["durchlaessigkeit", "losgelassenheit"] },
  { id: "2", title: "Travers", discipline: "dressage", level: "intermediate", goal_tags: ["seitengaenge", "biegung"] },
  { id: "3", title: "Cavaletti", discipline: "jumping", level: "beginner", goal_tags: ["rhythmus"] },
  { id: "4", title: "Neu", discipline: "", level: "", goal_tags: [] },
];

test("filters combine with and", () => {
  assert.equal(applyFilter(list, NO_FILTER).length, 4);
  assert.deepEqual(applyFilter(list, { ...NO_FILTER, discipline: "dressage" }).map((e) => e.id), ["1", "2"]);
  assert.deepEqual(applyFilter(list, { discipline: "dressage", level: "beginner", tag: null }).map((e) => e.id), ["1"]);
  assert.deepEqual(applyFilter(list, { ...NO_FILTER, tag: "rhythmus" }).map((e) => e.id), ["3"]);
  assert.deepEqual(applyFilter(list, { discipline: "jumping", level: "intermediate", tag: null }), []);
});

test("filter options list what exists, levels from easy to hard", () => {
  const o = filterOptions(list);
  assert.deepEqual(o.disciplines, ["dressage", "jumping"]);
  assert.deepEqual(o.levels, ["beginner", "intermediate"]);
  assert.deepEqual(o.tags, ["biegung", "durchlaessigkeit", "losgelassenheit", "rhythmus", "seitengaenge"]);
  assert.deepEqual(filterOptions([]), { disciplines: [], levels: [], tags: [] });
});

test("toggling a filter value selects and clears it", () => {
  const a = toggleFilter(NO_FILTER, "level", "beginner");
  assert.equal(a.level, "beginner");
  assert.equal(hasFilter(a), true);
  const b = toggleFilter(a, "level", "beginner");
  assert.equal(b.level, null);
  assert.equal(hasFilter(b), false);
  assert.equal(toggleFilter(a, "level", "advanced").level, "advanced");
});

test("labels are German with a readable fallback", () => {
  assert.equal(levelLabel("beginner"), "Einsteiger");
  assert.equal(levelLabel("expert"), "Expert");
  assert.equal(levelLabel(""), "");
  assert.equal(disciplineLabel("dressage"), "Dressur");
  assert.equal(disciplineLabel("some_new"), "Some new");
  assert.equal(tagLabel("seitengaenge"), "Seitengänge");
  assert.equal(tagLabel("neue_ziele"), "Neue ziele");
  assert.equal(stepCountLabel(1), "1 Schritt");
  assert.equal(stepCountLabel(4), "4 Schritte");
  assert.equal(checklistLabel(2, 5), "2 von 5 Schritten erledigt");
  assert.equal(checklistLabel(1, 1), "1 von 1 Schritt erledigt");
});

// --- libraries, phases, equipment of the catalog (docs/domains/training-planning.md) ------

test("new tags and libraries have German labels", () => {
  assert.equal(tagLabel("kondition"), "Kondition");
  assert.equal(tagLabel("ausdauer"), "Ausdauer");
  assert.equal(tagLabel("kraft"), "Kraft");
  assert.equal(tagLabel("gelaende"), "Gelände");
  assert.equal(tagLabel("mobilitaet"), "Beweglichkeit");
  assert.equal(tagLabel("turnier"), "Turnier");
  assert.equal(libraryLabel("cavaletti"), "Cavaletti");
  assert.equal(libraryLabel("hack"), "Gelände");
  assert.equal(libraryLabel("western"), "Western");
  for (const l of ["dressage", "jumping", "cavaletti", "groundwork", "lunge", "hack", "western"]) {
    assert.notEqual(libraryLabel(l), l, l);
    assert.equal(libraryLabel(l), disciplineLabel(l));
  }
  assert.equal(libraryLabel(""), "");
});

test("the library is `library`, else the discipline older backends send", () => {
  assert.equal(libraryOf({ discipline: "dressage", library: "cavaletti" }), "cavaletti");
  assert.equal(libraryOf({ discipline: "dressage" }), "dressage");
  assert.equal(libraryOf({ discipline: "dressage", library: "" }), "dressage");
  const mixed = [
    { id: "1", title: "A", discipline: "dressage", library: "cavaletti", level: "beginner", goal_tags: [] },
    { id: "2", title: "B", discipline: "dressage", level: "beginner", goal_tags: [] },
  ];
  assert.deepEqual(applyFilter(mixed, { ...NO_FILTER, discipline: "cavaletti" }).map((e) => e.id), ["1"]);
  assert.deepEqual(filterOptions(mixed).disciplines, ["cavaletti", "dressage"]);
});

test("the phase filter keeps what fits the part of the unit", () => {
  const phased = [
    { id: "1", title: "Lösen", discipline: "dressage", level: "beginner", goal_tags: [], phase: "warmup" },
    { id: "2", title: "Arbeit", discipline: "dressage", level: "beginner", goal_tags: [], phase: "main" },
    { id: "3", title: "Aus", discipline: "dressage", level: "beginner", goal_tags: [], phase: "cooldown" },
    { id: "4", title: "Alles", discipline: "dressage", level: "beginner", goal_tags: [], phase: "any" },
  ];
  assert.deepEqual(applyFilter(phased, { ...NO_FILTER, phase: "warmup" }).map((e) => e.id), ["1", "4"]);
  assert.deepEqual(applyFilter(phased, { ...NO_FILTER, phase: "main" }).map((e) => e.id), ["2", "4"]);
  assert.deepEqual(applyFilter(phased, { ...NO_FILTER, phase: "cooldown" }).map((e) => e.id), ["3", "4"]);
  assert.deepEqual(applyFilter(phased, { ...NO_FILTER, phase: null }).length, 4);
  assert.deepEqual(applyFilter(phased, { discipline: "dressage", level: "beginner", tag: null, phase: "main" }).map((e) => e.id), ["2", "4"]);
  assert.deepEqual(phaseOptions(phased), ["warmup", "main", "cooldown"]);
  assert.deepEqual(phaseOptions([phased[1]!]), ["main"]);
  assert.deepEqual(phaseOptions([phased[3]!]), ["warmup", "main", "cooldown"], "an `any` exercise fits every part");
  assert.deepEqual(phaseOptions(list), [], "older backend: no phases, no pills");
  assert.deepEqual(phaseOptions([]), []);
  assert.equal(phaseLabel("warmup"), "Lösen");
  assert.equal(phaseLabel("main"), "Arbeiten");
  assert.equal(phaseLabel("cooldown"), "Entspannen");
});

test("toggling the new filters and recognising an active filter", () => {
  const a = toggleFilter(NO_FILTER, "phase", "main");
  assert.equal(a.phase, "main");
  assert.equal(hasFilter(a), true);
  assert.equal(toggleFilter(a, "phase", "main").phase, null);
  assert.equal(hasFilter(toggleFilter(a, "phase", "main")), false);
  assert.equal(toggleFilter(a, "phase", "warmup").phase, "warmup");
  const f = toggleFilter(NO_FILTER, "focus", "takt");
  assert.equal(f.focus, "takt");
  assert.equal(hasFilter(f), true);
  assert.equal(hasFilter(toggleFilter(f, "focus", "takt")), false);
  assert.equal(hasFilter(NO_FILTER), false);
});

test("phase, intensity and minutes of an exercise in words", () => {
  assert.equal(exercisePhaseText("warmup"), "Lösen");
  assert.equal(exercisePhaseText("main"), "Arbeiten");
  assert.equal(exercisePhaseText("cooldown"), "Entspannen");
  assert.equal(exercisePhaseText("any"), "Lösen, Arbeiten oder Entspannen");
  assert.equal(exercisePhaseText(undefined), "");
  assert.equal(exerciseIntensityLabel("light"), "leicht");
  assert.equal(exerciseIntensityLabel("normal"), "normal");
  assert.equal(exerciseIntensityLabel("demanding"), "fordernd");
  assert.equal(exerciseIntensityLabel("neu"), "neu");
  assert.equal(exerciseIntensityLabel(undefined), "");
  assert.equal(minutesText(10), "10 Min.");
  assert.equal(minutesText(7.6), "8 Min.");
  assert.equal(minutesText(null), "");
  assert.equal(minutesText(0), "");
  assert.equal(minutesText(undefined), "");
});

test("a library row says library, minutes, steps and goals; a missing set-up comes first", () => {
  const e = { id: "1", title: "Trabstangen", discipline: "dressage", library: "cavaletti", level: "beginner", goal_tags: ["rhythmus", "takt"], minutes: 10, step_count: 4 };
  assert.equal(exerciseRowDetail(e), "Cavaletti · 10 Min. · 4 Schritte · Rhythmus, Takt");
  assert.equal(exerciseRowDetail({ ...e, available: false }), "Ausstattung fehlt · Cavaletti · 10 Min. · 4 Schritte · Rhythmus, Takt");
  assert.equal(exerciseRowDetail({ ...e, available: true }), "Cavaletti · 10 Min. · 4 Schritte · Rhythmus, Takt");
  // An older backend: discipline only, no minutes.
  assert.equal(
    exerciseRowDetail({ id: "2", title: "Travers", discipline: "dressage", level: "intermediate", goal_tags: [], step_count: 1 }),
    "Dressur · 1 Schritt",
  );
  assert.equal(exerciseRowDetail({ id: "3", title: "X", discipline: "", level: "", goal_tags: [] }), "");
});

test("the query of the exercise list names the horse and the focus", () => {
  assert.equal(exerciseQueryString({}), "");
  assert.equal(exerciseQueryString({ horse: null, focus: null }), "");
  assert.equal(exerciseQueryString({ horse: "h 1" }), "?horse=h%201");
  assert.equal(exerciseQueryString({ focus: "takt" }), "?focus=takt");
  assert.equal(exerciseQueryString({ horse: "h1", focus: "takt" }), "?horse=h1&focus=takt");
});

test("sources: links are opened, plain titles are only shown", () => {
  const items = sourceItems([
    "https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-09-2021/",
    "https://www.wehorse.com/de/blog/zuegel-aus-der-hand-kauen-lassen/",
    "https://de.wikipedia.org/wiki/Travers_(Reitsport)?x=1#y",
    "FN Richtlinien, Band 1",
    "  ",
    "javascript:alert(1)",
    "http://example.org",
    "https://exa mple.org/x",
  ]);
  assert.deepEqual(items, [
    { title: "magazin.pferdesport-deutschland.de", detail: "/project/lektion-im-fokus-09-2021", url: "https://magazin.pferdesport-deutschland.de/project/lektion-im-fokus-09-2021/" },
    { title: "wehorse.com", detail: "/de/blog/zuegel-aus-der-hand-kauen-lassen", url: "https://www.wehorse.com/de/blog/zuegel-aus-der-hand-kauen-lassen/" },
    { title: "de.wikipedia.org", detail: "/wiki/Travers_(Reitsport)", url: "https://de.wikipedia.org/wiki/Travers_(Reitsport)?x=1#y" },
    { title: "FN Richtlinien, Band 1", detail: "", url: null },
    { title: "javascript:alert(1)", detail: "", url: null },
    { title: "example.org", detail: "", url: "http://example.org" },
    { title: "https://exa mple.org/x", detail: "", url: null },
  ]);
  assert.deepEqual(sourceItems(undefined), []);
  assert.deepEqual(sourceItems([]), []);
});
