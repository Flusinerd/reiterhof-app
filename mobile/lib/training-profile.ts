// Pure conversion between the training profile API shape and the editable form state of
// app/horses/[id]/training-profile.tsx, including German validation messages.
import {
  ACTIVITIES,
  WEEKDAYS_SHORT,
  activityLabel,
  isValidDate,
  type Activity,
  type ActivityMode,
} from "./training.ts";

export type ShowDraft = { date: string; name: string; classes: string; helper: string };

/** Kinds of a day in the owner's week structure (JAN-93); "" leaves the day to the planner. */
export type DayKind = "" | "rest" | "recovery" | "light" | "normal" | "demanding";

/** What the owner chose for a weekday: a kind or an activity. */
export type DayChoice = DayKind | Activity;

export const DAY_KIND_OPTIONS: readonly { value: DayKind; label: string }[] = [
  { value: "", label: "Frei" },
  { value: "rest", label: "Ruhetag" },
  { value: "recovery", label: "Aktive Erholung" },
  { value: "light", label: "Leicht" },
  { value: "normal", label: "Normal" },
  { value: "demanding", label: "Fordernd" },
];

/** Short German label of a day choice. */
export function dayChoiceLabel(choice: DayChoice): string {
  const kind = DAY_KIND_OPTIONS.find((o) => o.value === choice);
  if (kind) return kind.label;
  return activityLabel(choice);
}

/** Choices for a weekday: the kinds, then the activities the profile allows. */
export function dayChoices(d: Pick<ProfileDraft, "modes">): { value: DayChoice; label: string }[] {
  return [
    ...DAY_KIND_OPTIONS,
    ...ACTIVITIES.filter((a) => d.modes[a].mode !== "off").map((a) => ({ value: a as DayChoice, label: activityLabel(a) })),
  ];
}

export type DayRuleApi = { kind: "" | "rest" | "recovery" | "light" | "normal" | "demanding" | "activity"; activity?: Activity };
export type QuotasApi = { demanding: number; recovery: number; activities?: Partial<Record<Activity, number>> };

export type RiderDraft = {
  userId: string;
  name: string;
  activities: Activity[];
  maxIntensity: "any" | "light" | "medium" | "intense";
  mayHackAlone: boolean;
  mayRideShows: boolean;
};

export type ProfileDraft = {
  discipline: string;
  level: string;
  modes: Record<Activity, { mode: ActivityMode; note: string }>;
  shows: ShowDraft[];
  seasonEnd: string;
  // Numbers are kept as text while typing.
  sessionsMin: string;
  sessionsMax: string;
  restDaysMin: string;
  restDaysMax: string;
  maxMinutes: string;
  restAfterShow: boolean;
  /** Week structure, Monday first (JAN-93). */
  days: DayChoice[];
  /** Quotas per week as text while typing; "" = none. */
  quotaDemanding: string;
  quotaRecovery: string;
  quotaActivities: Record<Activity, string>;
  status: "fit" | "reha" | "pause";
  riders: RiderDraft[];
};

/** Shape of the API profile that the draft needs (a subset of TrainingProfile). */
export type ProfileLike = {
  discipline: string;
  level: string;
  allowed_activities: { activity: Activity; mode: ActivityMode; note?: string }[];
  shows: { date: string; name: string; classes?: string; helper?: string }[];
  season_end: string | null;
  rhythm: {
    sessions_min: number;
    sessions_max: number;
    rest_days_min: number;
    rest_days_max: number;
    max_minutes: number;
    rest_after_show: boolean;
    days?: DayRuleApi[];
    quotas?: QuotasApi;
  };
  status: "fit" | "reha" | "pause";
  rider_rules: {
    user_id: string;
    name?: string;
    allowed_activities: Activity[];
    max_intensity: "any" | "light" | "medium" | "intense";
    may_hack_alone: boolean;
    may_ride_shows: boolean;
  }[];
};

