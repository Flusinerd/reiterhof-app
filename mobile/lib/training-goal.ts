// Pure helpers of the level goal (docs/domains/training-planning.md, "Level progression"): the classes
// a goal can name, the progress text, and the one-tap change of the horse's level when it is ready.
// No React Native imports (unit-tested with node --test). German UI text.

import { LEVEL_UP_KEY, type FocusWarning } from "./training-focus.ts";

/** The classes a goal can name (the library's levels above `beginner`). */
export type GoalLevel = "intermediate" | "advanced";

/** The lections coming next on the way to the goal: unmastered, prerequisites mastered. */
export type NextLection = { id: string; title: string };

/** `progress` of GET /horses/{id}/training-profile. */
export type LevelProgress = {
  /** The horse's current class (beginner, intermediate or advanced; "" when unknown). */
  level?: string;
  goal_level?: string | null;
  goal_until?: string | null;
  mastered: number;
  total: number;
  next?: NextLection[];
  ready?: boolean;
  /** German sentence, e.g. "Auf dem Weg zu L: 7 von 12 Lektionen gefestigt." */
  text?: string;
};

/** `progress` of GET /horses/{id}/today. */
export type TodayProgress = { text?: string; next?: NextLection[] };

const CLASSES = ["beginner", "intermediate", "advanced"] as const;

/** The class letter of a goal (intermediate ends at L, advanced starts at M). */
const LETTERS: Record<GoalLevel, string> = { intermediate: "L", advanced: "M" };

export function isGoalLevel(value: unknown): value is GoalLevel {
  return value === "intermediate" || value === "advanced";
}

/** The classes above the current one, lowest first; an unknown class counts as beginner. */
export function classesAbove(level: string | null | undefined): GoalLevel[] {
  const index = (CLASSES as readonly string[]).indexOf(level ?? "");
  return CLASSES.slice(Math.max(index, 0) + 1) as GoalLevel[];
}

/** "L" for intermediate, "M" for advanced; null for anything else. */
export function goalLetter(goal: string | null | undefined): string | null {
  return isGoalLevel(goal) ? LETTERS[goal] : null;
}

/** One choice of the goal picker; `value` null is "the class above the current one". */
export type GoalOption = { value: GoalLevel | null; label: string };

/**
 * The picker: the next class ("L (nächste Klasse)"), then every higher class, all named by their
 * letter like the progress text and the level-up. Nothing for a horse that is advanced already.
 */
export function goalOptions(level: string | null | undefined): GoalOption[] {
  const above = classesAbove(level);
  if (above.length === 0) return [];
  return [
    { value: null, label: `${LETTERS[above[0]]} (nächste Klasse)` },
    ...above.slice(1).map((g) => ({ value: g, label: LETTERS[g] })),
  ];
}

/** The option shown as chosen: an explicit goal above the next class, else "Nächste Klasse" (null). */
export function selectedGoal(progress: Pick<LevelProgress, "level" | "goal_level"> | null | undefined): GoalLevel | null {
  const goal = progress?.goal_level;
  if (!isGoalLevel(goal)) return null;
  return goal === classesAbove(progress?.level)[0] ? null : goal;
}

/** "DD.MM." of a YYYY-MM-DD date, "" otherwise. */
function dayMonth(date: string | null | undefined): string {
  const m = /^\d{4}-(\d{2})-(\d{2})$/.exec(date ?? "");
  return m ? `${m[2]}.${m[1]}.` : "";
}

/**
 * The sentence of the progress row: the API's `text`, else one built from the numbers in the same
 * forms ("Auf dem Weg zu L: 7 von 12 Lektionen gefestigt." / "Ziel L bis 15.03.: noch 5 Lektionen." /
 * "Luna ist bereit für L: 10 von 12 Lektionen gefestigt."). Null when there is nothing to say.
 */
export function progressText(progress: LevelProgress | null | undefined, horseName = ""): string | null {
  if (!progress) return null;
  if (progress.text?.trim()) return progress.text.trim();
  const letter = goalLetter(progress.goal_level);
  if (!letter || !(progress.total > 0)) return null;
  const done = `${progress.mastered} von ${progress.total} Lektionen gefestigt`;
  if (progress.ready) return `${horseName.trim() || "Das Pferd"} ist bereit für ${letter}: ${done}.`;
  const until = dayMonth(progress.goal_until);
  if (until) {
    const open = Math.max(0, progress.total - progress.mastered);
    return `Ziel ${letter} bis ${until}: noch ${open} ${open === 1 ? "Lektion" : "Lektionen"}.`;
  }
  return `Auf dem Weg zu ${letter}: ${done}.`;
}

/** The lections to show under the progress row (at most three, as the API sends them). */
export function nextLections(progress: { next?: readonly NextLection[] } | null | undefined): NextLection[] {
  return (progress?.next ?? []).slice(0, 3);
}

// --- level up ----------------------------------------------------------------------------

/** The `level_up` warning and the class it offers; `letter` is null when it cannot be told. */
export type LevelUp = { text: string; letter: string | null };

/**
 * The profile's `level_up` warning with the class to switch to. The letter is the one the warning
 * names ("Einstufung auf L anpassen?"), else the one of the goal.
 */
export function levelUp(
  warnings: readonly FocusWarning[] | null | undefined,
  progress: Pick<LevelProgress, "goal_level"> | null | undefined,
): LevelUp | null {
  const warning = (warnings ?? []).find((w) => w.key === LEVEL_UP_KEY);
  if (!warning) return null;
  const named = /Einstufung auf (.+?) anpassen/.exec(warning.text)?.[1]?.trim();
  return { text: warning.text, letter: named || goalLetter(progress?.goal_level) };
}

/** "Auf L einstufen". */
export function levelUpLabel(letter: string): string {
  return `Auf ${letter} einstufen`;
}

/** Question and explanation of the confirm sheet. */
export function levelUpConfirm(horseName: string, letter: string): { title: string; description: string } {
  const who = horseName.trim() || "Das Pferd";
  return {
    title: `Auf ${letter} einstufen?`,
    description: `${who} bekommt im Profil das Niveau ${letter}. Als Ziel gilt danach wieder die nächste Klasse.`,
  };
}

/** What the goal row and sheet need: the horse's class and the goal, whether or not the API sent `progress`. */
export type GoalState = Pick<LevelProgress, "level" | "goal_level" | "goal_until">;

/**
 * The goal of a profile: the top-level `goal_level`/`goal_until` (always sent), the class from
 * `progress` when there is one (it is null when the goal's class has no lessons).
 */
export function goalOfProfile(profile: {
  goal_level?: string | null;
  goal_until?: string | null;
  progress?: Partial<GoalState> | null;
}): GoalState {
  const progress = profile.progress;
  return {
    level: progress?.level,
    // A stored null means "the class above"; only an absent field (older backend) falls back to progress.
    goal_level: profile.goal_level !== undefined ? profile.goal_level : (progress?.goal_level ?? null),
    goal_until: profile.goal_until !== undefined ? profile.goal_until : (progress?.goal_until ?? null),
  };
}

/** The profile row "Ziel": the chosen class ("L (nächste Klasse)", "M"), with the end date. */
export function goalSummary(goal: GoalState | null | undefined): string {
  const selected = selectedGoal(goal);
  const options = goalOptions(goal?.level);
  const what = options.find((o) => o.value === selected)?.label ?? "Nächste Klasse";
  const until = dayMonth(goal?.goal_until);
  return until ? `${what} bis ${until}` : what;
}
