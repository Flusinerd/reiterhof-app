// Pure helpers for horses and the horse record (no React Native imports, so they can be
// unit-tested with node --test). All user-facing strings are German.

/** Age in whole years (calendar year difference); null without a birth year. */
export function ageFromBirthYear(birthYear: number | null | undefined, now: Date): number | null {
  if (!birthYear || !Number.isFinite(birthYear)) return null;
  const age = now.getFullYear() - birthYear;
  return age < 0 ? null : age;
}

/** "9 Jahre", "1 Jahr", "unter 1 Jahr"; "" without a birth year. */
export function ageText(birthYear: number | null | undefined, now: Date): string {
  const age = ageFromBirthYear(birthYear, now);
  if (age === null) return "";
  if (age === 0) return "unter 1 Jahr";
  return age === 1 ? "1 Jahr" : `${age} Jahre`;
}

const SEX_LABELS: Record<string, string> = { mare: "Stute", gelding: "Wallach", stallion: "Hengst" };

export const SEXES = ["mare", "gelding", "stallion"] as const;

export function sexLabel(sex: string | null | undefined): string {
  return (sex && SEX_LABELS[sex]) || "";
}

/** Joins the non-empty parts with " · ", e.g. "Stute · 9 Jahre · Hannoveraner". */
export function joinParts(parts: (string | null | undefined | false)[]): string {
  return parts.filter((p): p is string => !!p && p.trim() !== "").join(" · ");
}

// --- due dates -----------------------------------------------------------------------------

export type DueLevel = "none" | "overdue" | "soon" | "ok";

/** Items due within this many days are highlighted as "soon". */
export const SOON_DAYS = 7;

export function dueLevel(days: number | null | undefined): DueLevel {
  if (days === null || days === undefined) return "none";
  if (days < 0) return "overdue";
  return days <= SOON_DAYS ? "soon" : "ok";
}

/** "überfällig", "heute", "morgen", "in 3 Tagen", "in 5 Wochen", "in 3 Monaten"; "kein Termin" without a date. */
export function dueText(days: number | null | undefined): string {
  if (days === null || days === undefined) return "kein Termin";
  if (days < 0) return "überfällig";
  if (days === 0) return "heute";
  if (days === 1) return "morgen";
  if (days <= 14) return `in ${days} Tagen`;
  if (days < 60) {
    const weeks = Math.round(days / 7);
    return `in ${weeks} Wochen`;
  }
  const months = Math.round(days / 30);
  return `in ${months} Monaten`;
}

/** Longer text for lists: "seit 11 Tagen überfällig" or the same as dueText. */
export function dueDetailText(days: number | null | undefined): string {
  if (days !== null && days !== undefined && days < 0) {
    const n = -days;
    return n === 1 ? "seit 1 Tag überfällig" : `seit ${n} Tagen überfällig`;
  }
  return dueText(days);
}

/** "2026-10-08" -> "08.10.2026"; other input is returned unchanged. */
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "";
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
  return m ? `${m[3]}.${m[2]}.${m[1]}` : iso;
}

/** Parses "08.10.2026" (also "8.10.2026" and "8.10.26") to "2026-10-08"; null if invalid. */
export function parseGermanDate(text: string): string | null {
  const m = /^\s*(\d{1,2})\.(\d{1,2})\.(\d{2}|\d{4})\s*$/.exec(text);
  if (!m) return null;
  const day = Number(m[1]);
  const month = Number(m[2]);
  let year = Number(m[3]);
  if (m[3].length === 2) year += 2000;
  const d = new Date(Date.UTC(year, month - 1, day));
  if (d.getUTCFullYear() !== year || d.getUTCMonth() !== month - 1 || d.getUTCDate() !== day) return null;
  return `${String(year).padStart(4, "0")}-${String(month).padStart(2, "0")}-${String(day).padStart(2, "0")}`;
}

/** Parses "8:30" or "08:30" to "08:30"; null if invalid. */
export function parseTime(text: string): string | null {
  const m = /^\s*(\d{1,2})[:.](\d{2})\s*$/.exec(text);
  if (!m) return null;
  const h = Number(m[1]);
  const min = Number(m[2]);
  if (h > 23 || min > 59) return null;
  return `${String(h).padStart(2, "0")}:${String(min).padStart(2, "0")}`;
}

/** Whole number from a text field; null for empty or invalid input. */
export function parseWholeNumber(text: string): number | null {
  const t = text.trim();
  if (!/^\d+$/.test(t)) return null;
  return Number(t);
}

// --- health kinds --------------------------------------------------------------------------

export const HEALTH_KINDS = [
  "vaccination",
  "farrier",
  "deworming",
  "dentist",
  "physio",
  "medication",
  "vet",
] as const;
export type HealthKind = (typeof HEALTH_KINDS)[number];

