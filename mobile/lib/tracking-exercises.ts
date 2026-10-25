// Pure helpers of the exercise library browser (JAN-58) and the exercise checklist of the
// indoor tracker (JAN-65). No React Native imports (unit-tested with node --test).

import { MISSING_LABEL } from "./training-equipment.ts";
import { UNIT_PARTS, phaseFitsPart, type UnitPartKey } from "./training-focus.ts";

export type ExerciseLike = {
  id: string;
  title: string;
  discipline: string;
  level: string;
  goal_tags: string[];
  /** The library (what `discipline` now means); older backends send only `discipline`. */
  library?: string;
  /** The part of the unit it fits: `warmup`, `main`, `cooldown` or `any`. */
  phase?: string;
  /** Typical duration in minutes. */
  minutes?: number | null;
  intensity?: string;
  /** False when the stable's equipment is not enough for it (only with `?horse=`). */
  available?: boolean | null;
};

/** `phase` is a part of the unit (Lösen, Arbeiten, Entspannen); `focus` is a key of the focus catalog (filtered by the server). */
export type ExerciseFilter = {
  discipline: string | null;
  level: string | null;
  tag: string | null;
  phase?: UnitPartKey | null;
  focus?: string | null;
};

export const NO_FILTER: ExerciseFilter = { discipline: null, level: null, tag: null };

const LEVEL_ORDER = ["beginner", "intermediate", "advanced"] as const;

const LEVEL_LABELS: Record<string, string> = {
  beginner: "Einsteiger",
  intermediate: "Fortgeschritten",
  advanced: "Erfahren",
};

/** Libraries (and the older disciplines of the profile, which share the column). */
const DISCIPLINE_LABELS: Record<string, string> = {
  dressage: "Dressur",
  jumping: "Springen",
  cavaletti: "Cavaletti",
  hack: "Gelände",
  eventing: "Vielseitigkeit",
  leisure: "Freizeit",
  western: "Western",
  young_horse: "Jungpferd",
  groundwork: "Bodenarbeit",
  lunge: "Longe",
};

/** Goal tags are stored as ASCII keys; these are the German words. */
const TAG_LABELS: Record<string, string> = {
  seitengaenge: "Seitengänge",
  biegung: "Biegung",
  losgelassenheit: "Losgelassenheit",
  durchlaessigkeit: "Durchlässigkeit",
  springgymnastik: "Springgymnastik",
  rhythmus: "Rhythmus",
  takt: "Takt",
  balance: "Balance",
  aufmerksamkeit: "Aufmerksamkeit",
  vertrauen: "Vertrauen",
  anlehnung: "Anlehnung",
  schwung: "Schwung",
  geraderichten: "Geraderichten",
  versammlung: "Versammlung",
  gelassenheit: "Gelassenheit",
  kondition: "Kondition",
  ausdauer: "Ausdauer",
  kraft: "Kraft",
  gelaende: "Gelände",
  mobilitaet: "Beweglichkeit",
  turnier: "Turnier",
};

const capitalize = (s: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : s);

export function levelLabel(level: string): string {
  return LEVEL_LABELS[level] ?? (level ? capitalize(level) : "");
}

export function disciplineLabel(discipline: string): string {
  return DISCIPLINE_LABELS[discipline] ?? (discipline ? capitalize(discipline.replace(/_/g, " ")) : "");
}

/** German name of a library: Dressur, Springen, Cavaletti, Bodenarbeit, Longe, Gelände, Western. */
export const libraryLabel = disciplineLabel;

/** The library of an exercise: `library`, else the `discipline` older backends send. */
export function libraryOf(e: Pick<ExerciseLike, "discipline" | "library">): string {
  return e.library || e.discipline;
}

export function tagLabel(tag: string): string {
  return TAG_LABELS[tag] ?? capitalize(tag.replace(/_/g, " "));
}

/** "Lösen", "Arbeiten" or "Entspannen": the part of the unit a phase filter stands for. */
export function phaseLabel(part: string): string {
  return UNIT_PARTS.find((p) => p.key === part)?.label ?? part;
}

/** The phase of an exercise as words: "Lösen", "Arbeiten", "Entspannen", or all three for `any`; "" when unknown. */
export function exercisePhaseText(phase: string | null | undefined): string {
  if (phase === "any") return "Lösen, Arbeiten oder Entspannen";
  return UNIT_PARTS.find((p) => p.key === phase)?.label ?? "";
}

const INTENSITY_LABELS: Record<string, string> = { light: "leicht", normal: "normal", demanding: "fordernd" };

/** How demanding an exercise is (`intensity` of the catalog, not of an activity). */
export function exerciseIntensityLabel(intensity: string | null | undefined): string {
  return intensity ? (INTENSITY_LABELS[intensity] ?? intensity) : "";
}

/** "10 Min." for a typical duration, "" without one. */
export function minutesText(minutes: number | null | undefined): string {
  return minutes != null && minutes > 0 ? `${Math.round(minutes)} Min.` : "";
}

