import assert from "node:assert/strict";
import { test } from "node:test";

import {
  allowedClasses,
  arenaLetters,
  classCeiling,
  clampWindow,
  defaultClass,
  defaultTitle,
  fitArena,
  forbiddenText,
  formatSeconds,
  letterOffsets,
  letterPosition,
  moveLine,
  movementHint,
  musicParts,
  normalizeCatalog,
  normalizePlan,
  pathToSvg,
  pointAlong,
  positionAt,
  reinBalanceText,
  removeLine,
  renumber,
  requirementRows,
  requirementsReady,
  requirementSummary,
  setSummary,
  shareText,
  spokenSeconds,
  splitChecks,
  toggleMovement,
  updateMovement,
  type PlanLine,
} from "./dressage.ts";

const at = (arena: "20x40" | "20x60", letter: string) => arenaLetters(arena).find((l) => l.letter === letter)!;

test("letters of the 20x40 arena", () => {
  assert.deepEqual([at("20x40", "A").x, at("20x40", "A").y], [10, 0]);
  assert.deepEqual([at("20x40", "C").x, at("20x40", "C").y], [10, 40]);
  assert.deepEqual(letterOffsets("20x40"), [6, 20, 34]);
  assert.deepEqual([at("20x40", "K").x, at("20x40", "K").y], [0, 6]);
  assert.deepEqual([at("20x40", "M").x, at("20x40", "M").y], [20, 34]);
  assert.deepEqual([at("20x40", "X").x, at("20x40", "X").y], [10, 20]);
  assert.equal(at("20x40", "D").y, 6);
  assert.equal(at("20x40", "G").y, 34);
  assert.equal(arenaLetters("20x40").length, 2 + 9);
});

test("letters of the 20x60 arena", () => {
  assert.deepEqual(letterOffsets("20x60"), [6, 18, 30, 42, 54]);
  assert.equal(at("20x60", "S").y, 42);
  assert.equal(at("20x60", "R").x, 20);
  assert.equal(at("20x60", "X").y, 30);
  assert.equal(at("20x60", "I").y, 42);
  assert.equal(at("20x60", "C").y, 60);
  assert.equal(arenaLetters("20x60").length, 2 + 15);
});

test("fitArena keeps the proportions inside the box, A at the bottom", () => {
  const fit = fitArena("20x60", 340, 520);
  assert.ok(fit.width <= 340 + 0.001 && fit.height <= 520 + 0.001);
  const [ax, ay] = fit.toPx(10, 0);
  const [cx, cy] = fit.toPx(10, 60);
  assert.equal(ax, cx);
  assert.ok(ay > cy);
  assert.ok(Math.abs(ay - fit.pad - fit.scale * 60) < 1e-6);
  const wide = fitArena("20x40", 340, 2000);
  assert.ok(wide.width <= 340 + 0.001);
  assert.ok(Math.abs(wide.height / wide.width - (40 + 4.8) / (20 + 4.8)) < 1e-6);
});

test("left stays left, letters sit outside the fence", () => {
  const fit = fitArena("20x40", 300, 600);
  const k = letterPosition(fit, at("20x40", "K"));
  const f = letterPosition(fit, at("20x40", "F"));
  assert.ok(k[0] < fit.pad && f[0] > fit.pad + 20 * fit.scale);
  const a = letterPosition(fit, at("20x40", "A"));
  assert.ok(a[1] > fit.pad + 40 * fit.scale);
});

test("pathToSvg draws a polyline and drops repeated points", () => {
  const fit = fitArena("20x40", 248, 448); // scale 10, pad 24
  assert.equal(fit.scale, 10);
  assert.equal(pathToSvg([[10, 0], [10, 0], [10, 20]], fit), "M 124 424 L 124 224");
  assert.equal(pathToSvg([], fit), "");
});

test("formatSeconds and spokenSeconds", () => {
  assert.equal(formatSeconds(0), "0:00");
  assert.equal(formatSeconds(65), "1:05");
  assert.equal(formatSeconds(239.6), "4:00");
  assert.equal(formatSeconds(-3), "0:00");
  assert.equal(formatSeconds(Number.NaN), "0:00");
  assert.equal(spokenSeconds(65), "1 Minute 5 Sekunden");
  assert.equal(spokenSeconds(120), "2 Minuten");
  assert.equal(spokenSeconds(1), "1 Sekunde");
});

test("rein balance text", () => {
  assert.equal(reinBalanceText({ left: 48, right: 52 }), "Beide Hände ausgewogen: links 48 %, rechts 52 %");
  assert.equal(reinBalanceText({ left: 70, right: 30 }), "Linke Hand überwiegt: links 70 %, rechts 30 %");
  assert.equal(reinBalanceText({ left: 20, right: 80 }), "Rechte Hand überwiegt: links 20 %, rechts 80 %");
  assert.equal(reinBalanceText({ left: 0, right: 0 }), "");
  assert.equal(reinBalanceText({ left: 1, right: 2 }), "Rechte Hand überwiegt: links 33 %, rechts 67 %");
});

