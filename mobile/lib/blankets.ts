// Pure helpers for the blanket feature (M3): wire types, German labels, night line,
// progress and rule texts. No React Native imports, so this file is unit-tested with
// node --test. The wire types mirror docs/domains/blankets.md.

import { ApiError, errorMessage } from "./api-core.ts";
import { formatClock } from "./presence-format.ts";
import { formatMinutes, minutesOf } from "./reminders.ts";

export type StateAction = "covered" | "uncovered" | "checked";

export type Blanket = {
  id: string;
  horse_id: string;
  name: string;
  fill_g: number;
  color: string | null;
  location: string | null;
  photo_path: string | null;
  /** Relative API path ("/api/v1/files/..."); load it with fileSource(). */
  photo_url: string | null;
};

export type Rule = {
  id: string;
  /** 1-based priority; the first matching rule wins. */
  position: number;
  temp_min: number | null;
  temp_max: number | null;
  rain: boolean | null;
  blanket_id: string | null;
  note: string;
};

/** One hourly step of the forecast; rain values are those of the hour that ends at `time`. */
export type WeatherHour = {
  time: string;
  temp_c: number;
  rain_mm: number;
  rain_prob: number;
  wind_kmh: number;
};

/**
 * Forecast for the time the horse is covered: from the evening until the next morning (the
 * stable's cover window, 18:00 to 12:30 by default). `window_start` is the zero time and `timeline` empty for snapshots stored before
 * the details existed (`hasWeatherDetails`).
 */
export type Weather = {
  /** Lowest temperature in the window (name kept from the wire format). */
  night_min_c: number;
  will_rain: boolean;
  /** Highest precipitation probability in the window, percent. */
  rain_probability: number;
  /** Total precipitation in the window, mm. */
  rain_mm: number;
  /** Highest mean wind speed in the window, km/h. */
  wind_kmh: number;
  fetched_at: string;
  window_start: string;
  window_end: string;
  temp_max_c: number;
  /** Most rain in one hour, mm. */
  rain_peak_mm: number;
  rain_hours: number;
  /** Start of the first and end of the last rainy hour, null when dry. */
  rain_from: string | null;
  rain_until: string | null;
  timeline: WeatherHour[];
};

export type RecommendationStatus = "blanket" | "none" | "no_rule" | "no_weather";

export type Recommendation = {
  status: RecommendationStatus;
  /** 0-based index into the rules list, null when no rule matched. */
  rule_index: number | null;
  blanket: Blanket | null;
  /** The owner's wish of the matched rule, "" when there is none. */
  note: string;
};

export type BlanketState = {
  id: string;
  horse_id: string;
  /** The blanket day (YYYY-MM-DD, the evening the night starts). */
  day: string;
  action: StateAction;
  covered_with: string | null;
  covered_with_name: string | null;
  changed_at: string;
  changed_by: { id: string; name: string } | null;
  /** True when the auto-uncover job wrote the state (no author, the farm staff did it). */
  automatic: boolean;
};

export type HorseRef = { id: string; name: string; box: string | null; color_key: string | null };

export type TodayHorse = {
  horse: HorseRef;
  recommendation: Recommendation;
  state: BlanketState | null;
  done: boolean;
  is_mine: boolean;
};

export type Progress = { done: number; total: number };

export type Today = {
  day: string;
  weather: Weather | null;
  /** HH:MM, stable-local. */
  reminder_time: string;
  progress: Progress;
  horses: TodayHorse[];
};

export type Plan = {
  horse: HorseRef;
  day: string;
  weather: Weather | null;
  recommendation: Recommendation;
  rules: Rule[];
  blankets: Blanket[];
  helper_note: string | null;
  state: BlanketState | null;
  can_manage: boolean;
  /** The horse's cover window ("HH:MM", Deckenzeitraum); the weather is summarised over it. */
  cover_start: string;
  cover_end: string;
};

// --- texts ---------------------------------------------------------------------------------

const ACTION_LABELS: Record<StateAction, string> = {
  covered: "Eingedeckt",
  uncovered: "Abgedeckt",
  checked: "Geprüft",
};

export function actionLabel(action: StateAction): string {
  return ACTION_LABELS[action];
}

