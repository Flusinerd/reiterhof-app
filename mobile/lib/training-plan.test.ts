import assert from "node:assert/strict";
import { test } from "node:test";

import type { WeekDay } from "./api/training.ts";
import type { WeekDayLike } from "./training.ts";
import {
  MAX_NOTE,
  applicableDays,
  canAskForAI,
  dayEditBody,
  dayEditFrom,
  editedPlanDay,
  nextStep,
  openPlanDays,
  planApplyBody,
  planDayNote,
  planStatusText,
  replaceDraftDay,
  reproposeOptions,
  type DayEdit,
  type PlanDay,
  type PlanResponse,
} from "./training-plan.ts";

const day = (over: Partial<PlanDay> = {}): PlanDay => ({
  date: "2026-10-01",
  weekday: 3,
  activity: "hall",
  label: "Halle",
  minutes: 45,
  intensity: "medium",
  intensity_label: "mittel",
  reason: "Nach dem Ausritt gestern passt die Halle.",
  source: "ai",
  user: null,
  exercise: null,
  ...over,
});

test("the note labels AI proposals", () => {
  assert.equal(planDayNote(day()), "KI-Vorschlag: 45 Min. · Nach dem Ausritt gestern passt die Halle.");
  assert.equal(planDayNote(day({ source: "rules", activity: "rest", minutes: 0, reason: "" })), "Plan: Ruhetag");
  const long = planDayNote(day({ reason: "x".repeat(600) }));
  assert.equal(long.length, MAX_NOTE);
  assert.ok(long.endsWith("…"));
});

test("applying keeps a claimed person and plans rest days without one", () => {
  assert.deepEqual(planApplyBody(day()), {
    status: "planned",
    activity: "hall",
    user_id: "",
    note: "KI-Vorschlag: 45 Min. · Nach dem Ausritt gestern passt die Halle.",
  });
  assert.equal(planApplyBody(day({ user: { id: "u1", name: "Mia" } }))?.user_id, "u1");
  const rest = planApplyBody(day({ activity: "rest", minutes: 0, source: "rules", reason: "Ruhetag nach dem Turnier." }));
  assert.deepEqual(rest, { status: "rest", note: "Plan: Ruhetag · Ruhetag nach dem Turnier." });
  const withContent = planApplyBody(day({ focus: "Übergänge Schritt-Trab", exercise: { id: "ex1", title: "Übergänge" } }));
  assert.equal(withContent?.focus, "Übergänge Schritt-Trab");
  assert.equal(withContent?.exercise_id, "ex1");
  assert.equal("focus" in planApplyBody(day())!, false);
  // A rest day must not remove somebody's claim.
  assert.equal(planApplyBody(day({ activity: "rest", minutes: 0, user: { id: "u1", name: "Mia" } })), null);
});

test("status hints and the consent offer", () => {
  assert.equal(planStatusText("used", true), null);
  assert.match(planStatusText("no_consent", true)!, /Mit deiner Erlaubnis/);
  assert.match(planStatusText("no_consent", false)!, /nur der Besitzer/);
  for (const s of ["not_configured", "owner_under_16", "limit", "failed"] as const) assert.match(planStatusText(s, true)!, /festen Regeln/);
  assert.match(planStatusText("nothing_to_plan", true)!, /kein Tag/);

  const plan = (over: Partial<PlanResponse>): PlanResponse => ({
    horse_id: "h",
    start: "2026-09-28",
    end: "2026-10-04",
    source: "rules",
    ai_status: "no_consent",
    owner_is_me: true,
    days: [],
    ...over,
  });
  assert.equal(canAskForAI(plan({})), true);
  assert.equal(canAskForAI(plan({ owner_is_me: false })), false);
  assert.equal(canAskForAI(plan({ ai_status: "failed" })), false);
  assert.equal(canAskForAI(plan({ ai_status: "owner_under_16" })), false);
});

// --- next step -------------------------------------------------------------------------

// Week of 2026-09-28 (Monday); Wednesday the 30th is today.
const weekDay = (date: string, status: string, over: Partial<WeekDayLike> = {}): WeekDayLike => ({
  date,
  status,
  is_today: false,
  activity: null,
  ...over,
});

