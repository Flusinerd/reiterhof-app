import assert from "node:assert/strict";
import { test } from "node:test";

import {
  ACTIVITIES,
  QUICK_DURATIONS,
  activityIconKey,
  activityLabel,
  addDays,
  dayStatusLabel,
  dotTone,
  dotViewModels,
  formatClock,
  formatDayLong,
  gaitShareRows,
  isValidDate,
  parseFinishParams,
  reinSummary,
  secondsToMinutes,
  segmentViewModels,
  trainedCount,
  trainingErrorText,
  weekRangeLabel,
  weekdayIndex,
  weekdayShort,
} from "./training.ts";

test("activity labels and icons match the backend", () => {
  assert.equal(ACTIVITIES.length, 7);
  assert.equal(activityLabel("hall"), "Halle");
  assert.equal(activityLabel("walker"), "Führanlage");
  assert.equal(activityLabel("rest"), "Ruhetag");
  assert.equal(activityLabel("unknown"), "unknown");
  for (const a of ACTIVITIES) assert.notEqual(activityLabel(a), a);
  assert.equal(activityIconKey("jumping"), "trending-up");
  assert.equal(activityIconKey("rest"), "moon");
  assert.equal(activityIconKey("?"), "route");
});

test("quick-log duration chips", () => {
  assert.deepEqual([...QUICK_DURATIONS], [20, 30, 45, 60]);
});

test("clock formatting and minutes from seconds", () => {
  assert.equal(formatClock(0), "00:00");
  assert.equal(formatClock(75), "01:15");
  assert.equal(formatClock(3725), "1:02:05");
  assert.equal(formatClock(-5), "00:00");
  assert.equal(secondsToMinutes(10), 1);
  assert.equal(secondsToMinutes(89), 1);
  assert.equal(secondsToMinutes(91), 2);
  assert.equal(secondsToMinutes(2700), 45);
});

test("date helpers work on calendar dates", () => {
  assert.equal(weekdayIndex("2026-03-23"), 0); // Monday
  assert.equal(weekdayShort("2026-03-29"), "So");
  assert.equal(formatDayLong("2026-03-25"), "Mittwoch, 25. März");
  assert.equal(addDays("2026-03-23", 7), "2026-03-30");
  assert.equal(addDays("2026-03-01", -1), "2026-02-28");
  assert.equal(addDays("2026-03-28", 2), "2026-03-30"); // across the DST change
  assert.equal(weekRangeLabel("2026-03-23", "2026-03-29"), "23.–29. März");
  assert.equal(weekRangeLabel("2026-03-30", "2026-04-05"), "30. März – 5. April");
  assert.ok(isValidDate("2026-02-28"));
  assert.ok(!isValidDate("2026-02-30"));
  assert.ok(!isValidDate("28.02.2026"));
});

test("dot view models: trained, rest, nothing", () => {
  assert.equal(dotTone("nothing", "none"), "empty");
  assert.equal(dotTone("rest", "none"), "rest");
  assert.equal(dotTone("trained", "light"), "light");
  assert.equal(dotTone("trained", "intense"), "intense");
  assert.equal(dotTone("trained", "none"), "light"); // trained without a load still counts as a trained day

  const dots = [
    { date: "2026-03-23", kind: "trained", intensity: "medium", is_today: false },
    { date: "2026-03-24", kind: "rest", intensity: "none", is_today: false },
    { date: "2026-03-25", kind: "nothing", intensity: "none", is_today: true },
  ] as const;
  const vm = dotViewModels(dots);
  assert.deepEqual(
    vm.map((d) => [d.label, d.tone, d.isToday]),
    [
      ["Mo", "medium", false],
      ["Di", "rest", false],
      ["Mi", "empty", true],
    ],
  );
  assert.equal(vm[0]?.accessibilityLabel, "Montag, 23. März: trainiert, mittel");
  assert.equal(vm[1]?.accessibilityLabel, "Dienstag, 24. März: Ruhetag");
  assert.equal(vm[2]?.accessibilityLabel, "Heute: nichts eingetragen");
  assert.equal(trainedCount(dots), 1);
});

