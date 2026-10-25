import assert from "node:assert/strict";
import { test } from "node:test";

import type { WeekDay } from "./api/training.ts";
import type { Unit } from "./training-focus.ts";
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
  pickActivity,
  pickCustomFocus,
  pickFocusKey,
  pickMinutes,
  pickNoFocus,
  pickRest,
  pickUnitExercise,
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

// --- focus key, unit plan and equipment of a day (newer backends) ------------------------

const unit: Unit = {
  warmup: { exercise: { id: "w1", title: "Cavaletti auf dem Zirkel" }, minutes: 15 },
  main: { exercise: { id: "m1", title: "Schulterherein" }, minutes: 22 },
  cooldown: { exercise: null, minutes: 8 },
};

test("applying a proposal sends the focus key and the unit when it has them", () => {
  const body = planApplyBody(day({ focus_key: "takt", focus_label: "Takt", unit }))!;
  assert.equal(body.focus_key, "takt");
  assert.deepEqual(body.unit, {
    warmup: { exercise_id: "w1", minutes: 15 },
    main: { exercise_id: "m1", minutes: 22 },
    cooldown: { exercise_id: null, minutes: 8 },
  });
  // An older backend sends neither: the body stays as it was (it would reject unknown fields).
  const old = planApplyBody(day())!;
  assert.equal("focus_key" in old, false);
  assert.equal("unit" in old, false);
  assert.equal("focus_key" in planApplyBody(day({ focus_key: null, unit: null }))!, false);
  assert.equal("focus_key" in planApplyBody(day({ focus_key: "" }))!, false);
});

test("an edited proposal states a removed focus key, an untouched one states nothing", () => {
  const before = day({ focus_key: "takt", focus_label: "Takt", unit });
  const same = editedPlanDay(before, { ...planned, focusKey: "takt", unit });
  assert.equal(same.focus_key, "takt");
  assert.equal(same.focus_label, "Takt", "the label stays with its key");
  assert.deepEqual(same.unit, unit);
  const other = editedPlanDay(before, { ...planned, focusKey: "anlehnung" });
  assert.equal(other.focus_key, "anlehnung");
  assert.equal(other.focus_label, undefined, "a new key brings its own label from the catalog");
  assert.equal(other.unit, undefined);
  const removed = editedPlanDay(before, { ...planned, focusKey: null });
  assert.equal(removed.focus_key, null);
  assert.equal(planApplyBody(removed)?.focus_key, "", "removing is stored as the empty key");
  // Not touched on a backend without keys: nothing is sent.
  const untouched = editedPlanDay(day(), planned);
  assert.equal(untouched.focus_key, undefined);
  assert.equal("focus_key" in planApplyBody(untouched)!, false);
  assert.equal("unit" in planApplyBody(untouched)!, false);
  // A rest day has no focus and no unit.
  const rest = editedPlanDay(before, { ...planned, status: "rest", activity: null });
  assert.equal(rest.focus_key, null);
  assert.equal(rest.unit, null);
});

test("dayEditBody sends focus key, unit and equipment_ok only when the editor has a value", () => {
  const base = dayEditBody(planned);
  assert.equal("focus_key" in base, false);
  assert.equal("unit" in base, false);
  assert.equal("equipment_ok" in base, false);
  const full = dayEditBody({ ...planned, focusKey: "takt", unit, equipmentOk: false });
  assert.equal(full.focus_key, "takt");
  assert.equal(full.unit?.main?.exercise_id, "m1");
  assert.equal(full.equipment_ok, false);
  assert.equal(dayEditBody({ ...planned, focusKey: null }).focus_key, "", "null removes the key");
  assert.equal(dayEditBody({ ...planned, equipmentOk: null }).equipment_ok, null, "null is the location's default");
  assert.equal(dayEditBody({ ...planned, equipmentOk: true }).equipment_ok, true);
  // A rest day sends none of it.
  const rest = dayEditBody({ ...planned, status: "rest", activity: null, focusKey: "takt", unit, equipmentOk: true });
  assert.deepEqual(rest, { status: "rest", note: "Plan: Ruhetag" });
});