test("requirement checklist puts open items first", () => {
  const reqs = [
    { key: "a", label: "Mitteltrab", count: 1, met: true },
    { key: "b", label: "Volte im Galopp", count: 2, met: false },
  ];
  assert.deepEqual(requirementRows(reqs), [
    { key: "b", text: "Volte im Galopp (2×)", met: false },
    { key: "a", text: "Mitteltrab", met: true },
  ]);
  assert.equal(requirementSummary(reqs), "1 von 2 Anforderungen erfüllt");
  assert.equal(requirementSummary([reqs[0]!]), "Alle Anforderungen erfüllt");
  assert.equal(requirementSummary([]), "");
});

test("music parts group consecutive gaits with start and end", () => {
  const parts = musicParts([
    { gait: "trot", seconds: 30 },
    { gait: "trot", seconds: 20 },
    { gait: "walk", seconds: 40 },
    { gait: "canter", seconds: 60 },
  ]);
  assert.deepEqual(
    parts.map((p) => [p.label, p.seconds, p.from, p.to]),
    [
      ["Trab", 50, 0, 50],
      ["Schritt", 40, 50, 90],
      ["Galopp", 60, 90, 150],
    ],
  );
  assert.deepEqual(musicParts([]), []);
});

const line = (n: number, seconds: number, path: [number, number][]): PlanLine => ({ n, letters: "A–X", text: `Zeile ${n}`, gait: "trot", seconds, path });

test("edit helpers reorder, remove and renumber", () => {
  const lines = [line(1, 10, []), line(2, 10, []), line(3, 10, [])];
  assert.deepEqual(moveLine(lines, 2, -1).map((l) => l.text), ["Zeile 1", "Zeile 3", "Zeile 2"]);
  assert.deepEqual(moveLine(lines, 0, -1).map((l) => l.n), [1, 2, 3]);
  assert.deepEqual(renumber(removeLine(lines, 0)).map((l) => l.n), [1, 2]);
});

test("pointAlong and positionAt follow the path in time", () => {
  assert.deepEqual(pointAlong([[0, 0], [0, 10]], 0.5), [0, 5]);
  assert.deepEqual(pointAlong([[0, 0], [10, 0], [10, 10]], 0.75), [10, 5]);
  assert.equal(pointAlong([], 0.5), null);
  const lines = [line(1, 10, [[0, 0], [0, 10]]), line(2, 10, [[0, 10], [10, 10]])];
  assert.deepEqual(positionAt(lines, 5), { index: 0, point: [0, 5] });
  assert.deepEqual(positionAt(lines, 15), { index: 1, point: [5, 10] });
  assert.deepEqual(positionAt(lines, 99), { index: 1, point: [10, 10] });
  assert.equal(positionAt([line(1, 5, [])], 1), null);
});

test("normalizePlan reads a preview and a stored test, and survives missing parts", () => {
  const raw = {
    kind: "freestyle",
    class: "L*",
    arena: "20x60",
    seed: 7,
    lines: [{ n: 1, letters: "A–X", text: "Einreiten", gait: "trot", seconds: 12, path: [[10, 0], [10, 30]] }],
    seconds: 12,
    reins: { left: 1, right: 1 },
    music: [{ gait: "trot", seconds: 12 }],
    requirements: [{ key: "m", label: "Mitteltrab", count: 1, met: true }],
  };
  const plan = normalizePlan(raw)!;
  assert.equal(plan.kind, "freestyle");
  assert.equal(plan.arena, "20x60");
  assert.deepEqual(plan.checks, []);
  assert.deepEqual(plan.lines[0]!.path, [[10, 0], [10, 30]]);
  const withOption = normalizePlan({ ...raw, lines: [{ ...raw.lines[0], option: "einreiten:X" }] })!;
  assert.equal(withOption.lines[0]!.option, "einreiten:X");
  assert.equal(JSON.parse(JSON.stringify(withOption)).lines[0].option, "einreiten:X");
  assert.equal(plan.lines[0]!.option, undefined);
  const stored = normalizePlan({ id: "t1", title: "x", plan: raw })!;
  assert.equal(stored.seed, 7);
  assert.equal(normalizePlan({ nothing: true }), null);
  assert.equal(normalizePlan(null), null);
});