const HEALTH_LABELS: Record<HealthKind, string> = {
  vaccination: "Impfung",
  farrier: "Hufschmied",
  deworming: "Entwurmung",
  dentist: "Zahnarzt",
  physio: "Physiotherapie",
  medication: "Medikament",
  vet: "Tierarzt",
};

export function healthKindLabel(kind: string): string {
  return HEALTH_LABELS[kind as HealthKind] ?? kind;
}

// --- health intervals and reminders --------------------------------------------------------

export const INTERVAL_UNITS = ["day", "week", "month", "year"] as const;
export type IntervalUnit = (typeof INTERVAL_UNITS)[number];

/** Largest count per unit (about ten years, as in the backend). */
export const MAX_INTERVAL_COUNT: Record<IntervalUnit, number> = { day: 3650, week: 520, month: 120, year: 10 };

const UNIT_LABELS: Record<IntervalUnit, [string, string]> = {
  day: ["Tag", "Tage"],
  week: ["Woche", "Wochen"],
  month: ["Monat", "Monate"],
  year: ["Jahr", "Jahre"],
};

/** Plural label of a unit for pickers: "Tage", "Wochen", "Monate", "Jahre". */
export function intervalUnitLabel(unit: string): string {
  return UNIT_LABELS[unit as IntervalUnit]?.[1] ?? unit;
}

/** "1 Tag", "6 Wochen", "6 Monate", "1 Jahr". */
export function intervalText(count: number, unit: string): string {
  const labels = UNIT_LABELS[unit as IntervalUnit] ?? UNIT_LABELS.day;
  return `${count} ${count === 1 ? labels[0] : labels[1]}`;
}

/** "jährlich", "halbjährlich", "alle 6 Wochen", "täglich"; "" without interval. */
export function intervalRepeatText(count: number | null | undefined, unit: string): string {
  if (!count) return "";
  if (count === 1) {
    const single: Record<string, string> = { day: "täglich", week: "wöchentlich", month: "monatlich", year: "jährlich" };
    if (single[unit]) return single[unit];
  }
  if (unit === "month" && count === 6) return "halbjährlich";
  return `alle ${intervalText(count, unit)}`;
}

export type IntervalPreset = { count: number; unit: IntervalUnit };

/** Usual intervals per kind, offered as one-tap choices. */
export const INTERVAL_PRESETS: Record<HealthKind, IntervalPreset[]> = {
  vaccination: [
    { count: 6, unit: "month" },
    { count: 1, unit: "year" },
  ],
  farrier: [
    { count: 6, unit: "week" },
    { count: 8, unit: "week" },
  ],
  deworming: [
    { count: 3, unit: "month" },
    { count: 6, unit: "month" },
  ],
  dentist: [
    { count: 6, unit: "month" },
    { count: 1, unit: "year" },
  ],
  physio: [
    { count: 4, unit: "week" },
    { count: 3, unit: "month" },
  ],
  medication: [],
  vet: [
    { count: 6, unit: "month" },
    { count: 1, unit: "year" },
  ],
};

export function intervalPresets(kind: string): IntervalPreset[] {
  return INTERVAL_PRESETS[kind as HealthKind] ?? [];
}

/** Days before the due date offered for reminders. */
export const REMIND_DAY_OPTIONS = [0, 1, 3, 7, 14, 28] as const;

/** At most this many reminder days per item (as in the backend). */
export const MAX_REMIND_DAYS = 5;

/** "am Tag", "1 Tag vorher", "3 Tage vorher", "1 Woche vorher", "4 Wochen vorher". */
export function remindDayLabel(days: number): string {
  if (days === 0) return "am Tag";
  if (days % 7 === 0) return `${intervalText(days / 7, "week")} vorher`;
  return `${intervalText(days, "day")} vorher`;
}

/** Options for the picker: the fixed ones plus any other value the item already has, largest first. */
export function remindDayOptions(current: number[]): number[] {
  return [...new Set([...REMIND_DAY_OPTIONS, ...current])].sort((a, b) => b - a);
}

/** Adds or removes a day; keeps the list unique, largest first and at most MAX_REMIND_DAYS long. */
export function toggleRemindDay(days: number[], day: number): number[] {
  if (days.includes(day)) return days.filter((d) => d !== day);
  if (days.length >= MAX_REMIND_DAYS) return days;
  return [...days, day].sort((a, b) => b - a);
}

/** "Erinnerung 1 Woche vorher und am Tag, 08:00 Uhr"; "Keine Erinnerung" without days. */
export function remindText(days: number[], time: string): string {
  if (days.length === 0) return "Keine Erinnerung";
  const labels = [...days].sort((a, b) => b - a).map(remindDayLabel);
  const list = labels.length === 1 ? labels[0] : `${labels.slice(0, -1).join(", ")} und ${labels[labels.length - 1]}`;
  return `Erinnerung ${list}, ${time} Uhr`;
}

