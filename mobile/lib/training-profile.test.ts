import assert from "node:assert/strict";
import { test } from "node:test";

import { ACTIVITIES, DISCIPLINES, type Activity } from "./training.ts";
import {
  MAX_MINUTES_OPTIONS,
  activitySummary,
  applyDisciplineDefaults,
  dayChoiceLabel,
  dayChoices,
  disciplineDefaults,
  draftFromProfile,
  draftToInput,
  modeSummary,
  rhythmSummary,
  ridersSummary,
  setupDraft,
  setupStepError,
  showsSummary,
  structureSummary,
  toggleActivity,
  validateActivities,
  validateRhythm,
  validateSection,
  validateShows,
  validateStructure,
  type ProfileDraft,
  type ProfileLike,
  type ProfileSection,
} from "./training-profile.ts";

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

test("week structure and quotas round-trip", () => {
  const p: ProfileLike = {
    ...profile,
    rhythm: {
      ...profile.rhythm,
      days: [
        { kind: "rest" },
        { kind: "" },
        { kind: "demanding" },
        { kind: "activity", activity: "hack" },
        { kind: "recovery" },
        { kind: "" },
        { kind: "" },
      ],
      quotas: { demanding: 2, recovery: 1, activities: { hall: 2 } },
    },
  };
  const d = draftFromProfile(p);
  assert.deepEqual(d.days, ["rest", "", "demanding", "hack", "recovery", "", ""]);
  assert.equal(d.quotaDemanding, "2");
  assert.equal(d.quotaActivities.hall, "2");
  assert.equal(d.quotaActivities.hack, "");
  const r = draftToInput(d);
  assert.ok(r.ok);
  if (!r.ok) return;
  assert.deepEqual(r.input.rhythm.days, p.rhythm.days);
  assert.deepEqual(r.input.rhythm.quotas, { demanding: 2, recovery: 1, activities: { hall: 2 } });
});

test("a profile without structure gets seven free days and no quotas", () => {
  const d = draftFromProfile(profile);
  assert.deepEqual(d.days, ["", "", "", "", "", "", ""]);
  const r = draftToInput(d);
  assert.ok(r.ok);
  if (r.ok) assert.deepEqual(r.input.rhythm.quotas, { demanding: 0, recovery: 0, activities: {} });
});

test("week structure validation", () => {
  const base = draftFromProfile(profile);
  const bad = (patch: Partial<typeof base>) => {
    const r = draftToInput({ ...base, ...patch });
    assert.ok(!r.ok);
    return r.ok ? "" : r.error;
  };
  assert.match(bad({ days: ["jumping", "", "", "", "", "", ""] }), /Springen ist im Profil ausgeschaltet/);
  assert.match(bad({ days: ["rest", "rest", "rest", "", "", "", ""] }), /3 Ruhetage, erlaubt sind höchstens 2/);
  assert.match(bad({ quotaDemanding: "zwei" }), /Wochenziele: nur ganze Zahlen/);
  assert.match(bad({ quotaActivities: { ...base.quotaActivities, jumping: "1" } }), /Springen ist im Profil ausgeschaltet/);
  assert.match(bad({ quotaDemanding: "4", quotaRecovery: "3" }), /passen nicht in eine Woche/);
  assert.match(bad({ quotaActivities: { ...base.quotaActivities, hall: "8" } }), /höchstens 7/);
});

test("day choices offer the kinds and the allowed activities", () => {
  const d = draftFromProfile(profile);
  const labels = dayChoices(d).map((c) => c.label);
  assert.deepEqual(labels, ["Frei", "Ruhetag", "Aktive Erholung", "Leicht", "Normal", "Fordernd", "Halle", "Ausritt"]);
  assert.equal(dayChoiceLabel("recovery"), "Aktive Erholung");
  assert.equal(dayChoiceLabel("hack"), "Ausritt");
});

test("toggleActivity keeps the canonical order", () => {
  assert.deepEqual(toggleActivity(["hall", "walker"], "arena"), ["hall", "arena", "walker"]);
  assert.deepEqual(toggleActivity(["hall", "arena"], "hall"), ["arena"]);
  assert.deepEqual(toggleActivity([], "jumping"), ["jumping"]);
});

// --- per-section validation ------------------------------------------------------------

