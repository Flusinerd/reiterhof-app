// Pure helpers for the blanket feature (M3): wire types, German labels, night line,
// progress and rule texts. No React Native imports, so this file is unit-tested with
// node --test. The wire types mirror docs/domains/blankets.md.

import { ApiError, errorMessage } from "./api-core.ts";
import { formatClock } from "./presence-format.ts";

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

export type Weather = {
  night_min_c: number;
  will_rain: boolean;
  rain_probability: number;
  rain_mm: number;
  wind_kmh: number;
  fetched_at: string;
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

function temp(c: number): string {
  return `${Math.round(c)} °C`;
}

/** "Heute Nacht 3 °C, Regen" / "Heute Nacht 8 °C, trocken" / text without a forecast. */
export function nightLine(weather: Weather | null): string {
  if (!weather) return "Noch keine Wettervorhersage für heute Nacht";
  return `Heute Nacht ${temp(weather.night_min_c)}, ${weather.will_rain ? "Regen" : "trocken"}`;
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

/** "Noch 3 Pferde offen" / "Alle Pferde versorgt". */
export function progressText(p: Progress): string {
  const open = p.total - p.done;
  if (p.total === 0) return "Noch keine Pferde im Stall";
  if (open <= 0) return "Alle Pferde versorgt";
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
      return "Laut Deckenplan bleibt das Pferd ohne Decke";
    case "no_rule":
      return "Für diese Vorhersage gibt es keine Regel im Deckenplan";
    case "no_weather":
      return "Die Wettervorhersage fehlt noch";
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
    if (err.status === 403) return "Nur die Besitzerin, der Besitzer oder Admins dürfen den Deckenplan ändern.";
    if (err.code === "in_use") return "Die Decke wird von einer Regel verwendet. Ändere zuerst die Regel.";
    if (err.code === "file_too_large") return "Das Foto ist zu groß (höchstens 20 MB).";
  }
  return errorMessage(err);
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
  if (!hasWeather) return "Sobald die Vorhersage da ist, siehst du hier die Deckenempfehlung.";
  if (!hasHorses) return "Du hast keine eigenen Pferde oder Reitbeteiligungen. Alle Pferde findest du unter Decken.";
  if (lines.length === 0) return "Für deine Pferde gibt es noch keine Empfehlung.";
  return lines.join("\n");
}