test("share text and default title", () => {
  const plan = normalizePlan({
    kind: "practice",
    class: "E",
    arena: "20x40",
    lines: [{ n: 1, letters: "A–X", text: "Einreiten im Arbeitstrab", gait: "trot", seconds: 12, path: [] }],
    seconds: 12,
    reins: { left: 1, right: 1 },
  })!;
  const text = shareText("Übungsaufgabe E · 20×40", plan);
  assert.ok(text.startsWith("Übungsaufgabe E · 20×40\n20×40 · 0:12 Min."));
  assert.ok(text.includes("1. A–X: Einreiten im Arbeitstrab (0:12)"));
  assert.ok(text.includes("Beide Hände ausgewogen"));
  assert.ok(!text.includes("Musikteile"));
  assert.equal(defaultTitle("practice", "E", "20x40"), "Übungsaufgabe E · 20×40");
  assert.equal(defaultTitle("freestyle", "L*", "20x60"), "Kür L* · 20×60");
});

test("requirement editor helpers", () => {
  let list = toggleMovement([], "mitteltrab");
  list = toggleMovement(list, "volte");
  assert.deepEqual(list, [
    { key: "mitteltrab", count: 1, rein: "either" },
    { key: "volte", count: 1, rein: "either" },
  ]);
  list = updateMovement(list, "volte", { count: 3, rein: "both" });
  assert.deepEqual(list[1], { key: "volte", count: 3, rein: "both" });
  assert.equal(updateMovement(list, "volte", { count: 0 })[1]!.count, 1);
  assert.equal(updateMovement(list, "volte", { count: 99 })[1]!.count, 9);
  assert.deepEqual(toggleMovement(list, "volte").map((m) => m.key), ["mitteltrab"]);
  assert.deepEqual(clampWindow(240, 200), { min: 240, max: 240 });
  assert.equal(requirementsReady([], 210, 240), false);
  assert.equal(requirementsReady(list, 0, 240), false);
  assert.equal(requirementsReady(list, 210, 200), false);
  assert.equal(requirementsReady(list, 210, 240), true);
  assert.equal(setSummary({ id: "1", class: "L*", label: "x", movements: list, min_seconds: 210, max_seconds: 240 }), "3:30 bis 4:00 Min. · 2 Lektionen");
});

test("catalog: classes as key and label, movements without a class", () => {
  const cat = normalizeCatalog({
    classes: [{ key: "L*", label: "L*" }],
    movements: [
      { key: "a", label: "Mitteltrab", gait: "trot", lowest_class: "A" },
      { key: "b", label: "Fliegender Wechsel", gait: "canter" },
    ],
    requirement_sets: [{ id: "s1", class: "L*", label: "Meine Kür", movements: [{ key: "a", count: 2, rein: "left" }], min_seconds: 210, max_seconds: 240 }],
  });
  assert.deepEqual(cat.classes, [{ key: "L*", label: "L*" }]);
  assert.equal(movementHint(cat.movements[0]!), "Trab · ab Klasse A");
  assert.equal(movementHint(cat.movements[1]!), "Galopp · Klasse nicht belegt");
  assert.equal(cat.requirement_sets[0]!.movements[0]!.rein, "left");
  assert.deepEqual(normalizeCatalog(undefined), { movements: [], classes: [], requirement_sets: [] });
});

test("classes: ceiling by goal or level and the default", () => {
  const all = ["E", "A", "L", "L**"];
  assert.deepEqual(allowedClasses(all, classCeiling("beginner", null)), ["E", "A"]);
  assert.deepEqual(allowedClasses(all, classCeiling("beginner", "intermediate")), all);
  assert.deepEqual(allowedClasses(all, classCeiling("advanced", null)), all);
  assert.deepEqual(allowedClasses(all, classCeiling("", null)), all);
  assert.equal(defaultClass(all, "intermediate"), "L");
  assert.equal(defaultClass(all, "advanced"), "L**");
  assert.equal(defaultClass(["A", "L*"], "beginner"), "A");
  assert.equal(defaultClass(["A", "L*", "L**"], "intermediate"), "A");
});

test("checks are split into rules and advice, failing ones first", () => {
  const plan = normalizePlan({
    lines: [],
    checks: [
      { key: "chain", ok: true, rule: true, text: "a" },
      { key: "time", ok: false, rule: true, text: "b" },
      { key: "balance", ok: true, rule: false, text: "c" },
      { key: "finale", ok: false, rule: false, text: "d" },
      { key: "old", ok: true, text: "e" },
    ],
  })!;
  const { rules, advice } = splitChecks(plan.checks);
  assert.deepEqual(rules.map((k) => k.key), ["time", "chain", "old"]);
  assert.deepEqual(advice.map((k) => k.key), ["finale", "balance"]);
});

test("forbiddenText names who may use and who may delete", () => {
  assert.match(forbiddenText(), /Besitzer.*Admins.*Reiter/);
  assert.match(forbiddenText("delete"), /Löschen.*erstellt/);
});
