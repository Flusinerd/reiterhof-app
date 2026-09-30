// Pure helpers of the exercise library browser (JAN-58) and the exercise checklist of the
// indoor tracker (JAN-65). No React Native imports (unit-tested with node --test).

export type ExerciseLike = {
  id: string;
  title: string;
  discipline: string;
  level: string;
  goal_tags: string[];
};

export type ExerciseFilter = { discipline: string | null; level: string | null; tag: string | null };

export const NO_FILTER: ExerciseFilter = { discipline: null, level: null, tag: null };

const LEVEL_ORDER = ["beginner", "intermediate", "advanced"] as const;

const LEVEL_LABELS: Record<string, string> = {
  beginner: "Einsteiger",
  intermediate: "Fortgeschritten",
  advanced: "Erfahren",
};

const DISCIPLINE_LABELS: Record<string, string> = {
  dressage: "Dressur",
  jumping: "Springen",
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
};

const capitalize = (s: string) => (s ? s.charAt(0).toUpperCase() + s.slice(1) : s);

export function levelLabel(level: string): string {
  return LEVEL_LABELS[level] ?? (level ? capitalize(level) : "");
}

export function disciplineLabel(discipline: string): string {
  return DISCIPLINE_LABELS[discipline] ?? (discipline ? capitalize(discipline.replace(/_/g, " ")) : "");
}

export function tagLabel(tag: string): string {
  return TAG_LABELS[tag] ?? capitalize(tag.replace(/_/g, " "));
}

export function applyFilter<T extends ExerciseLike>(list: readonly T[], filter: ExerciseFilter): T[] {
  return list.filter(
    (e) =>
      (!filter.discipline || e.discipline === filter.discipline) &&
      (!filter.level || e.level === filter.level) &&
      (!filter.tag || e.goal_tags.includes(filter.tag)),
  );
}

export type FilterOptions = { disciplines: string[]; levels: string[]; tags: string[] };

/** The values that occur in the list (levels from easy to hard, the rest alphabetical by label). */
export function filterOptions(list: readonly ExerciseLike[]): FilterOptions {
  const disciplines = new Set<string>();
  const levels = new Set<string>();
  const tags = new Set<string>();
  for (const e of list) {
    if (e.discipline) disciplines.add(e.discipline);
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

/** Selects a value or clears it when it is already selected. */
export function toggleFilter(filter: ExerciseFilter, key: keyof ExerciseFilter, value: string): ExerciseFilter {
  return { ...filter, [key]: filter[key] === value ? null : value };
}

export function hasFilter(filter: ExerciseFilter): boolean {
  return !!(filter.discipline || filter.level || filter.tag);
}

/** "3 Schritte" / "1 Schritt". */
export function stepCountLabel(count: number): string {
  return count === 1 ? "1 Schritt" : `${count} Schritte`;
}

/** "2 von 5 Schritten erledigt". */
export function checklistLabel(done: number, total: number): string {
  return `${done} von ${total} ${total === 1 ? "Schritt" : "Schritten"} erledigt`;
}
