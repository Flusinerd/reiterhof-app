// Pure helpers for the official DWD weather warnings (backend: internal/weather, GET /weather/warnings).
// No React Native imports, so they can be unit-tested with node --test. UI strings are German.

export type WarningSeverity = "minor" | "moderate" | "severe" | "extreme";

export type WeatherWarning = {
  id: string;
  /** DWD event name in capitals, e.g. "STURMBÖEN". */
  event: string;
  severity: WarningSeverity;
  /** e.g. "Amtliche WARNUNG vor STURMBÖEN". */
  headline: string;
  description: string;
  instruction: string;
  /** ISO instants; expires null = until further notice. */
  onset: string;
  expires: string | null;
  fetched_at: string;
};

export type WarningsResponse = { warnings: WeatherWarning[] };

const SEVERITY_LABELS: Record<WarningSeverity, string> = {
  minor: "Wetterwarnung",
  moderate: "Markante Wetterwarnung",
  severe: "Unwetterwarnung",
  extreme: "Extremes Unwetter",
};

/** German name of the DWD warning level. */
export function severityLabel(severity: string): string {
  return SEVERITY_LABELS[severity as WarningSeverity] ?? "Wetterwarnung";
}

/** Color tone of the level: yellow/orange/red/violet at the DWD map onto info, accent and danger. */
export function severityTone(severity: string): "info" | "accent" | "danger" {
  if (severity === "severe" || severity === "extreme") return "danger";
  if (severity === "moderate") return "accent";
  return "info";
}

const WEEKDAYS = ["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"];

function parts(iso: string, timeZone: string): { day: string; date: string; clock: string; key: string } | null {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  const p = Object.fromEntries(
    new Intl.DateTimeFormat("en-GB", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
      weekday: "short",
      timeZone,
    })
      .formatToParts(d)
      .map((x) => [x.type, x.value]),
  );
  const wd = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].indexOf(p.weekday);
  return {
    day: WEEKDAYS[wd] ?? "",
    date: `${p.day}.${p.month}.`,
    clock: `${p.hour}:${p.minute}`,
    key: `${p.year}-${p.month}-${p.day}`,
  };
}

/**
 * Time range of a warning in the stable's time zone, as in the push:
 * "Sa 24.10., 14:00 bis 22:00 Uhr", "Sa 24.10., 23:00 bis So 25.10., 08:00 Uhr", "ab Sa 24.10., 14:00 Uhr".
 */
export function warningSpan(onset: string, expires: string | null, timeZone: string): string {
  const on = parts(onset, timeZone);
  if (!on) return "";
  const start = `${on.day} ${on.date}, ${on.clock}`;
  const ex = expires ? parts(expires, timeZone) : null;
  if (!ex) return `ab ${start} Uhr`;
  if (ex.key === on.key) return `${start} bis ${ex.clock} Uhr`;
  return `${start} bis ${ex.day} ${ex.date}, ${ex.clock} Uhr`;
}

/** Hero line for the start screen: "Unwetterwarnung: Amtliche ..." for the most severe warning, "" without warnings. */
export function warningSummary(warnings: WeatherWarning[]): string {
  if (warnings.length === 0) return "";
  const first = warnings[0];
  const more = warnings.length - 1;
  return `${severityLabel(first.severity)}: ${first.headline}${more > 0 ? ` (+${more} weitere)` : ""}`;
}