export function draftFromProfile(p: ProfileLike): ProfileDraft {
  const modes = {} as ProfileDraft["modes"];
  for (const a of ACTIVITIES) {
    const entry = p.allowed_activities.find((x) => x.activity === a);
    // Activities missing in the profile are off (same as in the recommender).
    modes[a] = { mode: entry?.mode ?? "off", note: entry?.note ?? "" };
  }
  return {
    discipline: p.discipline || "dressage",
    level: p.level,
    modes,
    shows: p.shows.map((s) => ({ date: s.date, name: s.name, classes: s.classes ?? "", helper: s.helper ?? "" })),
    seasonEnd: p.season_end ?? "",
    sessionsMin: String(p.rhythm.sessions_min),
    sessionsMax: String(p.rhythm.sessions_max),
    restDaysMin: String(p.rhythm.rest_days_min),
    restDaysMax: String(p.rhythm.rest_days_max),
    maxMinutes: String(p.rhythm.max_minutes),
    restAfterShow: p.rhythm.rest_after_show,
    days: Array.from({ length: 7 }, (_, i) => {
      const rule = p.rhythm.days?.[i];
      if (!rule) return "";
      return rule.kind === "activity" ? (rule.activity ?? "") : rule.kind;
    }),
    quotaDemanding: p.rhythm.quotas?.demanding ? String(p.rhythm.quotas.demanding) : "",
    quotaRecovery: p.rhythm.quotas?.recovery ? String(p.rhythm.quotas.recovery) : "",
    quotaActivities: Object.fromEntries(
      ACTIVITIES.map((a) => {
        const n = p.rhythm.quotas?.activities?.[a];
        return [a, n ? String(n) : ""];
      }),
    ) as Record<Activity, string>,
    status: p.status,
    riders: p.rider_rules.map((r) => ({
      userId: r.user_id,
      name: r.name ?? "",
      activities: r.allowed_activities,
      maxIntensity: r.max_intensity,
      mayHackAlone: r.may_hack_alone,
      mayRideShows: r.may_ride_shows,
    })),
  };
}

export type DraftResult =
  | { ok: true; input: ProfileLike }
  | { ok: false; error: string };

function int(text: string): number | null {
  const t = text.trim();
  return /^\d+$/.test(t) ? Number(t) : null;
}

export type ProfileSection = "discipline" | "activities" | "rhythm" | "structure" | "shows" | "riders";
export type SectionResult = { ok: true } | { ok: false; error: string };

const OK: SectionResult = { ok: true };
const fail = (error: string): { ok: false; error: string } => ({ ok: false, error });

type Built<T> = { ok: true; value: T } | { ok: false; error: string };

type RhythmValues = { sMin: number; sMax: number; rMin: number; rMax: number; maxMin: number };

function buildRhythm(d: ProfileDraft): Built<RhythmValues> {
  const sMin = int(d.sessionsMin);
  const sMax = int(d.sessionsMax);
  const rMin = int(d.restDaysMin);
  const rMax = int(d.restDaysMax);
  const maxMin = int(d.maxMinutes);
  if (sMin === null || sMax === null || rMin === null || rMax === null || maxMin === null) {
    return fail("Rhythmus: nur ganze Zahlen.");
  }
  if (sMin < 1 || sMax > 7 || sMin > sMax) {
    return fail("Einheiten pro Woche: 1 bis 7, Minimum höchstens Maximum.");
  }
  if (rMax > 6 || rMin > rMax) {
    return fail("Ruhetage pro Woche: 0 bis 6, Minimum höchstens Maximum.");
  }
  if (sMin + rMin > 7) return fail("Einheiten und Ruhetage passen nicht in eine Woche.");
  if (maxMin > 240) return fail("Höchstdauer max. 240 Minuten.");
  return { ok: true, value: { sMin, sMax, rMin, rMax, maxMin } };
}