const week = (...over: [string, Partial<WeekDayLike>][]): WeekDayLike[] => {
  const base: WeekDayLike[] = [
    weekDay("2026-09-28", "done", { activity: "hack" }),
    weekDay("2026-09-29", "empty"),
    weekDay("2026-09-30", "today", { is_today: true }),
    weekDay("2026-10-01", "open"),
    weekDay("2026-10-02", "open"),
    weekDay("2026-10-03", "planned", { activity: "hall" }),
    weekDay("2026-10-04", "rest"),
  ];
  return base.map((d) => {
    const patch = over.find(([date]) => date === d.date);
    return patch ? { ...d, ...patch[1] } : d;
  });
};

test("openPlanDays: open, or today/planned without an activity; never done, rest, empty or past", () => {
  assert.deepEqual(
    openPlanDays(week()).map((d) => d.date),
    ["2026-09-30", "2026-10-01", "2026-10-02"],
  );
  // Planned by somebody, but without an activity: open for the planner.
  const claimed = week(["2026-10-03", { activity: null }]);
  assert.deepEqual(
    openPlanDays(claimed).map((d) => d.date),
    ["2026-09-30", "2026-10-01", "2026-10-02", "2026-10-03"],
  );
  // Today with an activity is taken.
  assert.deepEqual(
    openPlanDays(week(["2026-09-30", { activity: "hall" }])).map((d) => d.date),
    ["2026-10-01", "2026-10-02"],
  );
  // "" counts as no activity.
  assert.equal(openPlanDays(week(["2026-10-03", { activity: "" }])).length, 4);
  // A planned day without activity before today is past.
  assert.equal(openPlanDays(week(["2026-09-29", { status: "planned" }])).length, 3);
  for (const status of ["done", "rest", "empty"]) {
    assert.deepEqual(openPlanDays([weekDay("2026-10-01", status)]), []);
  }
  assert.deepEqual(openPlanDays([]), []);
});

test("openPlanDays keeps the element type", () => {
  const rows = [{ date: "2026-10-01", status: "open", is_today: false, id: 7 }];
  assert.equal(openPlanDays(rows)[0]?.id, 7);
});

test("nextStep", () => {
  const owner = { has_profile: true, can_edit: true };
  assert.equal(nextStep(undefined, { days: week() }), null);
  assert.equal(nextStep(undefined, undefined), null);
  assert.deepEqual(nextStep({ has_profile: false, can_edit: true }, undefined), { kind: "profile" });
  assert.deepEqual(nextStep({ has_profile: false, can_edit: true }, { days: week() }), { kind: "profile" });
  assert.equal(nextStep({ has_profile: false, can_edit: false }, { days: week() }), null);
  assert.equal(nextStep({ has_profile: true, can_edit: false }, { days: week() }), null);
  assert.equal(nextStep(owner, undefined), null);
  // Counts all open days, today included.
  assert.deepEqual(nextStep(owner, { days: week() }), { kind: "plan", openDays: 3 });
  assert.deepEqual(nextStep(owner, { days: week(["2026-10-02", { status: "rest" }]) }), { kind: "plan", openDays: 2 });
  // Only today is open: nothing to plan ahead.
  const onlyToday = week(["2026-10-01", { status: "rest" }], ["2026-10-02", { status: "rest" }]);
  assert.equal(nextStep(owner, { days: onlyToday }), null);
  // Nothing open at all.
  const none = week(["2026-09-30", { activity: "hall" }], ["2026-10-01", { status: "rest" }], ["2026-10-02", { status: "rest" }]);
  assert.equal(nextStep(owner, { days: none }), null);
  assert.equal(nextStep(owner, { days: [] }), null);
});

// --- editing days ----------------------------------------------------------------------

const planned: DayEdit = {
  status: "planned",
  activity: "hall",
  minutes: 45,
  focus: "Übergänge",
  exercise: { id: "ex1", title: "Übergänge" },
  userId: null,
};

