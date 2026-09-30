// Pure helpers and types for the reminder center and the notification settings (no React
// Native imports, unit-tested). Backend: internal/reminders, docs/domains/reminders.md.
// All user-facing text is German.

import { ApiError, errorMessage } from "./api-core.ts";
import { formatClock, localDate } from "./presence-format.ts";

// --- API types (mirror backend/internal/reminders) ------------------------------------------

export type ReminderItem = {
  /** Row id for stored reminders, `c:...` for computed ones. */
  id: string;
  kind: string;
  title: string;
  body: string;
  /** RFC 3339. All-day items carry local midnight. */
  due_at: string;
  all_day: boolean;
  sent_at: string | null;
  /** App route (same convention as push `data.screen`), "" = none. */
  screen: string;
  computed: boolean;
  dismissible: boolean;
};

export type ReminderGroup = { key: "today" | "week"; label: string; items: ReminderItem[] };

export type BlanketCheckState = "empty" | "done" | "due" | "upcoming";

export type BlanketCheck = {
  day: string;
  /** HH:MM */
  time: string;
  due_at: string;
  done: number;
  total: number;
  state: BlanketCheckState;
  screen: string;
};

export type RemindersResponse = {
  range: "today" | "week";
  today: string;
  timezone: string;
  blanket_check: BlanketCheck;
  groups: ReminderGroup[];
};

export type NotificationKind = {
  kind: string;
  label: string;
  description: string;
  enabled: boolean;
  opt_in: boolean;
};

export type NotificationSettings = { items: NotificationKind[]; push_consent: boolean };

export type ReminderTimeInfo = {
  reminder_time: string;
  can_edit: boolean;
  /** Automatic uncovering (read-only, set by the operator): "HH:MM" or null = off. */
  auto_uncover_time: string | null;
  /** ISO weekdays, 1 = Monday .. 7 = Sunday. */
  auto_uncover_days: number[];
};

// --- kinds ---------------------------------------------------------------------------------

const KIND_LABELS: Record<string, string> = {
  last_person: "Decken",
  weather_change: "Wetter",
  medication: "Medikament",
  health_due: "Termin",
  reha_checkup: "Reha",
  helper: "Anfrage",
  new_request: "Anfrage",
  training_plan: "Training",
  urgent_observation: "Dringend",
  observation: "Auffälligkeit",
};

/** Short German name of a reminder kind for the chip in the list. */
export function kindLabel(kind: string): string {
  return KIND_LABELS[kind] ?? "Erinnerung";
}

// --- time and day formatting ---------------------------------------------------------------

const WEEKDAYS = ["Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"];
const MONTHS = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

function parseDay(day: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day);
  if (!m) return null;
  return new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])));
}

