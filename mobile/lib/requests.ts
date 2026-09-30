// Pure helpers for help requests (M4): types, labels, formatting, recurrence text.
// No React Native imports, so this file is unit-tested with node --test.
// The wire types mirror docs/domains/requests.md.

export type RequestType =
  | "blanket"
  | "show_helper"
  | "ride_share"
  | "exercise"
  | "feed_or_turnout"
  | "appointment_companion"
  | "other";

export type RequestStatus = "open" | "assigned" | "done" | "cancelled";

export type RequestHelper = {
  user_id: string;
  name: string;
  avatar_color: string | null;
  thanked: boolean;
  joined_at: string;
};

export type HelpRequest = {
  id: string;
  type: RequestType;
  horse_id: string | null;
  horse_name: string | null;
  horse_color_key: string | null;
  created_by: string;
  creator_name: string;
  /** YYYY-MM-DD, stable-local. */
  date: string;
  /** Inclusive last day, or null for a single day. */
  date_end: string | null;
  /** HH:MM. */
  time_from: string | null;
  time_to: string | null;
  location: string;
  description: string;
  tasks: string[];
  helpers_needed: number;
  status: RequestStatus;
  recurring_rule: string | null;
  series_id: string | null;
  /** The viewer's own reminder as helper; null when off or not a helper. */
  my_remind_at: string | null;
  payload: Record<string, unknown>;
  created_at: string;
  helpers: RequestHelper[];
  helpers_count: number;
  spots_left: number;
  is_creator: boolean;
  is_helper: boolean;
  can_accept: boolean;
};

export type ShowClass = { name: string; time?: string };

export type ShowHelperPayload = {
  show_name: string;
  classes: ShowClass[];
  tasks: string[];
  ride_along: boolean;
};

export type RideSharePayload = { destination: string; departure_time: string; seats_free: number };

export type ExercisePayload = { mode: "lunge" | "ride"; rules_note?: string };

export type FeedOrTurnoutPayload = { what: "feed" | "turnout" | "bring_in" };

export type AppointmentPayload = { with: "farrier" | "vet" | "other"; note?: string };

// --- types --------------------------------------------------------------------------------

export type RequestTypeMeta = {
  type: RequestType;
  /** German name of the type. */
  label: string;
  /** Lucide icon name; mapped to the component in components/request-type-icon.tsx. */
  icon: string;
  /** German text of the accept button. */
  acceptLabel: string;
  /** One line shown on the type tile of the new request screen. */
  hint: string;
};

export const REQUEST_TYPES: readonly RequestTypeMeta[] = [
  { type: "show_helper", label: "Turniertrottel", icon: "Trophy", acceptLabel: "Ich komme mit", hint: "Hilfe auf dem Turnier" },
  { type: "ride_share", label: "Mitfahrgelegenheit", icon: "Truck", acceptLabel: "Ich komme mit", hint: "Platz im Hänger" },
  { type: "exercise", label: "Bewegen", icon: "Footprints", acceptLabel: "Mach ich", hint: "Longieren oder Reiten" },
  { type: "feed_or_turnout", label: "Füttern und Rausstellen", icon: "Wheat", acceptLabel: "Mach ich", hint: "Füttern, raus, rein" },
  { type: "appointment_companion", label: "Terminbegleitung", icon: "Stethoscope", acceptLabel: "Mach ich", hint: "Hufschmied oder Tierarzt" },
  { type: "other", label: "Sonstiges", icon: "HandHelping", acceptLabel: "Mach ich", hint: "Alles andere" },
  { type: "blanket", label: "Decken", icon: "Shirt", acceptLabel: "Mach ich", hint: "Decke auf- oder abziehen" },
];

/** Types the user can pick in the new request form (blanket requests are created from the blanket tab). */
export const CREATABLE_TYPES: readonly RequestTypeMeta[] = REQUEST_TYPES.filter((t) => t.type !== "blanket");

const OTHER_META = REQUEST_TYPES.find((t) => t.type === "other") as RequestTypeMeta;