export function applyFilter<T extends ExerciseLike>(list: readonly T[], filter: ExerciseFilter): T[] {
  return list.filter(
    (e) =>
      (!filter.discipline || libraryOf(e) === filter.discipline) &&
      (!filter.level || e.level === filter.level) &&
      (!filter.tag || e.goal_tags.includes(filter.tag)) &&
      (!filter.phase || phaseFitsPart(e.phase, filter.phase)),
  );
}

export type FilterOptions = { disciplines: string[]; levels: string[]; tags: string[] };

/** The values that occur in the list (levels from easy to hard, the rest alphabetical by label). */
export function filterOptions(list: readonly ExerciseLike[]): FilterOptions {
  const disciplines = new Set<string>();
  const levels = new Set<string>();
  const tags = new Set<string>();
  for (const e of list) {
    if (libraryOf(e)) disciplines.add(libraryOf(e));
    if (e.level) levels.add(e.level);
    for (const t of e.goal_tags) tags.add(t);
  }
  const rank = (l: string) => {
    const i = (LEVEL_ORDER as readonly string[]).indexOf(l);
    return i < 0 ? LEVEL_ORDER.length : i;
  };
  return {
    disciplines: [...disciplines].sort((a, b) => disciplineLabel(a).localeCompare(disciplineLabel(b), "de")),
    levels: [...levels].sort((a, b) => rank(a) - rank(b) || a.localeCompare(b)),
    tags: [...tags].sort((a, b) => tagLabel(a).localeCompare(tagLabel(b), "de")),
  };
}

/**
 * The phase filters to offer: Lösen, Arbeiten, Entspannen, as far as an exercise with a phase
 * fits them. Without any phase in the list (older backend) there are none.
 */
export function phaseOptions(list: readonly ExerciseLike[]): UnitPartKey[] {
  const withPhase = list.filter((e) => !!e.phase);
  return UNIT_PARTS.map((p) => p.key).filter((part) => withPhase.some((e) => phaseFitsPart(e.phase, part)));
}

/** Selects a value or clears it when it is already selected. */
export function toggleFilter<K extends keyof ExerciseFilter>(
  filter: ExerciseFilter,
  key: K,
  value: NonNullable<ExerciseFilter[K]>,
): ExerciseFilter {
  return { ...filter, [key]: filter[key] === value ? null : value };
}

export function hasFilter(filter: ExerciseFilter): boolean {
  return !!(filter.discipline || filter.level || filter.tag || filter.phase || filter.focus);
}

/** "3 Schritte" / "1 Schritt". */
export function stepCountLabel(count: number): string {
  return count === 1 ? "1 Schritt" : `${count} Schritte`;
}

/** "2 von 5 Schritten erledigt". */
export function checklistLabel(done: number, total: number): string {
  return `${done} von ${total} ${total === 1 ? "Schritt" : "Schritten"} erledigt`;
}

/**
 * The line under the title of a library row: library, minutes, steps, goals; "Ausstattung fehlt"
 * first when the stable cannot set the exercise up.
 */
export function exerciseRowDetail(e: ExerciseLike & { step_count?: number }): string {
  return [
    e.available === false ? MISSING_LABEL : "",
    libraryLabel(libraryOf(e)),
    minutesText(e.minutes),
    e.step_count !== undefined ? stepCountLabel(e.step_count) : "",
    e.goal_tags.map(tagLabel).join(", "),
  ]
    .filter(Boolean)
    .join(" · ");
}

/** Query string of `GET /exercises`: the horse (adds `available`) and the focus (the server matches its tags); "" for none. */
export function exerciseQueryString(query: { horse?: string | null; focus?: string | null }): string {
  const parts: string[] = [];
  if (query.horse) parts.push(`horse=${encodeURIComponent(query.horse)}`);
  if (query.focus) parts.push(`focus=${encodeURIComponent(query.focus)}`);
  return parts.length > 0 ? `?${parts.join("&")}` : "";
}

// --- sources ---------------------------------------------------------------------------

/** One source of an exercise: a link that can be opened, or a plain title. */
export type SourceItem = {
  /** What the row says: the site of a link ("magazin.pferdesport-deutschland.de"), else the text. */
  title: string;
  /** The rest of a link ("/project/lektion-im-fokus-09-2021"), "" otherwise. */
  detail: string;
  /** Only http and https links; null for plain titles. */
  url: string | null;
};

/** Sources of the detail screen. Links keep their address, every other entry is shown as text and never opened. */
export function sourceItems(sources: readonly string[] | null | undefined): SourceItem[] {
  return (sources ?? [])
    .map((s) => s.trim())
    .filter((s) => s !== "")
    .map((s) => {
      const m = /^https?:\/\/([^/?#\s]+)([^?#\s]*)/i.exec(s);
      if (!m || !/^https?:\/\/\S+$/i.test(s)) return { title: s, detail: "", url: null };
      const path = (m[2] ?? "").replace(/\/+$/, "");
      return { title: (m[1] ?? s).replace(/^www\./i, ""), detail: path, url: s };
    });
}