const errorOf = (r: { ok: boolean; error?: string }) => (r.ok ? null : (r.error ?? null));

test("validateSection messages equal the draftToInput messages for the same faulty draft", () => {
  const base = draftFromProfile(profile);
  const cases: { section: ProfileSection; patch: Partial<ProfileDraft>; message: RegExp }[] = [
    { section: "rhythm", patch: { sessionsMin: "vier" }, message: /nur ganze Zahlen/ },
    { section: "rhythm", patch: { sessionsMin: "0" }, message: /Einheiten pro Woche/ },
    { section: "rhythm", patch: { restDaysMax: "9" }, message: /Ruhetage pro Woche/ },
    { section: "rhythm", patch: { sessionsMin: "7", sessionsMax: "7", restDaysMin: "1" }, message: /passen nicht/ },
    { section: "rhythm", patch: { maxMinutes: "300" }, message: /240/ },
    { section: "activities", patch: { modes: { ...base.modes, hall: { mode: "conditional", note: " " } } }, message: /Bedingung/ },
    { section: "shows", patch: { seasonEnd: "31.10." }, message: /Saisonende/ },
    { section: "shows", patch: { shows: [{ date: "morgen", name: "T", classes: "", helper: "" }] }, message: /Turnierdatum/ },
    { section: "shows", patch: { shows: [{ date: "2026-05-17", name: " ", classes: "", helper: "" }] }, message: /Namen/ },
    { section: "structure", patch: { days: ["jumping", "", "", "", "", "", ""] }, message: /Springen ist im Profil ausgeschaltet/ },
    { section: "structure", patch: { days: ["rest", "rest", "rest", "", "", "", ""] }, message: /3 Ruhetage, erlaubt sind höchstens 2/ },
    { section: "structure", patch: { quotaDemanding: "zwei" }, message: /Wochenziele: nur ganze Zahlen/ },
    { section: "structure", patch: { quotaDemanding: "4", quotaRecovery: "3" }, message: /passen nicht in eine Woche/ },
    { section: "structure", patch: { quotaActivities: { ...base.quotaActivities, hall: "8" } }, message: /höchstens 7/ },
  ];
  for (const c of cases) {
    const d = { ...base, ...c.patch };
    const whole = draftToInput(d);
    assert.ok(!whole.ok);
    const section = validateSection(d, c.section);
    assert.ok(!section.ok, `${c.section} ${JSON.stringify(c.patch)}`);
    assert.match(errorOf(section)!, c.message);
    assert.equal(errorOf(section), errorOf(whole));
  }
});

test("validators accept a good draft; discipline and riders are always ok", () => {
  const d = draftFromProfile(profile);
  for (const s of ["discipline", "activities", "rhythm", "structure", "shows", "riders"] as const) {
    assert.deepEqual(validateSection(d, s), { ok: true });
  }
  assert.deepEqual(validateActivities(d), { ok: true });
  assert.deepEqual(validateRhythm(d), { ok: true });
  assert.deepEqual(validateStructure(d), { ok: true });
  assert.deepEqual(validateShows(d), { ok: true });
  assert.deepEqual(validateSection({ ...d, discipline: "", sessionsMin: "x" }, "discipline"), { ok: true });
  assert.deepEqual(validateSection({ ...d, sessionsMin: "x" }, "riders"), { ok: true });
});

test("draftToInput reports the first fault in the old order: rhythm, condition, shows, structure", () => {
  const base = draftFromProfile(profile);
  const faulty = {
    ...base,
    sessionsMin: "0",
    modes: { ...base.modes, hall: { mode: "conditional" as const, note: "" } },
    shows: [{ date: "x", name: "", classes: "", helper: "" }],
    days: ["jumping", "", "", "", "", "", ""] as ProfileDraft["days"],
  };
  assert.match(errorOf(draftToInput(faulty))!, /Einheiten pro Woche/);
  const noRhythm = { ...faulty, sessionsMin: "4" };
  assert.match(errorOf(draftToInput(noRhythm))!, /Bedingung/);
  const noCondition = { ...noRhythm, modes: base.modes };
  assert.match(errorOf(draftToInput(noCondition))!, /Turnierdatum/);
  const noShows = { ...noCondition, shows: [] };
  assert.match(errorOf(draftToInput(noShows))!, /Wochenstruktur/);
});