export function isRequestType(value: unknown): value is RequestType {
  return typeof value === "string" && REQUEST_TYPES.some((t) => t.type === value);
}

/** Metadata of a type; unknown values fall back to "other". */
export function typeMeta(type: string | null | undefined): RequestTypeMeta {
  return REQUEST_TYPES.find((t) => t.type === type) ?? OTHER_META;
}

// --- show helper tasks ----------------------------------------------------------------------

export const SHOW_TASKS: readonly { value: string; label: string }[] = [
  { value: "hold_horse", label: "Pferd halten" },
  { value: "warm_up", label: "Abreiten" },
  { value: "film", label: "Filmen" },
  { value: "fetch_number", label: "Startnummer holen" },
  { value: "load_trailer", label: "Hänger beladen" },
];

/** German label of a task chip: show helper tasks are keys, all other tasks are free text. */
export function taskLabel(type: RequestType, task: string): string {
  if (type === "show_helper") return SHOW_TASKS.find((t) => t.value === task)?.label ?? task;
  return task;
}

// --- status ---------------------------------------------------------------------------------

export type BadgeVariant = "neutral" | "primary" | "accent" | "info" | "danger";

export function statusBadge(status: RequestStatus): { label: string; variant: BadgeVariant } {
  switch (status) {
    case "open":
      return { label: "Offen", variant: "accent" };
    case "assigned":
      return { label: "Vergeben", variant: "primary" };
    case "done":
      return { label: "Erledigt", variant: "neutral" };
    case "cancelled":
      return { label: "Abgesagt", variant: "danger" };
  }
}

// --- helpers --------------------------------------------------------------------------------

/** "1 von 2 Helfern", "0 von 1 Helfer"; ride shares count seats: "1 von 2 Plätzen". */
export function helperCountText(count: number, needed: number, type?: RequestType): string {
  if (type === "ride_share") return `${count} von ${needed} ${needed === 1 ? "Platz" : "Plätzen"}`;
  return `${count} von ${needed} ${needed === 1 ? "Helfer" : "Helfern"}`;
}

/** Longer status sentence for the detail screen. */
export function helperStatusText(count: number, needed: number, type?: RequestType): string {
  const left = needed - count;
  if (left <= 0) return type === "ride_share" ? "Alle Plätze belegt" : "Alle Helfer gefunden";
  if (type === "ride_share") return left === 1 ? "Noch 1 Platz frei" : `Noch ${left} Plätze frei`;
  return left === 1 ? "Noch 1 Helfer gesucht" : `Noch ${left} Helfer gesucht`;
}

export function acceptLabel(type: RequestType): string {
  return typeMeta(type).acceptLabel;
}

// --- dates ----------------------------------------------------------------------------------

const WEEKDAYS_SHORT = ["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"];
const WEEKDAYS_LONG = ["Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"];
const MONTHS_SHORT = ["Jan", "Feb", "Mär", "Apr", "Mai", "Jun", "Jul", "Aug", "Sep", "Okt", "Nov", "Dez"];

/** Parses YYYY-MM-DD into its parts without any time zone involved. Null if malformed. */
export function parseDate(date: string): { year: number; month: number; day: number; weekday: number } | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date);
  if (!m) return null;
  const [year, month, day] = [Number(m[1]), Number(m[2]), Number(m[3])];
  const utc = new Date(Date.UTC(year, month - 1, day));
  if (utc.getUTCFullYear() !== year || utc.getUTCMonth() !== month - 1 || utc.getUTCDate() !== day) return null;
  return { year, month, day, weekday: utc.getUTCDay() };
}

/** "Mi, 2. Okt". */
export function formatDay(date: string): string {
  const p = parseDate(date);
  if (!p) return date;
  return `${WEEKDAYS_SHORT[p.weekday]}, ${p.day}. ${MONTHS_SHORT[p.month - 1]}`;
}

export type When = {
  date: string;
  date_end?: string | null;
  time_from?: string | null;
  time_to?: string | null;
};

/**
 * "Mi, 2. Okt · 18:00" (with end time "Mi, 2. Okt · 18:00–19:30"); date ranges read
 * "Mo, 5. Okt – Mi, 7. Okt" and keep the start time: "Mo, 5. Okt – Mi, 7. Okt · 08:00".
 */
