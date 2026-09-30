// Pure conversion between the training profile API shape and the editable form state of
// app/horses/[id]/training-profile.tsx, including German validation messages.
import { ACTIVITIES, activityLabel, isValidDate, type Activity, type ActivityMode } from "./training.ts";

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

/** Validates the form and builds the PUT body; the error is a German sentence. */
export function draftToInput(d: ProfileDraft): DraftResult {
  const sMin = int(d.sessionsMin);
  const sMax = int(d.sessionsMax);
  const rMin = int(d.restDaysMin);
  const rMax = int(d.restDaysMax);
  const maxMin = int(d.maxMinutes);
  if (sMin === null || sMax === null || rMin === null || rMax === null || maxMin === null) {
    return { ok: false, error: "Rhythmus: nur ganze Zahlen." };
  }
  if (sMin < 1 || sMax > 7 || sMin > sMax) {
    return { ok: false, error: "Einheiten pro Woche: 1 bis 7, Minimum höchstens Maximum." };
  }
  if (rMax > 6 || rMin > rMax) {
    return { ok: false, error: "Ruhetage pro Woche: 0 bis 6, Minimum höchstens Maximum." };
  }
  if (sMin + rMin > 7) return { ok: false, error: "Einheiten und Ruhetage passen nicht in eine Woche." };
  if (maxMin > 240) return { ok: false, error: "Höchstdauer max. 240 Minuten." };

  const allowed: ProfileLike["allowed_activities"] = [];
  for (const a of ACTIVITIES) {
    const { mode, note } = d.modes[a];
    if (mode === "conditional") {
      if (!note.trim()) return { ok: false, error: "Bedingung für jede bedingte Aktivität fehlt." };
      allowed.push({ activity: a, mode, note: note.trim() });
    } else {
      allowed.push({ activity: a, mode });
    }
  }

  const shows: ProfileLike["shows"] = [];
  for (const s of d.shows) {
    if (!isValidDate(s.date.trim())) return { ok: false, error: "Turnierdatum als JJJJ-MM-TT, z. B. 2026-05-17." };
    if (!s.name.trim()) return { ok: false, error: "Jedes Turnier braucht einen Namen." };
    shows.push({
      date: s.date.trim(),
      name: s.name.trim(),
      ...(s.classes.trim() ? { classes: s.classes.trim() } : {}),
      ...(s.helper.trim() ? { helper: s.helper.trim() } : {}),
    });
  }
  const seasonEnd = d.seasonEnd.trim();
  if (seasonEnd && !isValidDate(seasonEnd)) {
    return { ok: false, error: "Saisonende als JJJJ-MM-TT, z. B. 2026-10-31." };
  }

  const isAllowed = (a: Activity) => d.modes[a].mode !== "off";
  const days: DayRuleApi[] = [];
  for (const choice of d.days) {
    if ((ACTIVITIES as readonly string[]).includes(choice)) {
      const a = choice as Activity;
      if (!isAllowed(a)) return { ok: false, error: `Wochenstruktur: ${activityLabel(a)} ist im Profil ausgeschaltet.` };
      days.push({ kind: "activity", activity: a });
    } else {
      days.push({ kind: choice as DayKind });
    }
  }
  const restDays = days.filter((x) => x.kind === "rest").length;
  if (restDays > rMax) return { ok: false, error: `Wochenstruktur: ${restDays} Ruhetage, erlaubt sind höchstens ${rMax}.` };

  const count = (text: string): number | null => (text.trim() === "" ? 0 : int(text));
  const qDemanding = count(d.quotaDemanding);
  const qRecovery = count(d.quotaRecovery);
  if (qDemanding === null || qRecovery === null) return { ok: false, error: "Wochenziele: nur ganze Zahlen." };
  const quotaActs: Partial<Record<Activity, number>> = {};
  let units = qDemanding + qRecovery;
  for (const a of ACTIVITIES) {
    const n = count(d.quotaActivities[a]);
    if (n === null) return { ok: false, error: "Wochenziele: nur ganze Zahlen." };
    if (n === 0) continue;
    if (!isAllowed(a)) return { ok: false, error: `Wochenziele: ${activityLabel(a)} ist im Profil ausgeschaltet.` };
    quotaActs[a] = n;
    units += n;
  }
  if (qDemanding > 7 || qRecovery > 7 || Object.values(quotaActs).some((n) => (n ?? 0) > 7)) {
    return { ok: false, error: "Wochenziele: höchstens 7 pro Woche." };
  }
  if (units + rMin > 7) return { ok: false, error: "Wochenziele und Ruhetage passen nicht in eine Woche." };

  return {
    ok: true,
    input: {
      discipline: d.discipline,
      level: d.level.trim(),
      allowed_activities: allowed,
      shows,
      season_end: seasonEnd || null,
      rhythm: {
        sessions_min: sMin,
        sessions_max: sMax,
        rest_days_min: rMin,
        rest_days_max: rMax,
        max_minutes: maxMin,
        rest_after_show: d.restAfterShow,
        days,
        quotas: { demanding: qDemanding, recovery: qRecovery, activities: quotaActs },
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