test("only validateActivities demands an activity; draftToInput keeps accepting none", () => {
  const d = setupDraft();
  assert.equal(errorOf(validateActivities(d)), "Mindestens eine Aktivität einschalten.");
  assert.equal(errorOf(validateSection(d, "activities")), "Mindestens eine Aktivität einschalten.");
  assert.ok(draftToInput(d).ok);
  const conditional = { ...d, modes: { ...d.modes, hack: { mode: "conditional" as const, note: " " } } };
  assert.equal(errorOf(validateActivities(conditional)), "Bedingung für jede bedingte Aktivität fehlt.");
  const ok = { ...d, modes: { ...d.modes, hack: { mode: "conditional" as const, note: "Nur mit Begleitung" } } };
  assert.deepEqual(validateActivities(ok), { ok: true });
});

test("structure validation skips the rhythm limits while the rhythm text is invalid", () => {
  const d = {
    ...draftFromProfile(profile),
    restDaysMax: "x",
    restDaysMin: "",
    days: ["rest", "rest", "rest", "", "", "", ""] as ProfileDraft["days"],
  };
  assert.deepEqual(validateStructure(d), { ok: true });
});

// --- discipline defaults and the setup draft -------------------------------------------

test("every discipline has defaults the backend accepts", () => {
  for (const { value } of DISCIPLINES) {
    const defaults = disciplineDefaults(value);
    assert.ok(
      ACTIVITIES.some((a) => defaults.modes[a].mode === "on"),
      `${value} has an activity on`,
    );
    assert.ok(ACTIVITIES.every((a) => defaults.modes[a].note === ""));
    const sMin = Number(defaults.sessionsMin);
    const sMax = Number(defaults.sessionsMax);
    const rMin = Number(defaults.restDaysMin);
    const rMax = Number(defaults.restDaysMax);
    assert.ok(sMin >= 1 && sMax <= 7 && sMin <= sMax, value);
    assert.ok(rMin >= 0 && rMax <= 6 && rMin <= rMax, value);
    assert.ok(sMin + rMin <= 7, value);
    assert.equal(defaults.restAfterShow, true);
    // The defaults pass the whole validation.
    assert.ok(draftToInput(applyDisciplineDefaults(setupDraft(), value)).ok, value);
  }
});

test("discipline defaults follow the table", () => {
  const on = (d: string): Activity[] => ACTIVITIES.filter((a) => disciplineDefaults(d).modes[a].mode === "on");
  assert.deepEqual(on("dressage"), ["hall", "arena", "hack", "lunge", "groundwork", "walker"]);
  assert.deepEqual(on("jumping"), [...ACTIVITIES]);
  assert.deepEqual(on("eventing"), ["hall", "arena", "hack", "lunge", "jumping", "walker"]);
  assert.deepEqual(on("leisure"), ["hall", "arena", "hack", "groundwork", "walker"]);
  assert.deepEqual(on("western"), ["arena", "hack", "groundwork", "walker"]);
  assert.deepEqual(on("young_horse"), ["hack", "lunge", "groundwork", "walker"]);
  const pick = (d: string) => {
    const x = disciplineDefaults(d);
    return [x.sessionsMin, x.sessionsMax, x.restDaysMin, x.restDaysMax, x.maxMinutes].join(",");
  };
  assert.equal(pick("dressage"), "4,5,1,2,60");
  assert.equal(pick("jumping"), "4,5,1,2,60");
  assert.equal(pick("eventing"), "5,6,1,2,90");
  assert.equal(pick("leisure"), "3,5,2,3,90");
  assert.equal(pick("western"), "4,5,1,2,60");
  assert.equal(pick("young_horse"), "3,4,2,3,30");
});

test("an unknown discipline gets the dressage defaults; the results are independent copies", () => {
  assert.deepEqual(disciplineDefaults("polo"), disciplineDefaults("dressage"));
  const a = disciplineDefaults("dressage");
  a.modes.hall.mode = "off";
  assert.equal(disciplineDefaults("dressage").modes.hall.mode, "on");
});

