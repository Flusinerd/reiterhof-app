import assert from "node:assert/strict";
import { test } from "node:test";

import {
  checkPhase,
  checkupText,
  dateRange,
  draftFromPlan,
  draftToInput,
  emptyPlanDraft,
  minutesRange,
  movePhase,
  newPhaseDraft,
  phaseSummary,
  placePhases,
  planEnd,
  planProgress,
  planStatusText,
  putPhase,
  rampMinutes,
  rehaErrorText,
  removePhase,
  weekRehaText,
  totalDays,
  type PhaseDraft,
  type PlanDraft,
  type PlanLike,
} from "./reha.ts";

test("rampMinutes rises linearly and rounds half up", () => {
  // The example of the specification: Schritt führen 10 -> 20 Min. over 9 days.
  assert.deepEqual(
    Array.from({ length: 9 }, (_, i) => rampMinutes(10, 20, i, 9)),
    [10, 11, 13, 14, 15, 16, 18, 19, 20],
  );
  assert.equal(rampMinutes(15, 20, 0, 1), 20, "one day allows the maximum");
  assert.equal(rampMinutes(15, 15, 3, 7), 15);
  assert.equal(rampMinutes(2, 15, 13, 14), 15);
  assert.equal(rampMinutes(20, 40, 6, 14), 29);
  assert.equal(rampMinutes(10, 20, -1, 9), 10);
  assert.equal(rampMinutes(10, 20, 99, 9), 20);
});

const spec = [
  { name: "Boxenruhe", days: 5 },
  { name: "Schritt führen", days: 9 },
  { name: "Schritt reiten", days: 14 },
  { name: "Trab aufbauen", days: 14 },
];

test("placePhases lays phases end to end, also across the DST change", () => {
  const placed = placePhases("2026-03-20", spec);
  assert.deepEqual(
    placed.map((p) => [p.start, p.end]),
    [
      ["2026-03-20", "2026-03-24"],
      ["2026-03-25", "2026-04-02"],
      ["2026-04-03", "2026-04-16"],
      ["2026-04-17", "2026-04-30"],
    ],
  );
  assert.equal(totalDays(spec), 42);
  assert.equal(planEnd("2026-03-20", spec), "2026-04-30");
  // 2026-03-29 (spring) and 2026-10-25 (autumn) are ordinary days: dates are calendar dates.
  assert.deepEqual(placePhases("2026-03-28", [{ days: 3 }]).map((p) => p.end), ["2026-03-30"]);
  assert.deepEqual(placePhases("2026-10-24", [{ days: 3 }]).map((p) => p.end), ["2026-10-26"]);
});

test("labels", () => {
  assert.equal(minutesRange(10, 20), "10–20 Min.");
  assert.equal(minutesRange(15, 15), "15 Min.");
  assert.equal(minutesRange(0, 0, true), "keine Bewegung");
  assert.equal(dateRange("2026-03-22", "2026-03-30"), "22.03. – 30.03.");
  assert.equal(dateRange("2026-03-22", "2026-03-22"), "22.03.");
  assert.equal(phaseSummary({ activity: "groundwork", days: 9, min_minutes: 10, max_minutes: 20 }), "Bodenarbeit · 10–20 Min. · 9 Tage");
  assert.equal(phaseSummary({ activity: "rest", days: 1, min_minutes: 0, max_minutes: 0 }), "Boxenruhe · 1 Tag");
  assert.equal(weekRehaText({ phase: "Schritt führen", activity: "groundwork", rest: false, minutes: 14, done: true }), "Schritt führen: Bodenarbeit 14 Min. · erledigt");
  assert.equal(weekRehaText({ phase: "Boxenruhe", activity: "rest", rest: true, minutes: 0, done: false }), "Boxenruhe: keine Bewegung");
  assert.equal(checkupText(0), "Heute");
  assert.equal(checkupText(1), "Morgen");
  assert.equal(checkupText(16), "In 16 Tagen");
  assert.equal(checkupText(-1), "Seit gestern überfällig");
  assert.equal(checkupText(-3), "Seit 3 Tagen überfällig");
  assert.equal(planStatusText("running", 9, 42, "2026-03-17"), "Tag 9 von 42");
  assert.equal(planStatusText("upcoming", 0, 42, "2026-03-30"), "Startet am 30.03.2026");
  assert.equal(planStatusText("finished", 0, 42, "2026-03-01"), "Alle Phasen sind abgeschlossen");
  assert.equal(planProgress("running", 21, 42), 0.5);
  assert.equal(planProgress("finished", 0, 42), 1);
  assert.equal(planProgress("upcoming", 0, 42), 0);
  assert.equal(rehaErrorText("not_active"), "Dieser Reha-Plan ist bereits beendet.");
  assert.equal(rehaErrorText("nope"), null);
});

const phase = (over: Partial<PhaseDraft> = {}): PhaseDraft => ({
  name: "Schritt führen",
  activity: "groundwork",
  days: "9",
  minMinutes: "10",
  maxMinutes: "20",
  conditions: " nur Boden fest ",
  ...over,
});