/** The API list of activities; a conditional activity needs its condition. */
function buildAllowed(d: Pick<ProfileDraft, "modes">): Built<ProfileLike["allowed_activities"]> {
  const allowed: ProfileLike["allowed_activities"] = [];
  for (const a of ACTIVITIES) {
    const { mode, note } = d.modes[a];
    if (mode === "conditional") {
      if (!note.trim()) return fail("Bedingung für jede bedingte Aktivität fehlt.");
      allowed.push({ activity: a, mode, note: note.trim() });
    } else {
      allowed.push({ activity: a, mode });
    }
  }
  return { ok: true, value: allowed };
}

function buildShows(d: ProfileDraft): Built<{ shows: ProfileLike["shows"]; seasonEnd: string }> {
  const shows: ProfileLike["shows"] = [];
  for (const s of d.shows) {
    if (!isValidDate(s.date.trim())) return fail("Turnierdatum als JJJJ-MM-TT, z. B. 2026-05-17.");
    if (!s.name.trim()) return fail("Jedes Turnier braucht einen Namen.");
    shows.push({
      date: s.date.trim(),
      name: s.name.trim(),
      ...(s.classes.trim() ? { classes: s.classes.trim() } : {}),
      ...(s.helper.trim() ? { helper: s.helper.trim() } : {}),
    });
  }
  const seasonEnd = d.seasonEnd.trim();
  if (seasonEnd && !isValidDate(seasonEnd)) {
    return fail("Saisonende als JJJJ-MM-TT, z. B. 2026-10-31.");
  }
  return { ok: true, value: { shows, seasonEnd } };
}

type StructureValues = { days: DayRuleApi[]; quotas: QuotasApi };

/** Week structure and quotas; the limits come from the rhythm (skipped while the rhythm text is invalid). */
function buildStructure(d: ProfileDraft): Built<StructureValues> {
  const rMin = int(d.restDaysMin);
  const rMax = int(d.restDaysMax);
  const isAllowed = (a: Activity) => d.modes[a].mode !== "off";
  const days: DayRuleApi[] = [];
  for (const choice of d.days) {
    if ((ACTIVITIES as readonly string[]).includes(choice)) {
      const a = choice as Activity;
      if (!isAllowed(a)) return fail(`Wochenstruktur: ${activityLabel(a)} ist im Profil ausgeschaltet.`);
      days.push({ kind: "activity", activity: a });
    } else {
      days.push({ kind: choice as DayKind });
    }
  }
  const restDays = days.filter((x) => x.kind === "rest").length;
  if (rMax !== null && restDays > rMax) {
    return fail(`Wochenstruktur: ${restDays} Ruhetage, erlaubt sind höchstens ${rMax}.`);
  }

  const count = (text: string): number | null => (text.trim() === "" ? 0 : int(text));
  const qDemanding = count(d.quotaDemanding);
  const qRecovery = count(d.quotaRecovery);
  if (qDemanding === null || qRecovery === null) return fail("Wochenziele: nur ganze Zahlen.");
  const quotaActs: Partial<Record<Activity, number>> = {};
  let units = qDemanding + qRecovery;
  for (const a of ACTIVITIES) {
    const n = count(d.quotaActivities[a]);
    if (n === null) return fail("Wochenziele: nur ganze Zahlen.");
    if (n === 0) continue;
    if (!isAllowed(a)) return fail(`Wochenziele: ${activityLabel(a)} ist im Profil ausgeschaltet.`);
    quotaActs[a] = n;
    units += n;
  }
  if (qDemanding > 7 || qRecovery > 7 || Object.values(quotaActs).some((n) => (n ?? 0) > 7)) {
    return fail("Wochenziele: höchstens 7 pro Woche.");
  }
  if (rMin !== null && units + rMin > 7) return fail("Wochenziele und Ruhetage passen nicht in eine Woche.");
  return { ok: true, value: { days, quotas: { demanding: qDemanding, recovery: qRecovery, activities: quotaActs } } };
}

const asResult = (r: Built<unknown>): SectionResult => (r.ok ? OK : r);