test("setupDraft starts empty but valid for the rhythm", () => {
  const d = setupDraft();
  assert.equal(d.discipline, "");
  assert.equal(d.level, "");
  assert.equal(d.status, "fit");
  assert.ok(ACTIVITIES.every((a) => d.modes[a].mode === "off" && d.modes[a].note === ""));
  assert.deepEqual([d.sessionsMin, d.sessionsMax, d.restDaysMin, d.restDaysMax, d.maxMinutes], ["4", "5", "1", "2", "60"]);
  assert.equal(d.restAfterShow, true);
  assert.deepEqual(d.days, ["", "", "", "", "", "", ""]);
  assert.equal(d.quotaDemanding, "");
  assert.equal(d.quotaRecovery, "");
  assert.ok(ACTIVITIES.every((a) => d.quotaActivities[a] === ""));
  assert.deepEqual(d.shows, []);
  assert.deepEqual(d.riders, []);
  assert.deepEqual(validateRhythm(d), { ok: true });
  // Two calls never share state.
  d.days[0] = "rest";
  assert.equal(setupDraft().days[0], "");
});

test("applyDisciplineDefaults sets the defaults and keeps level, status, shows, season end and riders", () => {
  const start: ProfileDraft = {
    ...setupDraft(),
    level: "L",
    status: "reha",
    seasonEnd: "2026-10-31",
    shows: [{ date: "2026-05-17", name: "Turnier", classes: "", helper: "" }],
    riders: [{ userId: "u1", name: "Mia", activities: ["hall"], maxIntensity: "any", mayHackAlone: false, mayRideShows: false }],
    days: ["rest", "", "", "", "", "", ""],
    quotaDemanding: "2",
    quotaActivities: { ...setupDraft().quotaActivities, hall: "1" },
  };
  const d = applyDisciplineDefaults(start, "western");
  assert.equal(d.discipline, "western");
  assert.equal(d.modes.arena.mode, "on");
  assert.equal(d.modes.hall.mode, "off");
  assert.equal(d.sessionsMin, "4");
  assert.equal(d.maxMinutes, "60");
  assert.deepEqual(d.days, ["", "", "", "", "", "", ""]);
  assert.equal(d.quotaDemanding, "");
  assert.equal(d.quotaActivities.hall, "");
  assert.equal(d.level, "L");
  assert.equal(d.status, "reha");
  assert.equal(d.seasonEnd, "2026-10-31");
  assert.equal(d.shows.length, 1);
  assert.equal(d.riders[0]?.name, "Mia");
  // The input draft stays untouched.
  assert.equal(start.discipline, "");
  assert.equal(start.quotaDemanding, "2");
  assert.equal(start.days[0], "rest");
});

test("setupStepError per step", () => {
  const d = setupDraft();
  assert.equal(setupStepError(d, 0), "Disziplin wählen.");
  assert.equal(setupStepError(d, 1), "Mindestens eine Aktivität einschalten.");
  assert.equal(setupStepError(d, 2), null);
  assert.equal(setupStepError(d, 3), null);
  assert.equal(setupStepError(d, 4), null);

  const picked = applyDisciplineDefaults(d, "jumping");
  assert.equal(setupStepError(picked, 0), null);
  assert.equal(setupStepError(picked, 1), null);
  const conditional = { ...picked, modes: { ...picked.modes, hack: { mode: "conditional" as const, note: "" } } };
  assert.equal(setupStepError(conditional, 1), "Bedingung für jede bedingte Aktivität fehlt.");

  const tooManyRestDays = { ...picked, days: ["rest", "rest", "rest", "", "", "", ""] as ProfileDraft["days"] };
  assert.equal(setupStepError(tooManyRestDays, 3), "Wochenstruktur: 3 Ruhetage, erlaubt sind höchstens 2.");
  assert.equal(setupStepError(tooManyRestDays, 2), null);
});

// --- summaries -------------------------------------------------------------------------

test("activitySummary", () => {
  assert.equal(activitySummary(setupDraft()), "Keine Aktivität an");
  assert.equal(activitySummary(disciplineDefaults("dressage")), "6 an");
  const d = disciplineDefaults("eventing");
  assert.equal(activitySummary({ modes: { ...d.modes, hack: { mode: "conditional", note: "x" } } }), "6 an · Ausritt bedingt");
  assert.equal(
    activitySummary({ modes: { ...d.modes, hack: { mode: "conditional", note: "x" }, hall: { mode: "conditional", note: "y" } } }),
    "6 an · Halle, Ausritt bedingt",
  );
  const onlyConditional = { ...setupDraft().modes, hack: { mode: "conditional" as const, note: "x" } };
  assert.equal(activitySummary({ modes: onlyConditional }), "1 an · Ausritt bedingt");
});