function addDays(day: string, n: number): string {
  const d = parseDay(day);
  if (!d) return day;
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

/** "Heute", "Morgen", "Gestern" or "Freitag, 2. Okt" for a YYYY-MM-DD day, seen from `today`. */
export function dayHeading(day: string, today: string): string {
  if (day === today) return "Heute";
  if (day === addDays(today, 1)) return "Morgen";
  if (day === addDays(today, -1)) return "Gestern";
  const d = parseDay(day);
  if (!d) return day;
  return `${WEEKDAYS[d.getUTCDay()]}, ${d.getUTCDate()}. ${MONTHS[d.getUTCMonth()]}`;
}

/** The moment an item is placed at: when it was sent, else when it is due. */
export function placedAt(item: ReminderItem): string {
  return item.sent_at ?? item.due_at;
}

/** Calendar day (stable time zone) an item belongs to. Overdue items count as `today`. */
export function itemDay(item: ReminderItem, timeZone: string, today: string): string {
  const day = localDate(placedAt(item), timeZone);
  return day < today ? today : day;
}

/** "08:00 Uhr", "ganztägig", or "Gesendet 08:01 Uhr" (only for the sent line). */
export function timeLabel(item: ReminderItem, timeZone: string): string {
  if (item.all_day) return "ganztägig";
  const clock = formatClock(item.due_at, timeZone);
  return clock ? `${clock} Uhr` : "";
}

export function sentLabel(item: ReminderItem, timeZone: string): string {
  if (!item.sent_at) return "";
  const clock = formatClock(item.sent_at, timeZone);
  return clock ? `Gesendet ${clock} Uhr` : "Gesendet";
}

export type DaySection = { day: string; heading: string; items: ReminderItem[] };

/** Splits items into consecutive day sections (sorted by the moment they are placed at). */
export function groupByDay(items: readonly ReminderItem[], timeZone: string, today: string): DaySection[] {
  const sorted = [...items].sort((a, b) => placedAt(a).localeCompare(placedAt(b)) || a.id.localeCompare(b.id));
  const sections: DaySection[] = [];
  for (const item of sorted) {
    const day = itemDay(item, timeZone, today);
    const last = sections[sections.length - 1];
    if (last && last.day === day) last.items.push(item);
    else sections.push({ day, heading: dayHeading(day, today), items: [item] });
  }
  return sections;
}

/** Total number of items over all groups. */
export function countItems(groups: readonly ReminderGroup[]): number {
  return groups.reduce((sum, g) => sum + g.items.length, 0);
}

// --- blanket check hero --------------------------------------------------------------------

export type BlanketHero = { value: string; unit: string; description: string };

/** Texts of the "Deckencheck" hero: progress "4/7" and one sentence for the state. */
export function blanketHero(check: BlanketCheck): BlanketHero {
  const value = `${check.done}/${check.total}`;
  const unit = check.total === 1 ? "Pferd versorgt" : "Pferde versorgt";
  switch (check.state) {
    case "empty":
      return { value: "–", unit: "", description: "Noch keine Pferde." };
    case "done":
      return { value, unit, description: "Alle versorgt." };
    case "due": {
      const open = check.total - check.done;
      return {
        value,
        unit,
        description: `${open === 1 ? "1 Pferd" : `${open} Pferde`} noch offen.`,
      };
    }
    default:
      return { value, unit, description: "Erinnerung folgt." };
  }
}

// --- reminder time of the stable -----------------------------------------------------------

/** Range accepted by the server (`PUT /stables/reminder-time`). */
export const REMINDER_TIME_MIN = "16:00";
export const REMINDER_TIME_MAX = "22:00";
export const REMINDER_TIME_STEP_MINUTES = 15;

/** Minutes since midnight of "HH:MM", or null when it is not a valid clock time. */
export function minutesOf(hhmm: string): number | null {
  const m = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(hhmm);
  return m ? Number(m[1]) * 60 + Number(m[2]) : null;
}

export function formatMinutes(total: number): string {
  const h = Math.floor(total / 60);
  const m = total % 60;
  return `${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

/** True for "HH:MM" between 16:00 and 22:00 (what the server accepts). */
export function isValidReminderTime(hhmm: string): boolean {
  const m = minutesOf(hhmm);
  return m !== null && m >= minutesOf(REMINDER_TIME_MIN)! && m <= minutesOf(REMINDER_TIME_MAX)!;
}

/**
 * Moves the time by `steps` quarter hours and keeps it inside the allowed range. A time
 * that is not on the grid (set on the server) first snaps to the next grid point in that direction.
 */
export function stepReminderTime(hhmm: string, steps: number): string {
  const min = minutesOf(REMINDER_TIME_MIN)!;
  const max = minutesOf(REMINDER_TIME_MAX)!;
  const current = minutesOf(hhmm) ?? min;
  const step = REMINDER_TIME_STEP_MINUTES;
  let next: number;
  if (current % step === 0) next = current + steps * step;
  else if (steps > 0) next = Math.ceil(current / step) * step + (steps - 1) * step;
  else next = Math.floor(current / step) * step + (steps + 1) * step;
  return formatMinutes(Math.min(max, Math.max(min, next)));
}

// --- errors --------------------------------------------------------------------------------

/** German text for the errors of the reminder and settings endpoints. */
export function reminderErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    switch (err.code) {
      case "forbidden":
        return "Nur Verwalter dürfen das.";
      case "validation_failed":
        return "Uhrzeit zwischen 16:00 und 22:00 Uhr wählen.";
      case "not_found":
        return "Diese Erinnerung gibt es nicht mehr.";
    }
  }
  return errorMessage(err);
}
