// Pure helpers for the presence screen (no React Native imports, unit-tested).
// All user-facing text is German.

export type Visibility = "all" | "only_day" | "hidden";

export const VISIBILITY_OPTIONS: readonly { value: Visibility; label: string }[] = [
  { value: "all", label: "Alle" },
  { value: "only_day", label: "Nur Tag" },
  { value: "hidden", label: "Versteckt" },
];

export function visibilityLabel(v: Visibility): string {
  return VISIBILITY_OPTIONS.find((o) => o.value === v)?.label ?? "Alle";
}

/** One sentence explaining what the other people see. */
export function visibilityDescription(v: Visibility): string {
  switch (v) {
    case "all":
      return "Alle im Stall sehen, dass und seit wann du da bist.";
    case "only_day":
      return "Andere sehen nur, dass du heute da bist oder an welchem Tag, aber keine Uhrzeiten.";
    case "hidden":
      return "Andere sehen dich nicht. Du selbst siehst weiterhin alles.";
  }
}

/** "18:40" in the given time zone (the stable's, e.g. Europe/Berlin). */
export function formatClock(iso: string, timeZone: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("de-DE", {
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone,
  }).format(date);
}

/** "seit 18:40", or "heute da" when the person hides the time (`since` is null). */
export function sinceLabel(since: string | null, timeZone: string): string {
  if (!since) return "heute da";
  const clock = formatClock(since, timeZone);
  return clock ? `seit ${clock}` : "heute da";
}

/** Calendar day (YYYY-MM-DD) of an instant in the given time zone. */
export function localDate(iso: string | Date, timeZone: string): string {
  const date = typeof iso === "string" ? new Date(iso) : iso;
  // en-CA formats as YYYY-MM-DD.
  return new Intl.DateTimeFormat("en-CA", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone,
  }).format(date);
}

/** Whole days from `from` to `to` (both YYYY-MM-DD). */
export function dayDiff(from: string, to: string): number {
  const a = Date.parse(`${from}T00:00:00Z`);
  const b = Date.parse(`${to}T00:00:00Z`);
  return Math.round((b - a) / 86_400_000);
}

const WEEKDAYS = ["Sonntag", "Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag"];

export type LastSeen = {
  /** YYYY-MM-DD in the stable's time zone. */
  last_seen_date: string;
  /** null when the person shows only the day. */
  last_seen_at: string | null;
};

/**
 * "zuletzt gesehen" text: "heute, 17:20", "gestern", "Montag, 17:20", "12.09.".
 * `today` is the current stable-local day (YYYY-MM-DD).
 */
export function lastSeenLabel(entry: LastSeen, today: string, timeZone: string): string {
  const diff = dayDiff(entry.last_seen_date, today);
  let day: string;
  if (diff <= 0) day = "heute";
  else if (diff === 1) day = "gestern";
  else if (diff < 7) day = WEEKDAYS[new Date(`${entry.last_seen_date}T12:00:00Z`).getUTCDay()];
  else {
    const [, m, d] = entry.last_seen_date.split("-");
    day = `${d}.${m}.`;
  }
  const clock = entry.last_seen_at ? formatClock(entry.last_seen_at, timeZone) : "";
  return clock ? `${day}, ${clock}` : day;
}

/** "kommt meist gegen 19 Uhr" or null when there is no hint. */
export function usualArrivalLabel(hour: number | null | undefined): string | null {
  if (hour === null || hour === undefined || !Number.isInteger(hour) || hour < 0 || hour > 23) return null;
  return `kommt meist gegen ${hour} Uhr`;
}