test("rhythmSummary", () => {
  assert.equal(rhythmSummary(applyDisciplineDefaults(setupDraft(), "dressage")), "4–5 Einheiten · 1–2 Ruhetage · max. 60 Min.");
  const d = setupDraft();
  assert.equal(rhythmSummary({ ...d, maxMinutes: "0" }), "4–5 Einheiten · 1–2 Ruhetage · unbegrenzt");
  assert.equal(rhythmSummary({ ...d, sessionsMin: "4", sessionsMax: "4" }), "4 Einheiten · 1–2 Ruhetage · max. 60 Min.");
  assert.equal(rhythmSummary({ ...d, restDaysMin: "2", restDaysMax: "2" }), "4–5 Einheiten · 2 Ruhetage · max. 60 Min.");
  assert.equal(
    rhythmSummary({ ...d, sessionsMin: "1", sessionsMax: "1", restDaysMin: "1", restDaysMax: "1" }),
    "1 Einheit · 1 Ruhetag · max. 60 Min.",
  );
});

test("structureSummary", () => {
  const d = setupDraft();
  assert.equal(structureSummary(d), "Keine festen Tage");
  assert.equal(structureSummary({ ...d, days: ["rest", "", "", "", "", "hack", ""] }), "Mo Ruhetag · Sa Ausritt");
  assert.equal(
    structureSummary({
      ...d,
      days: ["rest", "", "", "", "", "hack", ""],
      quotaDemanding: "2",
      quotaRecovery: "1",
      quotaActivities: { ...d.quotaActivities, hall: "2" },
    }),
    "Mo Ruhetag · Sa Ausritt · 2× fordernd · 1× aktive Erholung · 2× Halle",
  );
  assert.equal(structureSummary({ ...d, quotaDemanding: "2" }), "2× fordernd");
  assert.equal(structureSummary({ ...d, days: ["", "", "demanding", "", "", "", "recovery"] }), "Mi Fordernd · So Aktive Erholung");
  assert.equal(structureSummary({ ...d, quotaDemanding: "0", quotaRecovery: "x" }), "Keine festen Tage");
});

test("showsSummary", () => {
  const show = (date: string) => ({ date, name: "T", classes: "", helper: "" });
  assert.equal(showsSummary([], "2026-09-30"), "Keine Turniere");
  assert.equal(showsSummary([show("2026-05-17")], "2026-09-30"), "1 Turnier · 17.05.");
  assert.equal(showsSummary([show("")], "2026-09-30"), "1 Turnier");
  assert.equal(showsSummary([show("2026-05-17"), show("2026-10-03")], "2026-09-30"), "2 Turniere · nächstes 03.10.");
  assert.equal(showsSummary([show("2026-11-01"), show("2026-10-03")], "2026-09-30"), "2 Turniere · nächstes 03.10.");
  assert.equal(showsSummary([show("2026-10-03"), show("2026-11-01")], "2026-10-03"), "2 Turniere · nächstes 03.10.");
  assert.equal(showsSummary([show("2026-05-17"), show("2026-06-01")], "2026-09-30"), "2 Turniere");
});

test("ridersSummary", () => {
  const rider = (name: string, activities: Activity[]) => ({
    userId: name,
    name,
    activities,
    maxIntensity: "any" as const,
    mayHackAlone: false,
    mayRideShows: false,
  });
  assert.equal(ridersSummary([]), "Keine Reitbeteiligung");
  assert.equal(ridersSummary([rider("Mia", ["lunge", "hall"]), rider("Lea", [...ACTIVITIES])]), "Mia: Halle, Longe · Lea: alles");
  assert.equal(ridersSummary([rider("Mia", [])]), "Mia: nichts");
});

test("max minutes options", () => {
  assert.deepEqual(
    MAX_MINUTES_OPTIONS.map((o) => o.value),
    [30, 45, 60, 90, 120, 0],
  );
  assert.equal(MAX_MINUTES_OPTIONS[0]?.label, "30 Min.");
  assert.equal(MAX_MINUTES_OPTIONS[4]?.label, "120 Min.");
  assert.equal(MAX_MINUTES_OPTIONS[5]?.label, "Unbegrenzt");
});