test("dayEditFrom reads focus key, unit and equipment_ok of newer backends", () => {
  const weekDayRow = (over: Partial<WeekDay>): WeekDay => ({
    date: "2026-10-01",
    weekday: 3,
    status: "planned",
    is_today: false,
    user: null,
    is_me: false,
    activity: "hall",
    minutes: 45,
    note: null,
    show: null,
    can_take: false,
    focus: null,
    exercise: null,
    reha: null,
    ...over,
  });
  const from = dayEditFrom(weekDayRow({ focus_key: "takt", focus_label: "Takt", unit, equipment_ok: false }));
  assert.equal(from.focusKey, "takt");
  assert.deepEqual(from.unit, unit);
  assert.equal(from.equipmentOk, false);
  // null and absent both mean "none yet": the keys stay out, so that nothing is sent without a change.
  for (const row of [weekDayRow({ focus_key: null, unit: null }), weekDayRow({})]) {
    const edit = dayEditFrom(row);
    assert.equal("focusKey" in edit, false);
    assert.equal("unit" in edit, false);
    assert.equal("equipmentOk" in edit, false);
  }
  // equipment_ok null is a value of its own (the default), only an absent field is not.
  assert.equal(dayEditFrom(weekDayRow({ equipment_ok: null })).equipmentOk, null);
  // A proposal knows no equipment_ok.
  const fromPlan = dayEditFrom(day({ focus_key: "schwung", unit }));
  assert.equal(fromPlan.focusKey, "schwung");
  assert.deepEqual(fromPlan.unit, unit);
  assert.equal("equipmentOk" in fromPlan, false);
});

// --- what the sheet does to an edit ----------------------------------------------------

const hall: DayEdit = {
  status: "planned",
  activity: "hall",
  minutes: 45,
  focus: "Übergänge",
  exercise: { id: "m1", title: "Schulterherein" },
  userId: null,
  focusKey: "takt",
  unit,
};

test("choosing a focus: a new key takes the model's words with it, the same key keeps them", () => {
  assert.deepEqual(pickFocusKey(hall, "anlehnung").focusKey, "anlehnung");
  assert.equal(pickFocusKey(hall, "anlehnung").focus, "");
  assert.equal(pickFocusKey(hall, "takt").focus, "Übergänge");
  assert.equal(pickFocusKey({ ...hall, focusKey: undefined }, "takt").focusKey, "takt");
  // No focus: a key that was there is removed explicitly, otherwise nothing is sent.
  const none = pickNoFocus(hall, "takt");
  assert.equal(none.focusKey, null);
  assert.equal(none.focus, "");
  assert.equal(pickNoFocus(hall, undefined).focusKey, undefined);
  assert.equal(pickNoFocus(hall, null).focusKey, undefined);
  // Eigener Text: the catalog focus goes, the text stays.
  const custom = pickCustomFocus(hall, "takt");
  assert.equal(custom.focusKey, null);
  assert.equal(custom.focus, "Übergänge");
  assert.equal(pickCustomFocus({ ...hall, focusKey: undefined }, undefined).focusKey, undefined);
  assert.equal(dayEditBody(custom).focus_key, "");
  assert.equal(dayEditBody(custom).focus, "Übergänge");
  assert.equal("focus_key" in dayEditBody(pickNoFocus({ ...hall, focusKey: undefined }, undefined)), false);
});

test("changing the minutes shares them among the parts anew and keeps the exercises", () => {
  const longer = pickMinutes(hall, 60);
  assert.equal(longer.minutes, 60);
  assert.deepEqual(longer.unit?.warmup, { exercise: { id: "w1", title: "Cavaletti auf dem Zirkel" }, minutes: 20 });
  assert.deepEqual(longer.unit?.main, { exercise: { id: "m1", title: "Schulterherein" }, minutes: 30 });
  assert.deepEqual(longer.unit?.cooldown, { exercise: null, minutes: 10 });
  assert.deepEqual(longer.exercise, { id: "m1", title: "Schulterherein" });
  // Without a unit only the minutes change.
  const plain = pickMinutes({ ...hall, unit: undefined }, 60);
  assert.equal(plain.minutes, 60);
  assert.equal(plain.unit, undefined);
  assert.deepEqual(plain.exercise, hall.exercise);
});