/** German decimal comma, at most one decimal: 1.5 -> "1,5", 3 -> "3". */
function num(v: number): string {
  return (Math.round(v * 10) / 10).toString().replace(".", ",");
}

/** Rain below this (mm in the whole window) is drizzle at most and shown as "kaum Regen". */
const NEGLIGIBLE_RAIN_MM = 0.5;

/** Word for the amount of rain: "trocken", "leichter Regen", "mäßiger Regen", "starker Regen". */
export function rainIntensity(mm: number): string {
  if (mm < 0.1) return "trocken";
  if (mm < NEGLIGIBLE_RAIN_MM) return "kaum Regen";
  if (mm < 2) return "leichter Regen";
  if (mm < 8) return "mäßiger Regen";
  return "starker Regen";
}

/** "trocken" or "4,8 mm Regen" ("kaum Regen" below 0.5 mm). */
export function rainAmount(weather: Pick<Weather, "rain_mm">): string {
  const mm = weather.rain_mm;
  if (mm < 0.1) return "trocken";
  if (mm < NEGLIGIBLE_RAIN_MM) return "kaum Regen";
  return `${num(mm)} mm Regen`;
}

/** "3 °C" when min and max round to the same value, else "3 bis 9 °C". */
export function temperatureRange(weather: Pick<Weather, "night_min_c" | "temp_max_c">): string {
  const lo = Math.round(weather.night_min_c);
  const hi = Math.round(weather.temp_max_c);
  return hi > lo ? `${lo} bis ${hi} °C` : `${lo} °C`;
}

/** False for snapshots without the detail fields (stored before they existed). */
export function hasWeatherDetails(weather: Weather): boolean {
  return weather.timeline.length > 0;
}

/** "18:00 bis 12:30 Uhr" (the cover window) in the stable's time zone. */
export function windowLabel(weather: Weather, timeZone: string): string {
  if (!hasWeatherDetails(weather)) return "über Nacht";
  return `${formatClock(weather.window_start, timeZone)} bis ${formatClock(weather.window_end, timeZone)} Uhr`;
}

/** "22:00 bis 12:30 Uhr" / "trocken": when it rains within the window. */
export function rainTiming(weather: Weather, timeZone: string): string {
  if (!weather.rain_from || !weather.rain_until) return "trocken";
  const from = formatClock(weather.rain_from, timeZone);
  const until = formatClock(weather.rain_until, timeZone);
  return `${from} bis ${until} Uhr`;
}

/** "3 bis 9 °C, 4,8 mm Regen" for the given weather. */
function nightSummary(weather: Weather): string {
  return `${temperatureRange(weather)}, ${rainAmount(weather)}`;
}

/** "Heute Nacht 3 bis 9 °C, 4,8 mm Regen" / "Heute Nacht 8 °C, trocken" / text without a forecast. */
export function nightLine(weather: Weather | null): string {
  if (!weather) return "Noch keine Vorhersage";
  return `Heute Nacht ${nightSummary(weather)}`;
}

/** Height of a rain bar as fraction 0..1; 4 mm per hour and more is a full bar. */
export function weatherBarFraction(mm: number): number {
  return Math.min(1, Math.max(0, mm) / 4);
}

/** Facts of the weather card: label and value, rain first. Only facts that are known. */
export function weatherFacts(weather: Weather, timeZone: string): { label: string; value: string }[] {
  const rain = weather.rain_mm >= 0.1;
  const facts = [
    { label: "Temperatur", value: temperatureRange(weather) },
    {
      label: "Regen",
      value: rain ? `${num(weather.rain_mm)} mm (${rainIntensity(weather.rain_mm)})` : "trocken",
    },
    { label: "Regenwahrscheinlichkeit", value: `bis ${weather.rain_probability} %` },
    { label: "Wind", value: `bis ${Math.round(weather.wind_kmh)} km/h` },
  ];
  if (rain && weather.rain_from) {
    facts.splice(2, 0, { label: "Regenzeit", value: rainTiming(weather, timeZone) });
    if (weather.rain_peak_mm >= 0.5) {
      facts.splice(3, 0, { label: "Stärkster Regen", value: `${num(weather.rain_peak_mm)} mm pro Stunde` });
    }
  }
  return facts;
}