/** At least one activity on, and a condition for every conditional one. */
export function validateActivities(d: Pick<ProfileDraft, "modes">): SectionResult {
  if (!ACTIVITIES.some((a) => d.modes[a].mode !== "off")) return fail("Mindestens eine Aktivität einschalten.");
  return asResult(buildAllowed(d));
}

export function validateRhythm(d: ProfileDraft): SectionResult {
  return asResult(buildRhythm(d));
}

/** Week structure and weekly goals. */
export function validateStructure(d: ProfileDraft): SectionResult {
  return asResult(buildStructure(d));
}

/** Shows and the end of the season. */
export function validateShows(d: ProfileDraft): SectionResult {
  return asResult(buildShows(d));
}

/** Validates one section of the profile; discipline and riders have nothing to validate. */
export function validateSection(d: ProfileDraft, s: ProfileSection): SectionResult {
  switch (s) {
    case "activities":
      return validateActivities(d);
    case "rhythm":
      return validateRhythm(d);
    case "structure":
      return validateStructure(d);
    case "shows":
      return validateShows(d);
    case "discipline":
    case "riders":
      return OK;
  }
}

/** Validates the form and builds the PUT body; the error is a German sentence. */
export function draftToInput(d: ProfileDraft): DraftResult {
  const rhythm = buildRhythm(d);
  if (!rhythm.ok) return rhythm;
  const allowed = buildAllowed(d);
  if (!allowed.ok) return allowed;
  const shows = buildShows(d);
  if (!shows.ok) return shows;
  const structure = buildStructure(d);
  if (!structure.ok) return structure;
  const { sMin, sMax, rMin, rMax, maxMin } = rhythm.value;

  return {
    ok: true,
    input: {
      discipline: d.discipline,
      level: d.level.trim(),
      allowed_activities: allowed.value,
      shows: shows.value.shows,
      season_end: shows.value.seasonEnd || null,
      rhythm: {
        sessions_min: sMin,
        sessions_max: sMax,
        rest_days_min: rMin,
        rest_days_max: rMax,
        max_minutes: maxMin,
        rest_after_show: d.restAfterShow,
        days: structure.value.days,
        quotas: structure.value.quotas,
      },
      status: d.status,
      rider_rules: d.riders.map((r) => ({
        user_id: r.userId,
        allowed_activities: r.activities,
        max_intensity: r.maxIntensity,
        may_hack_alone: r.mayHackAlone,
        may_ride_shows: r.mayRideShows,
      })),
    },
  };
}

/** One line per activity that is not fully on, for the read-only view. */
export function modeSummary(d: ProfileDraft): { on: Activity[]; conditional: Activity[]; off: Activity[] } {
  return {
    on: ACTIVITIES.filter((a) => d.modes[a].mode === "on"),
    conditional: ACTIVITIES.filter((a) => d.modes[a].mode === "conditional"),
    off: ACTIVITIES.filter((a) => d.modes[a].mode === "off"),
  };
}

/** Toggles an activity in a rider's allow-list, keeping the canonical order. */
export function toggleActivity(list: readonly Activity[], activity: Activity): Activity[] {
  const has = list.includes(activity);
  return ACTIVITIES.filter((a) => (a === activity ? !has : list.includes(a)));
}

// --- setup wizard and summaries --------------------------------------------------------

export type DisciplineDefaults = Pick<
  ProfileDraft,
  "modes" | "sessionsMin" | "sessionsMax" | "restDaysMin" | "restDaysMax" | "maxMinutes" | "restAfterShow"
>;

type DefaultRow = { on: readonly Activity[]; sessions: [number, number]; rest: [number, number]; maxMinutes: number };