export function formatWhen(w: When): string {
  let text = formatDay(w.date);
  if (w.date_end && w.date_end !== w.date) {
    text += ` – ${formatDay(w.date_end)}`;
    return w.time_from ? `${text} · ${w.time_from}` : text;
  }
  if (w.time_from) {
    text += ` · ${w.time_from}`;
    if (w.time_to) text += `–${w.time_to}`;
  }
  return text;
}

/** Local calendar day as YYYY-MM-DD. */
export function toIsoDate(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Adds days to a YYYY-MM-DD date (DST safe). */
export function addDays(date: string, days: number): string {
  const p = parseDate(date);
  if (!p) return date;
  const d = new Date(Date.UTC(p.year, p.month - 1, p.day + days));
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}-${String(d.getUTCDate()).padStart(2, "0")}`;
}

/** "Heute", "Morgen" or the weekday and date. */
export function relativeDay(date: string, today: string): string {
  if (date === today) return "Heute";
  if (date === addDays(today, 1)) return "Morgen";
  return formatDay(date);
}

/** Accepts "18:30", "1830", "18.30", "8", "8:5"; returns "HH:MM" or null. */
export function normalizeTime(input: string): string | null {
  const s = input.trim().replace(".", ":");
  let h: string;
  let m: string;
  if (/^\d{1,2}$/.test(s)) {
    h = s;
    m = "0";
  } else if (/^\d{3,4}$/.test(s)) {
    h = s.slice(0, s.length - 2);
    m = s.slice(-2);
  } else {
    const parts = /^(\d{1,2}):(\d{1,2})$/.exec(s);
    if (!parts) return null;
    h = parts[1];
    m = parts[2];
  }
  const hh = Number(h);
  const mm = Number(m);
  if (hh > 23 || mm > 59) return null;
  return `${String(hh).padStart(2, "0")}:${String(mm).padStart(2, "0")}`;
}

/** Start and end of a request as local Date objects (all-day when there is no start time). */
export function eventRange(w: When): { start: Date; end: Date; allDay: boolean } | null {
  const first = parseDate(w.date);
  const last = parseDate(w.date_end ?? w.date);
  if (!first || !last) return null;
  if (!w.time_from) {
    return {
      start: new Date(first.year, first.month - 1, first.day),
      end: new Date(last.year, last.month - 1, last.day + 1),
      allDay: true,
    };
  }
  const [fh, fm] = w.time_from.split(":").map(Number);
  const start = new Date(first.year, first.month - 1, first.day, fh, fm);
  let end = new Date(start.getTime() + 60 * 60 * 1000);
  if (w.time_to) {
    const [th, tm] = w.time_to.split(":").map(Number);
    end = new Date(last.year, last.month - 1, last.day, th, tm);
  }
  return { start, end, allDay: false };
}

// --- helper reminder ------------------------------------------------------------------------

export type ReminderOption = "default" | "morning" | "two_hours" | "off";

export const REMINDER_OPTIONS: readonly { value: ReminderOption; label: string }[] = [
  { value: "default", label: "Vortag 18:00" },
  { value: "morning", label: "Am Morgen 07:00" },
  { value: "two_hours", label: "2 Stunden vorher" },
  { value: "off", label: "Keine" },
];

/**
 * RFC 3339 instant of a reminder option (device time zone), or null when there is none:
 * "off", an unparsable date, or "two_hours" without a start time.
 */
export function reminderAt(option: ReminderOption, date: string, timeFrom: string | null): string | null {
  const p = parseDate(date);
  if (!p || option === "off") return null;
  if (option === "default") return new Date(p.year, p.month - 1, p.day - 1, 18, 0).toISOString();
  if (option === "morning") return new Date(p.year, p.month - 1, p.day, 7, 0).toISOString();
  if (!timeFrom) return null;
  const [h, m] = timeFrom.split(":").map(Number);
  return new Date(p.year, p.month - 1, p.day, h - 2, m).toISOString();
}

/** The options a helper can still pick: their time lies ahead ("off" always does). */
export function reminderChoices(date: string, timeFrom: string | null, now: Date = new Date()) {
  return REMINDER_OPTIONS.filter((o) => {
    const at = reminderAt(o.value, date, timeFrom);
    return o.value === "off" || (at !== null && new Date(at) > now);
  });
}

/** Which option matches the stored reminder, or null for a custom time. */
export function currentReminder(mine: string | null, date: string, timeFrom: string | null): ReminderOption | null {
  if (!mine) return "off";
  const t = new Date(mine).getTime();
  return REMINDER_OPTIONS.find((o) => {
    const at = reminderAt(o.value, date, timeFrom);
    return at !== null && new Date(at).getTime() === t;
  })?.value ?? null;
}

// --- recurrence -----------------------------------------------------------------------------

const RRULE_DAYS = ["SU", "MO", "TU", "WE", "TH", "FR", "SA"];
const ACCUSATIVE_DAYS = ["Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"];

export type Recurrence = "none" | "daily" | "weekly";

/** The RRULE the API accepts for the simple choices of the form. "" means no recurrence. */
export function buildRule(kind: Recurrence, date: string): string {
  if (kind === "daily") return "FREQ=DAILY";
  if (kind === "weekly") {
    const p = parseDate(date);
    return p ? `FREQ=WEEKLY;BYDAY=${RRULE_DAYS[p.weekday]}` : "";
  }
  return "";
}

/**
 * German summary of a recurring_rule: "täglich", "jeden Mittwoch",
 * "jeden Montag und Freitag", "jeden Montag, Mittwoch und Freitag", optionally
 * followed by " bis 31. Dez". `startDate` names the weekday of a weekly rule without BYDAY.
 */
export function describeRule(rule: string | null | undefined, startDate?: string): string {
  if (!rule) return "";
  const parts = new Map<string, string>();
  for (const part of rule.replace(/^RRULE:/i, "").split(";")) {
    const [k, v] = part.split("=");
    if (k && v) parts.set(k.trim().toUpperCase(), v.trim().toUpperCase());
  }
  const freq = parts.get("FREQ");
  let text: string;
  if (freq === "DAILY") {
    text = "täglich";
  } else if (freq === "WEEKLY") {
    let days = (parts.get("BYDAY") ?? "").split(",").filter(Boolean);
    if (days.length === 0 && startDate) {
      const p = parseDate(startDate);
      if (p) days = [RRULE_DAYS[p.weekday]];
    }
    const names = days
      .map((d) => RRULE_DAYS.indexOf(d))
      .filter((i) => i >= 0)
      .sort((a, b) => ((a + 6) % 7) - ((b + 6) % 7))
      .map((i) => ACCUSATIVE_DAYS[i]);
    if (names.length === 0) text = "jede Woche";
    else if (names.length === 1) text = `jeden ${names[0]}`;
    else text = `jeden ${names.slice(0, -1).join(", ")} und ${names[names.length - 1]}`;
  } else {
    return "";
  }
  const until = parts.get("UNTIL");
  if (until && /^\d{8}/.test(until)) {
    const day = formatDay(`${until.slice(0, 4)}-${until.slice(4, 6)}-${until.slice(6, 8)}`);
    text += ` bis ${day.slice(day.indexOf(",") + 2)}`;
  }
  return text;
}

// --- list filters and queries ---------------------------------------------------------------

export type RequestFilter = "open" | "mine" | "helping" | "done";

export const REQUEST_FILTERS: readonly { value: RequestFilter; label: string }[] = [
  { value: "open", label: "Offen" },
  { value: "mine", label: "Meine" },
  { value: "helping", label: "Ich helfe" },
  { value: "done", label: "Erledigt" },
];

export type ListParams = {
  status?: string;
  type?: string;
  mine?: boolean;
  assigned?: boolean;
  horse_id?: string;
  from?: string;
  to?: string;
  limit?: number;
};

/** Query parameters of a filter pill; `today` is the local YYYY-MM-DD. */
export function filterParams(filter: RequestFilter, today: string): ListParams {
  switch (filter) {
    case "open":
      return { status: "open", from: today };
    case "mine":
      return { mine: true, status: "open,assigned,done" };
    case "helping":
      return { assigned: true, status: "open,assigned", from: today };
    case "done":
      return { status: "done", limit: 50 };
  }
}

/** "?a=1&b=x" (empty string without parameters); undefined and false values are skipped. */
export function toQuery(params: Record<string, string | number | boolean | undefined>): string {
  const pairs: string[] = [];
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === false || v === "") continue;
    pairs.push(`${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`);
  }
  return pairs.length ? `?${pairs.join("&")}` : "";
}

/** Headline such as "Turniertrottel · Herbstturnier" or "Bewegen: Luna". */
export function requestTitle(r: Pick<HelpRequest, "type" | "horse_name" | "payload">): string {
  let title = typeMeta(r.type).label;
  if (r.type === "show_helper" && typeof r.payload?.show_name === "string" && r.payload.show_name) {
    title += ` · ${r.payload.show_name}`;
  }
  if (r.horse_name) title += `: ${r.horse_name}`;
  return title;
}

/** Task chips of a request in German. */
export function taskChips(r: Pick<HelpRequest, "type" | "tasks">): string[] {
  return r.tasks.map((t) => taskLabel(r.type, t));
}

/** Weekday name for group headers, e.g. "Mittwoch". */
export function weekdayName(date: string): string {
  const p = parseDate(date);
  return p ? WEEKDAYS_LONG[p.weekday] : date;
}

// --- details --------------------------------------------------------------------------------

/** "Do, 1. Okt · 18:00" for an RFC 3339 instant, in the device time zone. */
export function formatInstant(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${formatDay(toIsoDate(d))} · ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const EXERCISE_MODES: Record<string, string> = { lunge: "Longieren", ride: "Reiten" };
const FEED_WHAT: Record<string, string> = { feed: "Füttern", turnout: "Rausstellen", bring_in: "Reinholen" };
const APPOINTMENT_WITH: Record<string, string> = { farrier: "Hufschmied", vet: "Tierarzt", other: "Sonstiges" };

/** Label/value rows for the type-specific payload of a request (German). */
export function payloadDetails(r: Pick<HelpRequest, "type" | "payload">): { label: string; value: string }[] {
  const p = (r.payload ?? {}) as Record<string, unknown>;
  const rows: { label: string; value: string }[] = [];
  const str = (v: unknown) => (typeof v === "string" ? v : "");
  switch (r.type) {
    case "show_helper": {
      if (str(p.show_name)) rows.push({ label: "Turnier", value: str(p.show_name) });
      const classes = Array.isArray(p.classes) ? (p.classes as ShowClass[]) : [];
      if (classes.length > 0) {
        rows.push({ label: "Prüfungen", value: classes.map((c) => (c.time ? `${c.name} (${c.time})` : c.name)).join(", ") });
      }
      if (p.ride_along === true) rows.push({ label: "Mitfahrgelegenheit", value: "Platz im Hänger" });
      break;
    }
    case "ride_share":
      if (str(p.destination)) rows.push({ label: "Ziel", value: str(p.destination) });
      if (str(p.departure_time)) rows.push({ label: "Abfahrt", value: str(p.departure_time) });
      break;
    case "exercise":
      if (EXERCISE_MODES[str(p.mode)]) rows.push({ label: "Wie", value: EXERCISE_MODES[str(p.mode)] });
      if (str(p.rules_note)) rows.push({ label: "Regeln", value: str(p.rules_note) });
      break;
    case "feed_or_turnout":
      if (FEED_WHAT[str(p.what)]) rows.push({ label: "Aufgabe", value: FEED_WHAT[str(p.what)] });
      break;
    case "appointment_companion":
      if (APPOINTMENT_WITH[str(p.with)]) rows.push({ label: "Termin bei", value: APPOINTMENT_WITH[str(p.with)] });
      if (str(p.note)) rows.push({ label: "Hinweis", value: str(p.note) });
      break;
    default:
      break;
  }
  return rows;
}