/** Unit line under the start hero: "heute Nacht, 4,8 mm Regen" / "heute Nacht, trocken". */
export function nightUnit(weather: Weather | null): string {
  return weather ? `heute Nacht, ${rainAmount(weather)}` : "heute Nacht";
}

/** Big number for the start hero: "3°". */
export function nightTempShort(weather: Weather | null): string {
  return weather ? `${Math.round(weather.night_min_c)}°` : "–";
}

/** "4/7" */
export function progressLabel(p: Progress): string {
  return `${p.done}/${p.total}`;
}

/** Fraction 0..1 for a progress bar; 1 when there are no horses. */
export function progressFraction(p: Progress): number {
  return p.total === 0 ? 1 : Math.min(1, p.done / p.total);
}

/** "Noch 3 Pferde offen" / "Alle versorgt". */
export function progressText(p: Progress): string {
  const open = p.total - p.done;
  if (p.total === 0) return "Noch keine Pferde";
  if (open <= 0) return "Alle versorgt";
  return open === 1 ? "Noch 1 Pferd offen" : `Noch ${open} Pferde offen`;
}

/** "Erinnerung um 20:30" */
export function reminderHint(time: string): string {
  return `Erinnerung um ${time}`;
}

/** Title of a recommendation: "Decke 100 g", "Keine Decke", ... */
export function recommendationTitle(rec: Recommendation): string {
  switch (rec.status) {
    case "blanket":
      return rec.blanket?.name ?? "Decke";
    case "none":
      return "Keine Decke";
    case "no_rule":
      return "Keine passende Regel";
    case "no_weather":
      return "Noch keine Empfehlung";
  }
}

/** Second line: where the blanket hangs, or why there is no recommendation. */
export function recommendationDetail(rec: Recommendation): string {
  switch (rec.status) {
    case "blanket":
      return rec.blanket?.location ? `Ort: ${rec.blanket.location}` : "Ort nicht eingetragen";
    case "none":
      return "Laut Deckenplan";
    case "no_rule":
      return "Für diese Vorhersage fehlt eine Regel";
    case "no_weather":
      return "Vorhersage fehlt noch";
  }
}

/** Buttons to show for a horse, depending on what the plan recommends. */
export function stateActions(rec: Recommendation): StateAction[] {
  switch (rec.status) {
    case "blanket":
      return ["covered", "uncovered"];
    case "none":
      return ["checked"];
    default:
      return ["covered", "uncovered", "checked"];
  }
}

/** "Eingedeckt mit Decke 100 g" / "Abgedeckt" / "Abgedeckt (Hof, automatisch)" / "Geprüft". */
export function stateLabel(state: BlanketState): string {
  if (state.automatic && state.action === "uncovered") return "Abgedeckt (Hof, automatisch)";
  if (state.action === "covered" && state.covered_with_name) return `Eingedeckt mit ${state.covered_with_name}`;
  return actionLabel(state.action);
}

/** "Mia, 19:04", "Hof, 12:30" for automatic states (time in the stable's time zone). */
export function stateByline(state: BlanketState, timeZone: string): string {
  const time = formatClock(state.changed_at, timeZone);
  const who = state.changed_by?.name ?? (state.automatic ? "Hof" : "");
  return [who, time].filter(Boolean).join(", ");
}

/** Filling as text: "Füllung 100 g" or "ohne Füllung". */
export function fillLabel(fillG: number): string {
  return fillG > 0 ? `Füllung ${fillG} g` : "ohne Füllung";
}

/** Condition of a rule in words: "0 bis unter 5 °C, bei Regen". */
export function ruleCondition(rule: Pick<Rule, "temp_min" | "temp_max" | "rain">): string {
  const { temp_min: min, temp_max: max } = rule;
  let t: string;
  if (min !== null && max !== null) t = `${min} bis unter ${max} °C`;
  else if (min !== null) t = `ab ${min} °C`;
  else if (max !== null) t = `unter ${max} °C`;
  else t = "Jede Temperatur";
  if (rule.rain === true) return `${t}, bei Regen`;
  if (rule.rain === false) return `${t}, ohne Regen`;
  return t;
}