/** Starting values per discipline; the owner adjusts them in the setup. */
const DISCIPLINE_TABLE: Record<string, DefaultRow> = {
  dressage: {
    on: ["hall", "arena", "hack", "lunge", "groundwork", "walker"],
    sessions: [4, 5],
    rest: [1, 2],
    maxMinutes: 60,
  },
  jumping: { on: ACTIVITIES, sessions: [4, 5], rest: [1, 2], maxMinutes: 60 },
  eventing: {
    on: ["hall", "arena", "hack", "lunge", "jumping", "walker"],
    sessions: [5, 6],
    rest: [1, 2],
    maxMinutes: 90,
  },
  leisure: { on: ["hall", "arena", "hack", "groundwork", "walker"], sessions: [3, 5], rest: [2, 3], maxMinutes: 90 },
  western: { on: ["arena", "hack", "groundwork", "walker"], sessions: [4, 5], rest: [1, 2], maxMinutes: 60 },
  young_horse: { on: ["hack", "lunge", "groundwork", "walker"], sessions: [3, 4], rest: [2, 3], maxMinutes: 30 },
};

function modesOn(on: readonly Activity[]): ProfileDraft["modes"] {
  const modes = {} as ProfileDraft["modes"];
  for (const a of ACTIVITIES) modes[a] = { mode: on.includes(a) ? "on" : "off", note: "" };
  return modes;
}

/** Activities, rhythm and rest after a show for a discipline; an unknown one gets the dressage values. */
export function disciplineDefaults(discipline: string): DisciplineDefaults {
  const row = DISCIPLINE_TABLE[discipline] ?? DISCIPLINE_TABLE.dressage!;
  return {
    modes: modesOn(row.on),
    sessionsMin: String(row.sessions[0]),
    sessionsMax: String(row.sessions[1]),
    restDaysMin: String(row.rest[0]),
    restDaysMax: String(row.rest[1]),
    maxMinutes: String(row.maxMinutes),
    restAfterShow: true,
  };
}

function freeDays(): DayChoice[] {
  return ["", "", "", "", "", "", ""];
}

function noQuotas(): Record<Activity, string> {
  return Object.fromEntries(ACTIVITIES.map((a) => [a, ""])) as Record<Activity, string>;
}

/** The empty draft of the setup wizard: no discipline yet, every activity off, seven free days. */
export function setupDraft(): ProfileDraft {
  return {
    discipline: "",
    level: "",
    modes: modesOn([]),
    shows: [],
    seasonEnd: "",
    sessionsMin: "4",
    sessionsMax: "5",
    restDaysMin: "1",
    restDaysMax: "2",
    maxMinutes: "60",
    restAfterShow: true,
    days: freeDays(),
    quotaDemanding: "",
    quotaRecovery: "",
    quotaActivities: noQuotas(),
    status: "fit",
    riders: [],
  };
}

/**
 * Picks a discipline and takes over its defaults (activities, rhythm); resets the week structure
 * and goals, which depend on the activities. Level, status, shows, season end and riders stay.
 */
export function applyDisciplineDefaults(d: ProfileDraft, discipline: string): ProfileDraft {
  return {
    ...d,
    discipline,
    ...disciplineDefaults(discipline),
    days: freeDays(),
    quotaDemanding: "",
    quotaRecovery: "",
    quotaActivities: noQuotas(),
  };
}

/** Why the wizard cannot leave a step (0 discipline, 1 activities, 2 rhythm, 3 week structure, 4 summary). */
export function setupStepError(d: ProfileDraft, step: 0 | 1 | 2 | 3 | 4): string | null {
  switch (step) {
    case 0:
      return d.discipline === "" ? "Disziplin wählen." : null;
    case 1: {
      const r = validateActivities(d);
      return r.ok ? null : r.error;
    }
    case 3: {
      const r = validateStructure(d);
      return r.ok ? null : r.error;
    }
    default:
      return null;
  }
}

/** "5 an · Ausritt bedingt", "6 an" or "Keine Aktivität an". */
export function activitySummary(d: Pick<ProfileDraft, "modes">): string {
  const on = ACTIVITIES.filter((a) => d.modes[a].mode === "on");
  const conditional = ACTIVITIES.filter((a) => d.modes[a].mode === "conditional");
  const count = on.length + conditional.length;
  if (count === 0) return "Keine Aktivität an";
  if (conditional.length === 0) return `${count} an`;
  return `${count} an · ${conditional.map(activityLabel).join(", ")} bedingt`;
}