/** The four tiles of the horse record, in display order. */
export const SUMMARY_KINDS = ["vaccination", "farrier", "deworming", "dentist"] as const;

// --- documents -----------------------------------------------------------------------------

export const DOCUMENT_KINDS = ["passport", "vaccination_record", "insurance", "other"] as const;
export type DocumentKind = (typeof DOCUMENT_KINDS)[number];

const DOCUMENT_LABELS: Record<DocumentKind, string> = {
  passport: "Equidenpass",
  vaccination_record: "Impfpass",
  insurance: "Versicherung",
  other: "Sonstiges",
};

export function documentKindLabel(kind: string): string {
  return DOCUMENT_LABELS[kind as DocumentKind] ?? kind;
}

// --- rider rules ---------------------------------------------------------------------------

export type RiderRule = {
  key: string;
  label: string;
  description: string;
};

/** Positive list of what a rider (RB) may do; keep in sync with backend internal/horses/rules.go. */
export const RIDER_RULES: readonly RiderRule[] = [
  { key: "ride", label: "Reiten", description: "Darf das Pferd reiten." },
  { key: "groom", label: "Pflegen", description: "Darf das Pferd putzen und versorgen." },
  { key: "log_sessions", label: "Einheiten eintragen", description: "Darf Trainingseinheiten eintragen." },
  {
    key: "report_observations",
    label: "Auffälligkeiten melden",
    description: "Darf Auffälligkeiten am Pferd melden.",
  },
  { key: "take_week_slots", label: "Wochenplan", description: "Darf Slots im Wochenplan übernehmen." },
  { key: "hack_alone", label: "Allein ausreiten", description: "Darf allein ins Gelände." },
  { key: "shows", label: "Turniere", description: "Darf das Pferd auf Turnieren reiten." },
];

/** Rules a new rider gets (backend default). */
export const DEFAULT_RIDER_RULES: readonly string[] = ["log_sessions", "report_observations", "take_week_slots"];

export function ruleLabel(key: string): string {
  return RIDER_RULES.find((r) => r.key === key)?.label ?? key;
}

/** "Reiten, Pflegen" in canonical order; "Keine Freigaben" for an empty list. */
export function rulesSummary(rules: readonly string[]): string {
  if (rules.length === 0) return "Keine Freigaben";
  const ordered = RIDER_RULES.filter((r) => rules.includes(r.key)).map((r) => r.label);
  const unknown = rules.filter((k) => !RIDER_RULES.some((r) => r.key === k));
  return [...ordered, ...unknown].join(", ");
}

/** Toggles a rule in a list, keeping the canonical order. */
export function toggleRule(rules: readonly string[], key: string): string[] {
  const next = rules.includes(key) ? rules.filter((r) => r !== key) : [...rules, key];
  const known = RIDER_RULES.map((r) => r.key).filter((k) => next.includes(k));
  return [...known, ...next.filter((k) => !known.includes(k))];
}

// --- phone ---------------------------------------------------------------------------------

/** `tel:` URL for a phone number (keeps digits and a leading +); null if there are no digits. */
export function telUrl(phone: string | null | undefined): string | null {
  if (!phone) return null;
  const trimmed = phone.trim();
  const digits = trimmed.replace(/\D/g, "");
  if (digits === "") return null;
  return `tel:${trimmed.startsWith("+") ? "+" : ""}${digits}`;
}

// --- routes --------------------------------------------------------------------------------

/** App routes of the horse record. Other agents add training-profile, blanket-plan and reha here. */
export const horseRoutes = {
  list: "/horses",
  create: "/horses/new",
  detail: (id: string) => `/horses/${id}`,
  edit: (id: string) => `/horses/${id}/edit`,
  emergency: (id: string) => `/horses/${id}/emergency`,
  health: (id: string) => `/horses/${id}/health`,
  documents: (id: string) => `/horses/${id}/documents`,
  trainingProfile: (id: string) => `/horses/${id}/training-profile`,
  trainingSetup: (id: string) => `/horses/${id}/training-setup`,
  trainingProfileSection: (id: string, section: "activities" | "rhythm" | "structure" | "shows" | "riders") =>
    `/horses/${id}/training-profile/${section}`,
  blanketPlan: (id: string) => `/horses/${id}/blanket-plan`,
  reha: (id: string) => `/horses/${id}/reha`,
} as const;

/** "Termin begleiten": opens the new-request screen (built by the requests feature) for a horse. */
export function appointmentCompanionRoute(horseId: string): string {
  return `/requests/new?type=appointment_companion&horse=${encodeURIComponent(horseId)}`;
}
