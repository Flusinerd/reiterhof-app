import assert from "node:assert/strict";
import { test } from "node:test";

import { MAX_NOTE, canAskForAI, planApplyBody, planDayNote, planStatusText, type PlanDay, type PlanResponse } from "./training-plan.ts";

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