test("changing the activity keeps what still fits and re-splits the unit", () => {
  // Hall to arena: same libraries, so the exercises stay; 45 is offered there too.
  const arena = pickActivity(hall, "arena", "dressage");
  assert.equal(arena.activity, "arena");
  assert.equal(arena.minutes, 45);
  assert.deepEqual(arena.exercise, { id: "m1", title: "Schulterherein" });
  assert.equal(arena.unit?.main?.exercise?.id, "m1");
  // Hall to lunge: other libraries, the exercises go; 45 is no lunge duration, so the default is taken.
  const lunge = pickActivity(hall, "lunge", "dressage");
  assert.equal(lunge.minutes, 25);
  assert.equal(lunge.exercise, null);
  assert.deepEqual(lunge.unit, {
    warmup: { exercise: null, minutes: 10 },
    main: { exercise: null, minutes: 10 },
    cooldown: { exercise: null, minutes: 5 },
  });
  // A jumping horse: the hall uses jumping, cavaletti and dressage, the jumping place only the first two.
  assert.equal(pickActivity(hall, "jumping", "jumping").exercise, null);
  assert.equal(pickActivity({ ...hall, activity: "arena" }, "hall", "jumping").exercise?.id, "m1");
  // Walker has no unit plan.
  const walker = pickActivity(hall, "walker", "dressage");
  assert.equal(walker.unit, undefined);
  assert.equal(walker.exercise, null);
  // Without a unit, from rest: the exercise logic of old.
  const fromRest = pickActivity({ ...hall, status: "rest", activity: null, minutes: 0, unit: undefined, exercise: null }, "hall", "dressage");
  assert.equal(fromRest.status, "planned");
  assert.equal(fromRest.minutes, 45);
  assert.equal(fromRest.exercise, null);
  assert.equal(fromRest.unit, undefined);
  assert.deepEqual(pickActivity({ ...hall, unit: undefined }, "arena", "dressage").exercise, hall.exercise);
});

test("an exercise for a part: the main part is the day's single exercise too", () => {
  const other = { id: "m2", title: "Travers" };
  const main = pickUnitExercise(hall, "main", other);
  assert.deepEqual(main.unit?.main, { exercise: other, minutes: 22 });
  assert.deepEqual(main.exercise, other);
  const warm = pickUnitExercise(hall, "warmup", other);
  assert.deepEqual(warm.unit?.warmup?.exercise, other);
  assert.deepEqual(warm.exercise, { id: "m1", title: "Schulterherein" }, "the main exercise is untouched");
  const cleared = pickUnitExercise(hall, "main", null);
  assert.equal(cleared.exercise, null);
  assert.equal(dayEditBody(cleared).exercise_id, "");
  assert.equal(dayEditBody(main).unit?.main?.exercise_id, "m2");
  assert.equal(dayEditBody(main).exercise_id, "m2");
  // A day without a unit gets one from the split.
  const fresh = pickUnitExercise({ ...hall, unit: undefined, exercise: null }, "main", other);
  assert.deepEqual(fresh.unit?.warmup, { exercise: null, minutes: 15 });
  assert.deepEqual(fresh.unit?.main, { exercise: other, minutes: 22 });
  // No activity, nothing to pick for.
  assert.equal(pickUnitExercise({ ...hall, status: "rest", activity: null }, "main", other).unit, hall.unit);
});

test("a rest day has no focus, exercise or unit; only an existing key is removed", () => {
  const rest = pickRest(hall, "takt");
  assert.equal(rest.status, "rest");
  assert.equal(rest.activity, null);
  assert.equal(rest.minutes, 0);
  assert.equal(rest.focusKey, null);
  assert.equal(rest.unit, undefined);
  assert.equal(rest.exercise, null);
  assert.equal(pickRest(hall, undefined).focusKey, undefined);
  assert.deepEqual(dayEditBody(rest), { status: "rest", note: "Plan: Ruhetag" });
});
