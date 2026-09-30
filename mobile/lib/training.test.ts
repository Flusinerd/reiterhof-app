import assert from "node:assert/strict";
import { test } from "node:test";

import {
  ACTIVITIES,
  MINUTE_CHOICES,
  QUICK_DURATIONS,
  activityIconKey,
  activityLabel,
  addDays,
  dayStatusLabel,
  dotTone,
  dotViewModels,
  exerciseLibrary,
  formatClock,
  formatDayLong,
  gaitShareRows,
  isValidDate,
  minuteChoices,
  parseFinishParams,
  reinSummary,
  secondsToMinutes,
  segmentViewModels,
  trainedCount,
  trainingErrorText,
  weekRangeLabel,
  weekStripItems,
  weekdayIndex,
  weekdayShort,
  type WeekDayLike,
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

// --- day editing and the week strip ----------------------------------------------------

test("minute choices per activity", () => {
  assert.deepEqual(MINUTE_CHOICES.hall, [30, 45, 60]);
  assert.deepEqual(MINUTE_CHOICES.arena, [30, 45, 60]);
  assert.deepEqual(MINUTE_CHOICES.hack, [30, 45, 60, 90, 120]);
  assert.deepEqual(MINUTE_CHOICES.lunge, [15, 20, 25, 30]);
  assert.deepEqual(MINUTE_CHOICES.jumping, [30, 40, 45]);
  assert.deepEqual(MINUTE_CHOICES.groundwork, [15, 20, 30, 45]);
  assert.deepEqual(MINUTE_CHOICES.walker, [20, 30, 45, 60]);
  for (const a of ACTIVITIES) assert.ok(MINUTE_CHOICES[a].length > 0);
});

test("minuteChoices adds the current value, sorted and unique", () => {
  assert.deepEqual(minuteChoices("hall"), [30, 45, 60]);
  assert.deepEqual(minuteChoices("hall", 45), [30, 45, 60]);
  assert.deepEqual(minuteChoices("hall", 50), [30, 45, 50, 60]);
  assert.deepEqual(minuteChoices("hall", 10), [10, 30, 45, 60]);
  assert.deepEqual(minuteChoices("hack", 150), [30, 45, 60, 90, 120, 150]);
  assert.deepEqual(minuteChoices("lunge", 0), [15, 20, 25, 30]);
  assert.deepEqual(minuteChoices("lunge", -5), [15, 20, 25, 30]);
  // The constant is not modified.
  minuteChoices("hall", 5).push(1);
  assert.deepEqual(MINUTE_CHOICES.hall, [30, 45, 60]);
});

test("exerciseLibrary mirrors the backend", () => {
  assert.equal(exerciseLibrary("jumping", "dressage"), "jumping");
  assert.equal(exerciseLibrary("groundwork", "dressage"), "groundwork");
  assert.equal(exerciseLibrary("lunge", "jumping"), "lunge");
  assert.equal(exerciseLibrary("hall", "dressage"), "dressage");
  assert.equal(exerciseLibrary("hall", "jumping"), "jumping");
  assert.equal(exerciseLibrary("arena", "jumping"), "jumping");
  assert.equal(exerciseLibrary("arena", "western"), "dressage");
  assert.equal(exerciseLibrary("hall", ""), "dressage");
  assert.equal(exerciseLibrary("hack", "dressage"), "");
  assert.equal(exerciseLibrary("walker", "jumping"), "");
  assert.equal(exerciseLibrary("rest", "jumping"), "");
});

test("week strip items", () => {
  // 2026-09-28 is a Monday; Wednesday is today.
  const days: WeekDayLike[] = [
    { date: "2026-09-28", status: "done", is_today: false, activity: "hack" },
    { date: "2026-09-29", status: "empty", is_today: false, activity: null },
    { date: "2026-09-30", status: "today", is_today: true, activity: null },
    { date: "2026-10-01", status: "planned", is_today: false, activity: "hall" },
    { date: "2026-10-02", status: "rest", is_today: false, activity: null },
    { date: "2026-10-03", status: "open", is_today: false },
    { date: "2026-10-04", status: "planned", is_today: false, activity: "" },
  ];
  const items = weekStripItems(days);
  assert.deepEqual(
    items.map((i) => i.label),
    ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"],
  );
  assert.deepEqual(
    items.map((i) => i.tone),
    ["done", "empty", "open", "planned", "rest", "open", "open"],
  );
  assert.deepEqual(
    items.map((i) => i.iconKey),
    ["check", null, null, "hall", null, null, null],
  );
  assert.deepEqual(
    items.map((i) => i.isToday),
    [false, false, true, false, false, false, false],
  );
  assert.equal(items[0]?.key, "2026-09-28");
  assert.equal(items[0]?.accessibilityLabel, "Montag: erledigt, Ausritt");
  assert.equal(items[1]?.accessibilityLabel, "Dienstag: kein Training");
  assert.equal(items[2]?.accessibilityLabel, "Heute: offen");
  assert.equal(items[3]?.accessibilityLabel, "Donnerstag: geplant, Halle");
  assert.equal(items[4]?.accessibilityLabel, "Freitag: Ruhetag");
  assert.equal(items[5]?.accessibilityLabel, "Samstag: offen");
});

test("week strip: today with an activity is planned, a done day without one still shows the check", () => {
  const [today, done] = weekStripItems([
    { date: "2026-09-30", status: "today", is_today: true, activity: "lunge" },
    { date: "2026-09-29", status: "done", is_today: false },
  ]);
  assert.equal(today?.tone, "planned");
  assert.equal(today?.iconKey, "lunge");
  assert.equal(today?.accessibilityLabel, "Heute: geplant, Longe");
  assert.equal(done?.tone, "done");
  assert.equal(done?.iconKey, "check");
  assert.equal(done?.accessibilityLabel, "Dienstag: erledigt");
  assert.deepEqual(weekStripItems([]), []);
});
