import assert from "node:assert/strict";
import { test } from "node:test";

import { draftFromProfile, draftToInput, modeSummary, toggleActivity, type ProfileLike } from "./training-profile.ts";

const profile: ProfileLike = {
  discipline: "dressage",
  level: "L",
  allowed_activities: [
    { activity: "hall", mode: "on" },
    { activity: "hack", mode: "conditional", note: "Nur mit Begleitung" },
    { activity: "jumping", mode: "off" },
  ],
  shows: [{ date: "2026-05-17", name: "Turnier", classes: "Dressur L" }],
  season_end: "2026-10-31",
  rhythm: { sessions_min: 4, sessions_max: 5, rest_days_min: 1, rest_days_max: 2, max_minutes: 60, rest_after_show: true },
  status: "fit",
  rider_rules: [
    { user_id: "u1", name: "Mia", allowed_activities: ["hall"], max_intensity: "medium", may_hack_alone: false, may_ride_shows: false },
  ],
};

test("draft: missing activities are off, all seven activities are present", () => {
  const d = draftFromProfile(profile);
  assert.equal(Object.keys(d.modes).length, 7);
  assert.equal(d.modes.hall.mode, "on");
  assert.equal(d.modes.hack.note, "Nur mit Begleitung");
  assert.equal(d.modes.walker.mode, "off");
  assert.equal(d.sessionsMax, "5");
  assert.deepEqual(modeSummary(d).on, ["hall"]);
  assert.deepEqual(modeSummary(d).conditional, ["hack"]);
});

test("draft round-trips into the PUT body", () => {
  const r = draftToInput(draftFromProfile(profile));
  assert.ok(r.ok);
  if (!r.ok) return;
  assert.equal(r.input.allowed_activities.length, 7);
  assert.deepEqual(r.input.allowed_activities.find((a) => a.activity === "hack"), {
    activity: "hack",
    mode: "conditional",
    note: "Nur mit Begleitung",
  });
  assert.deepEqual(r.input.shows, [{ date: "2026-05-17", name: "Turnier", classes: "Dressur L" }]);
  assert.equal(r.input.season_end, "2026-10-31");
  assert.equal(r.input.rhythm.rest_after_show, true);
  assert.deepEqual(r.input.rider_rules[0]?.allowed_activities, ["hall"]);
});

test("validation messages are German", () => {
  const base = draftFromProfile(profile);
  const bad = (patch: Partial<typeof base>) => {
    const r = draftToInput({ ...base, ...patch });
    assert.ok(!r.ok);
    return r.ok ? "" : r.error;
  };
  assert.match(bad({ sessionsMin: "vier" }), /nur ganze Zahlen/);
  assert.match(bad({ sessionsMin: "0" }), /Einheiten pro Woche/);
  assert.match(bad({ sessionsMin: "6", sessionsMax: "5" }), /Einheiten pro Woche/);
  assert.match(bad({ restDaysMax: "9" }), /Ruhetage/);
  assert.match(bad({ sessionsMin: "7", restDaysMin: "1", sessionsMax: "7" }), /passen nicht/);
  assert.match(bad({ maxMinutes: "300" }), /240/);
  assert.match(bad({ seasonEnd: "31.10." }), /Saisonende/);
  assert.match(bad({ shows: [{ date: "morgen", name: "T", classes: "", helper: "" }] }), /Turnierdatum/);
  assert.match(bad({ shows: [{ date: "2026-05-17", name: " ", classes: "", helper: "" }] }), /Namen/);
  assert.match(
    bad({ modes: { ...base.modes, hall: { mode: "conditional", note: " " } } }),
    /Bedingung/,
  );
  const empty = draftToInput({ ...base, seasonEnd: "", maxMinutes: "0" });
  assert.ok(empty.ok);
  if (empty.ok) assert.equal(empty.input.season_end, null);
});

test("toggleActivity keeps the canonical order", () => {
  assert.deepEqual(toggleActivity(["hall", "walker"], "arena"), ["hall", "arena", "walker"]);
  assert.deepEqual(toggleActivity(["hall", "arena"], "hall"), ["arena"]);
  assert.deepEqual(toggleActivity([], "jumping"), ["jumping"]);
});