/** Name of the blanket a rule names, or "Keine Decke". */
export function ruleBlanketName(rule: Pick<Rule, "blanket_id">, blankets: readonly Blanket[]): string {
  if (!rule.blanket_id) return "Keine Decke";
  return blankets.find((b) => b.id === rule.blanket_id)?.name ?? "Keine Decke";
}

// --- dates -----------------------------------------------------------------------------------

function dayNumber(day: string): number | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(day);
  if (!m) return null;
  return Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])) / 86_400_000;
}

/** "Heute Nacht", "Gestern Nacht" or "Nacht auf den 28.09." for a blanket day (the night starts that evening). */
export function historyDayLabel(day: string, today: string): string {
  const d = dayNumber(day);
  const t = dayNumber(today);
  if (d === null || t === null) return day;
  if (d === t) return "Heute Nacht";
  if (d === t - 1) return "Gestern Nacht";
  return `Nacht vom ${day.slice(8, 10)}.${day.slice(5, 7)}.`;
}

/** Number of nights in a history list where the horse was blanketed (action covered). */
export function coveredNights(states: readonly BlanketState[]): number {
  return new Set(states.filter((s) => s.action === "covered").map((s) => s.day)).size;
}

// --- errors ----------------------------------------------------------------------------------

/** German text for a failed blanket call. */
export function blanketErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 403) return "Nur Besitzerin, Besitzer oder Admins dürfen das ändern.";
    if (err.code === "in_use") return "Die Decke wird in einer Regel verwendet.";
    if (err.code === "file_too_large") return "Foto zu groß (max. 20 MB).";
    if (err.code === "validation_failed" && /cover_/.test(err.message)) {
      return "Beginn zwischen 15:00 und 23:00 Uhr, Ende zwischen 04:00 und 15:00 Uhr wählen.";
    }
  }
  return errorMessage(err);
}

// --- cover window --------------------------------------------------------------------------

export const COVER_START_MIN = "15:00";
export const COVER_START_MAX = "23:00";
export const COVER_END_MIN = "04:00";
export const COVER_END_MAX = "15:00";
const COVER_STEP_MINUTES = 15;

export type CoverWindowField = "start" | "end";

const COVER_LIMITS: Record<CoverWindowField, [string, string]> = {
  start: [COVER_START_MIN, COVER_START_MAX],
  end: [COVER_END_MIN, COVER_END_MAX],
};

/** Moves one end of the cover window by `steps` quarter hours and keeps it inside the allowed range. */
export function stepCoverTime(field: CoverWindowField, hhmm: string, steps: number): string {
  const [min, max] = COVER_LIMITS[field].map((t) => minutesOf(t)!) as [number, number];
  const step = COVER_STEP_MINUTES;
  const current = minutesOf(hhmm) ?? min;
  let next: number;
  if (current % step === 0) next = current + steps * step;
  else if (steps > 0) next = Math.ceil(current / step) * step + (steps - 1) * step;
  else next = Math.floor(current / step) * step + (steps + 1) * step;
  return formatMinutes(Math.min(max, Math.max(min, next)));
}

// --- start screen ----------------------------------------------------------------------------

export function firstName(name: string): string {
  return name.trim().split(/\s+/)[0] ?? "";
}

/** "Guten Morgen, Jan" depending on the local hour. */
export function greeting(hour: number, name: string): string {
  const word = hour < 11 ? "Guten Morgen" : hour < 18 ? "Guten Tag" : "Guten Abend";
  const who = firstName(name);
  return who ? `${word}, ${who}` : word;
}

/** One line per own horse: "Luna: Decke 100 g". Horses without a recommendation are skipped. */
export function myBlanketLines(horses: readonly TodayHorse[], mineIds: ReadonlySet<string>): string[] {
  return horses
    .filter((h) => mineIds.has(h.horse.id) && h.recommendation.status !== "no_weather")
    .map((h) => `${h.horse.name}: ${recommendationTitle(h.recommendation)}`);
}

/** Description of the weather hero on the start screen. */
export function startWeatherText(lines: readonly string[], hasWeather: boolean, hasHorses: boolean): string {
  if (!hasWeather) return "Noch keine Vorhersage.";
  if (!hasHorses) return "Keine eigenen Pferde. Alle findest du unter Decken.";
  if (lines.length === 0) return "Noch keine Empfehlung.";
  return lines.join("\n");
}