test("checkPhase validates with German messages", () => {
  const ok = checkPhase(phase(), 1);
  assert.deepEqual(ok, {
    ok: true,
    phase: { name: "Schritt führen", days: 9, activity: "groundwork", min_minutes: 10, max_minutes: 20, conditions: "nur Boden fest" },
  });
  const err = (over: Partial<PhaseDraft>) => {
    const r = checkPhase(phase(over), 2);
    assert.equal(r.ok, false);
    return r.ok ? "" : r.error;
  };
  assert.match(err({ name: " " }), /^Phase 2: Bitte einen Namen/);
  assert.match(err({ name: "x".repeat(81) }), /80 Zeichen/);
  assert.match(err({ days: "0" }), /zwischen 1 und 365 Tagen/);
  assert.match(err({ days: "abc" }), /zwischen 1 und 365 Tagen/);
  assert.match(err({ days: "366" }), /zwischen 1 und 365 Tagen/);
  assert.match(err({ minMinutes: "" }), /Minuten müssen/);
  assert.match(err({ minMinutes: "0" }), /Minuten müssen/);
  assert.match(err({ maxMinutes: "241" }), /Minuten müssen/);
  assert.match(err({ minMinutes: "30" }), /„Von“ darf nicht größer/);
  assert.match(err({ conditions: "x".repeat(501) }), /500 Zeichen/);
  // A rest phase has no minutes, whatever the fields hold.
  assert.deepEqual(checkPhase(phase({ activity: "rest", minMinutes: "", maxMinutes: "" }), 1), {
    ok: true,
    phase: { name: "Schritt führen", days: 9, activity: "rest", min_minutes: 0, max_minutes: 0, conditions: "nur Boden fest" },
  });
});

const draft = (over: Partial<PlanDraft> = {}): PlanDraft => ({
  diagnosis: " Sehnenzerrung ",
  vet: "Dr. Berger",
  startDate: "2026-03-17",
  checkupDate: "2026-04-10",
  abortCriteria: "Lahmheit",
  phases: [phase()],
  ...over,
});

test("draftToInput builds the API body and rejects bad forms", () => {
  const r = draftToInput(draft());
  assert.equal(r.ok, true);
  if (r.ok) {
    assert.equal(r.input.diagnosis, "Sehnenzerrung");
    assert.equal(r.input.start_date, "2026-03-17");
    assert.equal(r.input.checkup_date, "2026-04-10");
    assert.equal(r.input.phases.length, 1);
  }
  const bad = (over: Partial<PlanDraft>) => {
    const x = draftToInput(draft(over));
    assert.equal(x.ok, false);
    return x.ok ? "" : x.error;
  };
  assert.match(bad({ diagnosis: "" }), /Diagnose/);
  assert.match(bad({ startDate: "17.03.2026" }), /Startdatum/);
  assert.match(bad({ startDate: "2026-02-30" }), /Startdatum/);
  assert.match(bad({ checkupDate: "bald" }), /Kontrolldatum/);
  assert.match(bad({ checkupDate: "2026-03-01" }), /vor dem Start/);
  assert.match(bad({ phases: [] }), /mindestens eine Phase/);
  assert.match(bad({ phases: [phase(), phase({ name: "" })] }), /^Phase 2:/);
  assert.match(bad({ abortCriteria: "x".repeat(1001) }), /1000 Zeichen/);
  assert.match(bad({ phases: [phase({ days: "365" }), phase({ days: "365" }), phase({ days: "1" })] }), /730 Tage/);
  assert.match(bad({ phases: Array.from({ length: 21 }, () => phase()) }), /Höchstens 20 Phasen/);
  // The checkup is optional.
  const noCheckup = draftToInput(draft({ checkupDate: "" }));
  assert.equal(noCheckup.ok && noCheckup.input.checkup_date, "");
});

test("draftFromPlan round-trips through draftToInput", () => {
  const plan: PlanLike = {
    diagnosis: "Sehnenzerrung",
    vet: "",
    start_date: "2026-03-17",
    checkup_date: null,
    abort_criteria: "",
    phases: [
      { name: "Boxenruhe", days: 5, activity: "rest", min_minutes: 0, max_minutes: 0, conditions: "" },
      { name: "Schritt", days: 9, activity: "walker", min_minutes: 10, max_minutes: 20, conditions: "eben" },
    ],
  };
  const d = draftFromPlan(plan);
  assert.equal(d.phases[0]?.minMinutes, "");
  assert.equal(d.checkupDate, "");
  const r = draftToInput(d);
  assert.equal(r.ok, true);
  if (r.ok) assert.deepEqual(r.input.phases, plan.phases);
});

test("list helpers reorder without mutating", () => {
  const list = ["a", "b", "c"];
  assert.deepEqual(movePhase(list, 1, -1), ["b", "a", "c"]);
  assert.deepEqual(movePhase(list, 1, 1), ["a", "c", "b"]);
  assert.deepEqual(movePhase(list, 0, -1), ["a", "b", "c"]);
  assert.deepEqual(movePhase(list, 2, 1), ["a", "b", "c"]);
  assert.deepEqual(removePhase(list, 1), ["a", "c"]);
  assert.deepEqual(putPhase(list, 1, "x"), ["a", "x", "c"]);
  assert.deepEqual(putPhase(list, -1, "x"), ["a", "b", "c", "x"]);
  assert.deepEqual(list, ["a", "b", "c"]);
});

test("new phases start from sensible defaults", () => {
  assert.deepEqual(newPhaseDraft(), { name: "", activity: "walker", days: "7", minMinutes: "10", maxMinutes: "20", conditions: "" });
  const next = newPhaseDraft(phase({ activity: "lunge", maxMinutes: "25" }));
  assert.equal(next.activity, "lunge");
  assert.equal(next.minMinutes, "25");
  assert.equal(newPhaseDraft(phase({ activity: "rest" })).activity, "walker");
  assert.equal(emptyPlanDraft("2026-03-25").startDate, "2026-03-25");
});