test("editedPlanDay replaces the unit and drops the model's level and reason context", () => {
  const before = day({ level: "normal", level_label: "normal", focus: "Alt", exercise: { id: "old", title: "Alt" }, user: { id: "u1", name: "Mia" } });
  const after = editedPlanDay(before, { ...planned, activity: "hack", minutes: 90, focus: "", exercise: null });
  assert.equal(after.activity, "hack");
  assert.equal(after.label, "Ausritt");
  assert.equal(after.minutes, 90);
  assert.equal(after.focus, "");
  assert.equal(after.exercise, null);
  assert.equal(after.edited, true);
  assert.equal("level" in after, false);
  assert.equal("level_label" in after, false);
  // Kept.
  assert.equal(after.intensity, "medium");
  assert.equal(after.intensity_label, "mittel");
  assert.equal(after.reason, before.reason);
  assert.equal(after.source, "ai");
  assert.deepEqual(after.user, { id: "u1", name: "Mia" });
  assert.equal(after.date, before.date);
  // The input stays untouched.
  assert.equal(before.activity, "hall");
  assert.equal(before.edited, undefined);
  assert.equal(before.level, "normal");
});

test("editedPlanDay: a rest day has no activity content", () => {
  const after = editedPlanDay(day({ focus: "Alt", level: "light", level_label: "leicht" }), { ...planned, status: "rest", activity: null });
  assert.equal(after.activity, "rest");
  assert.equal(after.label, "Ruhetag");
  assert.equal(after.minutes, 0);
  assert.equal(after.focus, "");
  assert.equal(after.exercise, null);
  assert.equal(after.edited, true);
  assert.equal("level" in after, false);
});

test("an edited day notes the plan without the model's reason", () => {
  assert.equal(planDayNote(editedPlanDay(day(), planned)), "Plan: 45 Min.");
  assert.equal(planDayNote(editedPlanDay(day(), { ...planned, status: "rest", activity: null })), "Plan: Ruhetag");
  // Unedited days keep their note.
  assert.equal(planDayNote(day()), "KI-Vorschlag: 45 Min. · Nach dem Ausritt gestern passt die Halle.");
});

test("applying an edited day stores its note and lets focus and exercise be cleared", () => {
  const cleared = editedPlanDay(day({ focus: "Alt", exercise: { id: "old", title: "Alt" } }), { ...planned, focus: "", exercise: null });
  assert.deepEqual(planApplyBody(cleared), { status: "planned", activity: "hall", user_id: "", note: "Plan: 45 Min.", focus: "", exercise_id: "" });
  const withContent = planApplyBody(editedPlanDay(day(), planned));
  assert.equal(withContent?.focus, "Übergänge");
  assert.equal(withContent?.exercise_id, "ex1");
  assert.deepEqual(planApplyBody(editedPlanDay(day(), { ...planned, status: "rest", activity: null })), { status: "rest", note: "Plan: Ruhetag" });
});

test("applicableDays counts the days that would be stored", () => {
  const mia = { id: "u1", name: "Mia" };
  const days = [
    day({ date: "2026-10-01" }),
    day({ date: "2026-10-02", activity: "rest", minutes: 0 }),
    day({ date: "2026-10-03", activity: "rest", minutes: 0, user: mia }),
    day({ date: "2026-10-04", user: mia }),
  ];
  assert.equal(applicableDays(days), 3);
  assert.equal(applicableDays([]), 0);
  assert.equal(applicableDays([days[2]!]), 0);
});

test("replaceDraftDay replaces by date and keeps the order", () => {
  const days = [day({ date: "2026-10-01" }), day({ date: "2026-10-02" }), day({ date: "2026-10-03" })];
  const fresh = day({ date: "2026-10-02", activity: "hack", minutes: 60 });
  const next = replaceDraftDay(days, fresh);
  assert.deepEqual(
    next.map((d) => d.date),
    ["2026-10-01", "2026-10-02", "2026-10-03"],
  );
  assert.equal(next[1], fresh);
  assert.equal(next[0], days[0]);
  assert.equal(days[1]?.activity, "hall");
  // A date that is not in the draft adds nothing.
  assert.equal(replaceDraftDay(days, day({ date: "2026-10-09" })).length, 3);
});

