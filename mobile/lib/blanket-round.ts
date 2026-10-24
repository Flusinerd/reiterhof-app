// The evening round ("Abendrunde", app/blankets/round.tsx): one horse per screen. Pure logic of
// the order, the progress segments, which horse comes next, the texts of a horse's screen and the
// summary at the end (docs/domains/blankets.md, "Evening round"). No React Native imports
// (unit-tested with node --test). German UI text.

import {
  actionLabel,
  rainAmount,
  recommendationTitle,
  stateByline,
  stateLabel,
  stateActions,
  temperatureRange,
  type Recommendation,
  type StateAction,
  type TodayHorse,
  type Weather,
} from "./blankets.ts";

/** Box names compare like people read them: "9" before "10", "A2" before "A10". */
function byBox(a: TodayHorse, b: TodayHorse): number {
  const ba = a.horse.box?.trim() ?? "";
  const bb = b.horse.box?.trim() ?? "";
  if (ba && !bb) return -1;
  if (!ba && bb) return 1;
  return (
    ba.localeCompare(bb, "de", { numeric: true, sensitivity: "base" }) ||
    a.horse.name.localeCompare(b.horse.name, "de") ||
    a.horse.id.localeCompare(b.horse.id)
  );
}

/**
 * The horses in the order of the walk: the user's own (owner or rider) first, then the others,
 * each group by box. Done horses stay in the list (so that one can go back to them); `nextIndex`
 * skips them.
 */
export function roundOrder(horses: readonly TodayHorse[], mineIds: ReadonlySet<string>): TodayHorse[] {
  const mine = horses.filter((h) => mineIds.has(h.horse.id)).sort(byBox);
  const others = horses.filter((h) => !mineIds.has(h.horse.id)).sort(byBox);
  return [...mine, ...others];
}

/** Index of the first horse that is not done yet, or null when all are. */
export function firstOpenIndex(order: readonly TodayHorse[]): number | null {
  const i = order.findIndex((h) => !h.done);
  return i < 0 ? null : i;
}

/**
 * The horse after `current` that is not done. Horses skipped with "Später" come around again at the
 * end: when nothing open is left behind `current`, the first open one before it comes. Null when
 * no other horse is open (the round is over; `current` itself does not count).
 */
export function nextIndex(order: readonly TodayHorse[], current: number): number | null {
  for (let i = current + 1; i < order.length; i++) if (!order[i]!.done) return i;
  for (let i = 0; i < current; i++) if (!order[i]!.done) return i;
  return null;
}

/** Previous horse in the list, done or not (going back to correct one); null at the start. */
export function previousIndex(current: number): number | null {
  return current > 0 ? current - 1 : null;
}

export type SegmentState = "done" | "current" | "open";

/** One segment per horse for the progress bar on top: the current one wins over done. */
export function roundSegments(order: readonly TodayHorse[], current: number | null): SegmentState[] {
  return order.map((h, i) => (i === current ? "current" : h.done ? "done" : "open"));
}

/** "Abendrunde · 7 Pferde" */
export function roundEyebrow(count: number): string {
  return `Abendrunde · ${count === 1 ? "1 Pferd" : `${count} Pferde`}`;
}

/** "Box 12 · dein Pferd", "Box 12", "dein Pferd" or "". */
export function boxLine(item: Pick<TodayHorse, "horse">, mine: boolean): string {
  const box = item.horse.box?.trim();
  return [box ? `Box ${box}` : "", mine ? "dein Pferd" : ""].filter(Boolean).join(" · ");
}

// --- texts of a horse's screen ---------------------------------------------------------------

/** Without a recommendation the round still lets the person decide ("Eingedeckt", "Abgedeckt", "Geprüft"). */
export function hasRecommendation(rec: Recommendation): boolean {
  return rec.status === "blanket" || rec.status === "none";
}

/** Big title of tonight: the blanket, "Keine Decke", or why there is no advice. */
export function roundTitle(rec: Recommendation): string {
  return hasRecommendation(rec) ? recommendationTitle(rec) : "Keine Empfehlung — Regel prüfen";
}

/** "4 bis 9 °C, 4,8 mm Regen, Wind bis 62 km/h" (wind only when it is strong), from the night's weather. */
export function weatherLine(weather: Weather | null): string {
  if (!weather) return "Noch keine Vorhersage";
  const parts = [temperatureRange(weather), rainAmount(weather)];
  if (weather.wind_kmh >= 40) parts.push(`Wind bis ${Math.round(weather.wind_kmh)} km/h`);
  return parts.join(", ");
}

/** Lines under the title: the weather, "von Hand gewählt" or the owner's wish, where the blanket hangs. */
export function roundDetails(rec: Recommendation, weather: Weather | null): string[] {
  const lines = [weatherLine(weather)];
  if (rec.manual) lines.push(rec.note ? `von Hand gewählt: ${rec.note}` : "von Hand gewählt");
  else if (rec.note) lines.push(`Besitzerwunsch: ${rec.note}`);
  if (rec.status === "blanket" && rec.blanket?.location) lines.push(`Ort: ${rec.blanket.location}`);
  return lines;
}

export type RoundAction = { action: StateAction; label: string };

/**
 * The buttons that set the state. The first of `stateActions` is the big one: "Eingedeckt" for a
 * blanket (and without advice), "Keine Decke — passt" when the rules want none. The rest (Abgedeckt,
 * Geprüft) are small.
 */
export function roundActions(rec: Recommendation): { primary: RoundAction; extra: RoundAction[] } {
  const [first, ...rest] = stateActions(rec) as [StateAction, ...StateAction[]];
  const label = (a: StateAction) => (a === "checked" && rec.status === "none" ? "Keine Decke — passt" : actionLabel(a));
  return {
    primary: { action: first, label: label(first) },
    extra: rest.map((action) => ({ action, label: label(action) })),
  };
}

/** The peek row: "Merlin als Nächstes" over "Fleecedecke · Box 11" ("Keine Empfehlung" without advice). */
export function peekText(item: TodayHorse): { title: string; detail: string } {
  const box = item.horse.box?.trim();
  const rec = item.recommendation;
  return {
    title: `${item.horse.name} als Nächstes`,
    detail: [hasRecommendation(rec) ? recommendationTitle(rec) : "Keine Empfehlung", box ? `Box ${box}` : ""]
      .filter(Boolean)
      .join(" · "),
  };
}

// --- summary ---------------------------------------------------------------------------------

export type SummaryRow = {
  horseId: string;
  name: string;
  /** "Eingedeckt mit Regendecke 100 g" / "Noch offen". */
  text: string;
  /** Who and when ("Jan, 19:04"), empty for open horses. */
  meta: string;
  done: boolean;
};

export type RoundSummary = {
  /** "7 von 7 versorgt" */
  title: string;
  subtitle: string;
  openCount: number;
  rows: SummaryRow[];
};

/** What was set for each horse, open horses marked. */
export function roundSummary(order: readonly TodayHorse[], timeZone: string): RoundSummary {
  const done = order.filter((h) => h.done).length;
  const open = order.length - done;
  return {
    title: `${done} von ${order.length} versorgt`,
    subtitle: open === 0 ? "Alle Pferde sind versorgt." : open === 1 ? "Ein Pferd ist noch offen." : `${open} Pferde sind noch offen.`,
    openCount: open,
    rows: order.map((h) => ({
      horseId: h.horse.id,
      name: h.horse.name,
      done: h.done,
      text: h.done && h.state ? stateLabel(h.state) : "Noch offen",
      meta: h.done && h.state ? stateByline(h.state, timeZone) : "",
    })),
  };
}
