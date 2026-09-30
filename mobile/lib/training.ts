// Pure helpers of the training feature (no React Native imports, unit-tested with node --test):
// activity labels and icon keys, duration chips, dot and segment view models, week row
// labels, session summaries. API shapes are in lib/api/training.ts.

export type Activity = "hall" | "arena" | "hack" | "lunge" | "jumping" | "groundwork" | "walker";
export type ActivityOrRest = Activity | "rest";
export type Intensity = "none" | "light" | "medium" | "intense";

/** Canonical order, same as the backend (also the order of chips). */
export const ACTIVITIES: readonly Activity[] = [
  "hall",
  "arena",
  "hack",
  "lunge",
  "jumping",
  "groundwork",
  "walker",
];

const ACTIVITY_LABELS: Record<ActivityOrRest, string> = {
  hall: "Halle",
  arena: "Platz",
  hack: "Ausritt",
  lunge: "Longe",
  jumping: "Springen",
  groundwork: "Bodenarbeit",
  walker: "Führanlage",
  rest: "Ruhetag",
};

export function isActivity(value: unknown): value is Activity {
  return typeof value === "string" && (ACTIVITIES as readonly string[]).includes(value);
}

export function activityLabel(activity: string): string {
  return ACTIVITY_LABELS[activity as ActivityOrRest] ?? activity;
}

/** Key of the Lucide icon used for an activity; components/training-activity-icon.tsx maps it. */
export type ActivityIconKey =
  | "warehouse"
  | "trees"
  | "route"
  | "rotate"
  | "trending-up"
  | "hand"
  | "footprints"
  | "moon";

const ACTIVITY_ICONS: Record<ActivityOrRest, ActivityIconKey> = {
  hall: "warehouse",
  arena: "trees",
  hack: "route",
  lunge: "rotate",
  jumping: "trending-up",
  groundwork: "hand",
  walker: "footprints",
  rest: "moon",
};

export function activityIconKey(activity: string): ActivityIconKey {
  return ACTIVITY_ICONS[activity as ActivityOrRest] ?? "route";
}

/** Default minutes when the user picks an activity themselves (mirrors the recommender). */
export const DEFAULT_MINUTES: Record<Activity, number> = {
  hall: 45,
  arena: 45,
  hack: 60,
  lunge: 25,
  jumping: 40,
  groundwork: 30,
  walker: 30,
};

/** Duration chips of "Nur eintragen" (minutes). */
export const QUICK_DURATIONS: readonly number[] = [20, 30, 45, 60];

/** "Wie viel Zeit?" chips of the training tab (minutes available today). */
export const TIME_OPTIONS: readonly number[] = [30, 45, 60, 90];

export function formatMinutes(minutes: number): string {
  return `${Math.round(minutes)} Min.`;
}