test("dayEditBody always carries user_id; rest has no activity", () => {
  assert.deepEqual(dayEditBody(planned), {
    status: "planned",
    activity: "hall",
    user_id: "",
    note: "Plan: 45 Min.",
    focus: "Übergänge",
    exercise_id: "ex1",
  });
  const cleared = dayEditBody({ ...planned, focus: "", exercise: null, userId: "u1" });
  assert.equal(cleared.user_id, "u1");
  assert.equal(cleared.focus, "");
  assert.equal(cleared.exercise_id, "");
  assert.equal("user_id" in dayEditBody({ ...planned, userId: null }), true);
  assert.equal(dayEditBody({ ...planned, userId: null }).user_id, "");
  const rest = dayEditBody({ ...planned, status: "rest", activity: null, userId: "u1" });
  assert.deepEqual(rest, { status: "rest", note: "Plan: Ruhetag" });
  assert.equal("activity" in rest, false);
});

test("dayEditFrom a saved week day", () => {
  const weekDayRow = (over: Partial<WeekDay>): WeekDay => ({
    date: "2026-10-01",
    weekday: 3,
    status: "planned",
    is_today: false,
    user: { id: "u1", name: "Mia" },
    is_me: false,
    activity: "hack",
    minutes: 60,
    note: null,
    show: null,
    can_take: false,
    focus: null,
    exercise: null,
    reha: null,
    ...over,
  });
  assert.deepEqual(dayEditFrom(weekDayRow({})), {
    status: "planned",
    activity: "hack",
    minutes: 60,
    focus: "",
    exercise: null,
    userId: "u1",
  });
  const withContent = dayEditFrom(weekDayRow({ focus: "Gelände", exercise: { id: "ex1", title: "T" }, user: null }));
  assert.equal(withContent.focus, "Gelände");
  assert.deepEqual(withContent.exercise, { id: "ex1", title: "T" });
  assert.equal(withContent.userId, null);
  const rest = dayEditFrom(weekDayRow({ status: "rest", activity: null, minutes: 0, user: null }));
  assert.equal(rest.status, "rest");
  assert.equal(rest.activity, null);
  assert.equal(dayEditFrom(weekDayRow({ status: "open", activity: null, user: null })).status, "planned");
});

test("dayEditFrom a draft day", () => {
  assert.deepEqual(dayEditFrom(day({ focus: "Übergänge", exercise: { id: "ex1", title: "T" }, user: { id: "u1", name: "Mia" } })), {
    status: "planned",
    activity: "hall",
    minutes: 45,
    focus: "Übergänge",
    exercise: { id: "ex1", title: "T" },
    userId: "u1",
  });
  const rest = dayEditFrom(day({ activity: "rest", minutes: 0 }));
  assert.equal(rest.status, "rest");
  assert.equal(rest.activity, null);
  assert.equal(rest.focus, "");
  assert.equal(rest.userId, null);
});

test("reproposeOptions: the other draft days are context, the current activity is excluded", () => {
  const days = [
    day({ date: "2026-10-01", activity: "hall", minutes: 45 }),
    day({ date: "2026-10-02", activity: "rest", minutes: 0 }),
    day({ date: "2026-10-03", activity: "hack", minutes: 60 }),
  ];
  assert.deepEqual(reproposeOptions(days, "2026-10-03"), {
    day: "2026-10-03",
    draft: [
      { date: "2026-10-01", activity: "hall", minutes: 45 },
      { date: "2026-10-02", activity: "rest", minutes: 0 },
    ],
    exclude: ["hack"],
  });
  // A rest day excludes nothing; the rest days of the context carry no minutes.
  const forRest = reproposeOptions(days, "2026-10-02");
  assert.deepEqual(forRest.exclude, []);
  assert.deepEqual(forRest.draft.map((d) => d.date), ["2026-10-01", "2026-10-03"]);
  // A day that is not in the draft: everything is context.
  const unknown = reproposeOptions(days, "2026-10-04");
  assert.equal(unknown.draft.length, 3);
  assert.deepEqual(unknown.exclude, []);
  assert.equal(reproposeOptions([], "2026-10-01").draft.length, 0);
  assert.equal(reproposeOptions([day({ activity: "rest", minutes: 7 }), day({ date: "2026-10-02" })], "2026-10-02").draft[0]?.minutes, 0);
});