/** Minimum and maximum as "4–5", or "4" when equal; the raw text while it is not a number. */
function range(min: string, max: string): string {
  const a = min.trim();
  const b = max.trim();
  return a === b ? a : `${a}–${b}`;
}

/** "4–5 Einheiten · 1–2 Ruhetage · max. 60 Min." */
export function rhythmSummary(d: ProfileDraft): string {
  const sessions = range(d.sessionsMin, d.sessionsMax);
  const rest = range(d.restDaysMin, d.restDaysMax);
  const maxMinutes = d.maxMinutes.trim();
  return [
    `${sessions} ${sessions === "1" ? "Einheit" : "Einheiten"}`,
    `${rest} ${rest === "1" ? "Ruhetag" : "Ruhetage"}`,
    maxMinutes === "0" ? "unbegrenzt" : `max. ${maxMinutes} Min.`,
  ].join(" · ");
}

/** "Mo Ruhetag · Sa Ausritt · 2× fordernd", or "Keine festen Tage". */
export function structureSummary(d: ProfileDraft): string {
  const parts: string[] = [];
  d.days.forEach((choice, i) => {
    if (choice !== "") parts.push(`${WEEKDAYS_SHORT[i] ?? ""} ${dayChoiceLabel(choice)}`);
  });
  const quota = (text: string, label: string) => {
    const n = int(text);
    if (n !== null && n > 0) parts.push(`${n}× ${label}`);
  };
  quota(d.quotaDemanding, "fordernd");
  quota(d.quotaRecovery, "aktive Erholung");
  for (const a of ACTIVITIES) quota(d.quotaActivities[a], activityLabel(a));
  return parts.length === 0 ? "Keine festen Tage" : parts.join(" · ");
}

/** "DD.MM." of a YYYY-MM-DD date; "" when it is not (yet) a date. */
function dayMonth(date: string): string {
  const m = /^\d{4}-(\d{2})-(\d{2})$/.exec(date.trim());
  return m ? `${m[2]}.${m[1]}.` : "";
}

/** "Keine Turniere", "1 Turnier · 17.05." or "2 Turniere · nächstes 17.05." (next = first one from today on). */
export function showsSummary(shows: readonly ShowDraft[], today: string): string {
  if (shows.length === 0) return "Keine Turniere";
  if (shows.length === 1) {
    const only = dayMonth(shows[0]!.date);
    return only ? `1 Turnier · ${only}` : "1 Turnier";
  }
  const head = `${shows.length} Turniere`;
  const next = shows
    .map((s) => s.date.trim())
    .filter((date) => dayMonth(date) !== "" && date >= today)
    .sort()[0];
  return next ? `${head} · nächstes ${dayMonth(next)}` : head;
}

/** "Mia: Halle, Longe · Lea: alles", or "Keine Reitbeteiligung". */
export function ridersSummary(riders: readonly RiderDraft[]): string {
  if (riders.length === 0) return "Keine Reitbeteiligung";
  return riders
    .map((r) => {
      const allowed = ACTIVITIES.filter((a) => r.activities.includes(a));
      const what =
        allowed.length === ACTIVITIES.length ? "alles" : allowed.length === 0 ? "nichts" : allowed.map(activityLabel).join(", ");
      return `${r.name || "Reitbeteiligung"}: ${what}`;
    })
    .join(" · ");
}

/** Choices of "Höchstdauer pro Einheit"; 0 means no limit. */
export const MAX_MINUTES_OPTIONS: readonly { value: number; label: string }[] = [
  ...[30, 45, 60, 90, 120].map((value) => ({ value, label: `${value} Min.` })),
  { value: 0, label: "Unbegrenzt" },
];