/** "mm:ss", or "h:mm:ss" from one hour on. */
export function formatClock(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${pad(m)}:${pad(sec)}`;
}

/** Tracked seconds to whole minutes, at least 1 (the API requires >= 1). */
export function secondsToMinutes(seconds: number): number {
  return Math.max(1, Math.round(seconds / 60));
}

// --- intensity -------------------------------------------------------------------------

const INTENSITY_LABELS: Record<Intensity, string> = {
  none: "keine",
  light: "leicht",
  medium: "mittel",
  intense: "intensiv",
};

export function intensityLabel(intensity: string): string {
  return INTENSITY_LABELS[intensity as Intensity] ?? intensity;
}

// --- dates -----------------------------------------------------------------------------

export const WEEKDAYS_SHORT = ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"] as const;
export const WEEKDAYS_LONG = [
  "Montag",
  "Dienstag",
  "Mittwoch",
  "Donnerstag",
  "Freitag",
  "Samstag",
  "Sonntag",
] as const;
const MONTHS = [
  "Januar",
  "Februar",
  "März",
  "April",
  "Mai",
  "Juni",
  "Juli",
  "August",
  "September",
  "Oktober",
  "November",
  "Dezember",
] as const;

/** Parses YYYY-MM-DD as a UTC date (no timezone shifts). */
export function parseDate(date: string): Date {
  const [y, m, d] = date.split("-").map(Number);
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1));
}

/** Weekday index, 0 = Monday. */
export function weekdayIndex(date: string): number {
  return (parseDate(date).getUTCDay() + 6) % 7;
}

export function weekdayShort(date: string): string {
  return WEEKDAYS_SHORT[weekdayIndex(date)] ?? "";
}

/** "Mittwoch, 25. März". */
export function formatDayLong(date: string): string {
  const d = parseDate(date);
  return `${WEEKDAYS_LONG[weekdayIndex(date)]}, ${d.getUTCDate()}. ${MONTHS[d.getUTCMonth()]}`;
}

/** "25.03.2026". */
export function formatDate(date: string): string {
  const [y, m, d] = date.split("-");
  return `${d}.${m}.${y}`;
}

/** Adds days to a YYYY-MM-DD date. */
export function addDays(date: string, days: number): string {
  const d = parseDate(date);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

/** "23.–29. März" or "30. März – 5. April" for a week. */
export function weekRangeLabel(start: string, end: string): string {
  const a = parseDate(start);
  const b = parseDate(end);
  if (a.getUTCMonth() === b.getUTCMonth()) {
    return `${a.getUTCDate()}.–${b.getUTCDate()}. ${MONTHS[b.getUTCMonth()]}`;
  }
  return `${a.getUTCDate()}. ${MONTHS[a.getUTCMonth()]} – ${b.getUTCDate()}. ${MONTHS[b.getUTCMonth()]}`;
}

// --- 7-day dots ("Was heute?") ---------------------------------------------------------

export type DotKind = "trained" | "rest" | "nothing";
export type ApiDot = { date: string; kind: DotKind; intensity: Intensity; is_today: boolean };

/** Visual tone of a dot: empty (nothing), rest, or the intensity of a trained day. */
export type DotTone = "empty" | "rest" | "light" | "medium" | "intense";

export type DotViewModel = {
  key: string;
  label: string;
  tone: DotTone;
  isToday: boolean;
  accessibilityLabel: string;
};

export function dotTone(kind: DotKind, intensity: Intensity): DotTone {
  if (kind === "rest") return "rest";
  if (kind === "nothing") return "empty";
  return intensity === "medium" || intensity === "intense" ? intensity : "light";
}

export function dotViewModels(dots: readonly ApiDot[]): DotViewModel[] {
  return dots.map((d) => {
    const tone = dotTone(d.kind, d.intensity);
    const what =
      d.kind === "trained"
        ? `trainiert, ${intensityLabel(tone === "empty" || tone === "rest" ? "none" : tone)}`
        : d.kind === "rest"
          ? "Ruhetag"
          : "nichts eingetragen";
    return {
      key: d.date,
      label: weekdayShort(d.date),
      tone,
      isToday: d.is_today,
      accessibilityLabel: `${d.is_today ? "Heute" : formatDayLong(d.date)}: ${what}`,
    };
  });
}

/** Number of trained days in the dots. */
export function trainedCount(dots: readonly ApiDot[]): number {
  return dots.filter((d) => d.kind === "trained").length;
}

// --- week segments and rows ------------------------------------------------------------

export type ApiSegment = { date: string; load: number; level: Intensity; sessions: number };

export type SegmentViewModel = {
  key: string;
  label: string;
  level: Intensity;
  /** 0..1, height of the bar relative to the tallest possible bar. */
  ratio: number;
  accessibilityLabel: string;
};

const SEGMENT_RATIO: Record<Intensity, number> = { none: 0.08, light: 0.34, medium: 0.67, intense: 1 };

export function segmentViewModels(segments: readonly ApiSegment[]): SegmentViewModel[] {
  return segments.map((s) => ({
    key: s.date,
    label: weekdayShort(s.date),
    level: s.level,
    ratio: SEGMENT_RATIO[s.level] ?? SEGMENT_RATIO.none,
    accessibilityLabel:
      s.sessions === 0
        ? `${weekdayShort(s.date)}: keine Belastung`
        : `${weekdayShort(s.date)}: Belastung ${intensityLabel(s.level)}`,
  }));
}

export type DayStatus = "done" | "today" | "planned" | "open" | "empty" | "rest";

export type RowUser = { id: string; name: string } | null;

/** Short status text of a week row. */
export function dayStatusLabel(status: DayStatus, user: RowUser, isMe: boolean, restReason?: string): string {
  switch (status) {
    case "done":
      return isMe ? "Ich, erledigt" : user ? `${user.name}, erledigt` : "Erledigt";
    case "today":
      return user ? `Heute: ${isMe ? "Ich" : user.name}` : "Heute noch offen";
    case "planned":
      return isMe ? "Ich" : (user?.name ?? "Geplant");
    case "open":
      return "Niemand eingetragen";
    case "empty":
      return "Kein Training";
    case "rest":
      return restReason === "after_show" ? "Ruhetag nach dem Turnier" : "Ruhetag";
  }
}

// --- session summary and finish screen -------------------------------------------------

export const FEEL_OPTIONS = [
  { value: "fresh", label: "Frisch" },
  { value: "loose", label: "Locker" },
  { value: "tired", label: "Müde" },
  { value: "tense", label: "Klemmig" },
] as const;
export type Feel = (typeof FEEL_OPTIONS)[number]["value"];

export const FOCUS_OPTIONS = [
  { value: 1, label: "Schwer" },
  { value: 2, label: "Besser" },
  { value: 3, label: "Sitzt" },
] as const;

export type GaitShareRow = { gait: string; label: string; percent: number };

const GAIT_LABELS: Record<string, string> = { halt: "Halt", walk: "Schritt", trot: "Trab", canter: "Galopp" };

/** Gait shares (fractions 0..1) as rows with rounded percentages, biggest first; empty when nothing is tracked. */
export function gaitShareRows(shares: Record<string, number> | null | undefined): GaitShareRow[] {
  if (!shares) return [];
  return Object.entries(shares)
    .filter(([g, v]) => g in GAIT_LABELS && v > 0)
    .map(([gait, v]) => ({ gait, label: GAIT_LABELS[gait] ?? gait, percent: Math.round(v * 100) }))
    .sort((a, b) => b.percent - a.percent);
}

export type ReinSegment = { rein: "left" | "right"; minutes: number };

export type ReinSummary = { leftPercent: number; rightPercent: number; changes: number };

/** Rein shares and number of rein changes; null when there are no segments. */
export function reinSummary(segments: readonly ReinSegment[] | null | undefined): ReinSummary | null {
  if (!segments || segments.length === 0) return null;
  let left = 0;
  let right = 0;
  let changes = 0;
  segments.forEach((s, i) => {
    if (s.rein === "left") left += s.minutes;
    else right += s.minutes;
    if (i > 0 && segments[i - 1]?.rein !== s.rein) changes++;
  });
  const total = left + right;
  if (total <= 0) return { leftPercent: 0, rightPercent: 0, changes };
  const leftPercent = Math.round((left / total) * 100);
  return { leftPercent, rightPercent: 100 - leftPercent, changes };
}

/** Params the tracking screen hands to the finish screen (all strings, as router params are). */
export type FinishParams = {
  horse: string;
  activity: Activity;
  minutes: number;
  startedAt?: string;
  exerciseId?: string;
  gaitShares?: Record<string, number>;
  reinChanges?: ReinSegment[];
  distanceM?: number;
};

function parseJson<T>(raw: string | undefined): T | undefined {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return undefined;
  }
}

/** Reads the finish screen params; returns null when horse or activity is missing/invalid. */
export function parseFinishParams(p: Record<string, string | string[] | undefined>): FinishParams | null {
  const one = (k: string) => {
    const v = p[k];
    return Array.isArray(v) ? v[0] : v;
  };
  const horse = one("horse");
  const activity = one("activity");
  if (!horse || !isActivity(activity)) return null;
  const minutes = Math.round(Number(one("minutes")));
  const distance = Number(one("distance_m"));
  return {
    horse,
    activity,
    minutes: Number.isFinite(minutes) && minutes >= 1 ? Math.min(minutes, 600) : 1,
    startedAt: one("started_at") || undefined,
    exerciseId: one("exercise") || undefined,
    gaitShares: parseJson<Record<string, number>>(one("gait")),
    reinChanges: parseJson<ReinSegment[]>(one("rein")),
    distanceM: Number.isFinite(distance) && distance > 0 ? Math.round(distance) : undefined,
  };
}

/** "Luna war ..." lead-in of the feel chips. */
export function feelPrompt(horseName: string): string {
  return `${horseName} war …`;
}

// --- errors ----------------------------------------------------------------------------

/** German text for training-specific API error codes, or null for other codes. */
export function trainingErrorText(code: string): string | null {
  switch (code) {
    case "forbidden":
      return "Keine Berechtigung.";
    case "conflict":
      return "Dieser Tag ist schon vergeben.";
    case "not_found":
      return "Nicht gefunden.";
    default:
      return null;
  }
}

// --- profile editing -------------------------------------------------------------------

export const DISCIPLINES = [
  { value: "dressage", label: "Dressur" },
  { value: "jumping", label: "Springen" },
  { value: "eventing", label: "Vielseitigkeit" },
  { value: "leisure", label: "Freizeit" },
  { value: "western", label: "Western" },
  { value: "young_horse", label: "Jungpferd" },
] as const;

export function disciplineLabel(value: string): string {
  return DISCIPLINES.find((d) => d.value === value)?.label ?? value;
}

export const PROFILE_STATUSES = [
  { value: "fit", label: "Fit" },
  { value: "reha", label: "Reha" },
  { value: "pause", label: "Pause" },
] as const;

export function statusLabel(value: string): string {
  return PROFILE_STATUSES.find((s) => s.value === value)?.label ?? value;
}

export type ActivityMode = "on" | "off" | "conditional";
export const ACTIVITY_MODES: readonly { value: ActivityMode; label: string }[] = [
  { value: "on", label: "An" },
  { value: "conditional", label: "Bedingt" },
  { value: "off", label: "Aus" },
];

export const MAX_INTENSITY_OPTIONS = [
  { value: "any", label: "Egal" },
  { value: "light", label: "Leicht" },
  { value: "medium", label: "Mittel" },
  { value: "intense", label: "Intensiv" },
] as const;

/** Matches YYYY-MM-DD with a real calendar date. */
export function isValidDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  return parseDate(value).toISOString().slice(0, 10) === value;
}