test("segment view models grow with the level", () => {
  const vm = segmentViewModels([
    { date: "2026-03-23", load: 0, level: "none", sessions: 0 },
    { date: "2026-03-24", load: 20, level: "light", sessions: 1 },
    { date: "2026-03-25", load: 40, level: "medium", sessions: 1 },
    { date: "2026-03-26", load: 80, level: "intense", sessions: 2 },
  ]);
  assert.deepEqual(
    vm.map((s) => s.label),
    ["Mo", "Di", "Mi", "Do"],
  );
  const r = vm.map((s) => s.ratio);
  assert.ok(r[0]! < r[1]! && r[1]! < r[2]! && r[2]! < r[3]!);
  assert.equal(r[3], 1);
  assert.equal(vm[0]?.accessibilityLabel, "Mo: keine Belastung");
  assert.equal(vm[3]?.accessibilityLabel, "Do: Belastung intensiv");
});

test("week row status labels", () => {
  const jan = { id: "1", name: "Jan" };
  assert.equal(dayStatusLabel("open", null, false), "Niemand eingetragen");
  assert.equal(dayStatusLabel("planned", jan, true), "Ich");
  assert.equal(dayStatusLabel("planned", jan, false), "Jan");
  assert.equal(dayStatusLabel("done", jan, false), "Jan, erledigt");
  assert.equal(dayStatusLabel("done", null, false), "Erledigt");
  assert.equal(dayStatusLabel("today", null, false), "Heute noch offen");
  assert.equal(dayStatusLabel("today", jan, true), "Heute: Ich");
  assert.equal(dayStatusLabel("rest", null, false, "after_show"), "Ruhetag nach dem Turnier");
  assert.equal(dayStatusLabel("rest", null, false, "planned"), "Ruhetag");
  assert.equal(dayStatusLabel("empty", null, false), "Kein Training");
});

test("gait shares and rein summary", () => {
  assert.deepEqual(gaitShareRows({ walk: 0.3, trot: 0.5, canter: 0.2, halt: 0 }), [
    { gait: "trot", label: "Trab", percent: 50 },
    { gait: "walk", label: "Schritt", percent: 30 },
    { gait: "canter", label: "Galopp", percent: 20 },
  ]);
  assert.deepEqual(gaitShareRows({}), []);
  assert.deepEqual(gaitShareRows(null), []);
  assert.deepEqual(gaitShareRows({ canter: 0.2 }), [{ gait: "canter", label: "Galopp", percent: 20 }]);

  assert.equal(reinSummary([]), null);
  assert.equal(reinSummary(undefined), null);
  assert.deepEqual(
    reinSummary([
      { rein: "left", minutes: 20 },
      { rein: "right", minutes: 20 },
      { rein: "left", minutes: 10 },
    ]),
    { leftPercent: 60, rightPercent: 40, changes: 2 },
  );
  assert.deepEqual(reinSummary([{ rein: "right", minutes: 0 }]), { leftPercent: 0, rightPercent: 0, changes: 0 });
});

test("finish params are parsed defensively", () => {
  assert.equal(parseFinishParams({}), null);
  assert.equal(parseFinishParams({ horse: "h1", activity: "rest" }), null);
  assert.equal(parseFinishParams({ activity: "hall" }), null);

  const p = parseFinishParams({
    horse: "h1",
    activity: "hall",
    minutes: "45.4",
    exercise: "e1",
    gait: '{"walk":0.5,"trot":0.5}',
    rein: "kaputt",
  });
  assert.deepEqual(p, {
    horse: "h1",
    activity: "hall",
    minutes: 45,
    startedAt: undefined,
    exerciseId: "e1",
    gaitShares: { walk: 0.5, trot: 0.5 },
    reinChanges: undefined,
    distanceM: undefined,
  });
  assert.equal(parseFinishParams({ horse: "h", activity: "hack", minutes: "0" })?.minutes, 1);
  assert.equal(parseFinishParams({ horse: "h", activity: "hack", minutes: "9999" })?.minutes, 600);
  assert.equal(parseFinishParams({ horse: ["h", "x"], activity: ["hack"], minutes: ["30"] })?.minutes, 30);
});

test("training error texts are German", () => {
  assert.match(trainingErrorText("forbidden") ?? "", /Berechtigung/);
  assert.match(trainingErrorText("conflict") ?? "", /vergeben/);
  assert.equal(trainingErrorText("internal"), null);
});
