// Pure logic of "Wie war's?", the step after a ride (docs/domains/training.md, "Finish"): feeling
// chips, the exercise checklist that starts out ticked from the tracker or the plan, the place
// line under the duration, the note placeholder and the body of the session. No React Native
// imports (unit-tested with node --test). German UI text.

import { activityLabel, type Activity, type ReinSegment } from "./training.ts";

// --- feeling ---------------------------------------------------------------------------

/** How the session felt for the rider (`feeling` of the API). Not the horse's `feel`. */
export const FEELING_OPTIONS = [
  { value: "hard", label: "Schwer" },
  { value: "good", label: "Gut" },
  { value: "easy", label: "Sehr locker" },
] as const;
export type Feeling = (typeof FEELING_OPTIONS)[number]["value"];

export function feelingLabel(feeling: string | null | undefined): string {
  return FEELING_OPTIONS.find((f) => f.value === feeling)?.label ?? "";
}

/**
 * The rating of the practised exercise that the feeling stands for (1 Schwer, 2 Besser, 3 Sitzt),
 * so that "Sehr locker" still unlocks the next step of the exercise chain. Without a feeling
 * nothing is rated.
 */
export function focusFromFeeling(feeling: Feeling | null): 1 | 2 | 3 | undefined {
  switch (feeling) {
    case "hard":
      return 1;
    case "good":
      return 2;
    case "easy":
      return 3;
    default:
      return undefined;
  }
}

// --- place and duration ----------------------------------------------------------------

const PLACES: Record<Activity, string> = {
  hall: "in der Halle",
  arena: "auf dem Platz",
  hack: "im Gelände",
  lunge: "an der Longe",
  jumping: "beim Springen",
  groundwork: "bei der Bodenarbeit",
  walker: "an der Führanlage",
};

/** "in der Halle", "im Gelände", ... */
export function placeLabel(activity: Activity): string {
  return PLACES[activity];
}

/** Line next to the big number: "Minuten in der Halle, aus dem Tracking". */
export function durationCaption(activity: Activity, tracked: boolean): string {
  return `Minuten ${placeLabel(activity)}${tracked ? ", aus dem Tracking" : ""}`;
}

/** Minutes typed into the big number field: whole minutes 1 to 600, else null. */
export function parseMinutes(text: string): number | null {
  if (!/^\d{1,3}$/.test(text.trim())) return null;
  const n = Number(text);
  return n >= 1 && n <= 600 ? n : null;
}

/** "Luna · gerade eben" */
export function finishEyebrow(horseName: string): string {
  return `${horseName} · gerade eben`;
}

// --- note ------------------------------------------------------------------------------

type HorseWho = {
  owner: { id: string; name: string } | null;
  riders: readonly { user_id: string; name: string }[];
};

/**
 * First name of the person who also rides or owns the horse and reads the diary: the owner for a
 * rider, the first rider for the owner. Null when there is nobody else.
 */
export function otherPerson(horse: HorseWho | undefined, myId: string | undefined): string | null {
  if (!horse || !myId) return null;
  const other =
    horse.owner && horse.owner.id !== myId ? horse.owner.name : horse.riders.find((r) => r.user_id !== myId)?.name;
  const first = other?.trim().split(/\s+/)[0];
  return first ? first : null;
}

/** "Notiz für Mia …" with another person, else "Notiz …". */
export function notePlaceholder(other: string | null): string {
  return other ? `Notiz für ${other} …` : "Notiz …";
}

/** Label of the switch that shows the session to the rider ("Mia sieht die Einheit"). */
export function visibleLabel(other: string): string {
  return `${other} sieht die Einheit`;
}

// --- exercise checklist ----------------------------------------------------------------

/** What the checklist rows are: the steps of the exercise, or the exercise itself when it has none. */
export function checklistItems(title: string, steps: readonly string[] | undefined): string[] {
  return steps && steps.length > 0 ? [...steps] : [title];
}

/**
 * Rows that start ticked: the steps the tracker reported; without tracker data (a ride that was
 * only planned, or entered afterwards) the plan counts as followed and everything is ticked, one
 * tap takes a row off.
 */
export function initialChecked(count: number, stepsDone: readonly number[] | undefined): number[] {
  if (stepsDone === undefined) return Array.from({ length: count }, (_, i) => i);
  return stepsDone.filter((i) => i < count);
}

/** "2 von 3 geübt" for the line next to the exercise title. */
export function practisedLabel(done: number, total: number): string {
  return `${done} von ${total} geübt`;
}

// --- body ------------------------------------------------------------------------------

/** What the finish screen sends; structurally a `SessionInput` of lib/api/training.ts. */
export type FinishBody = {
  activity: Activity;
  minutes: number;
  started_at?: string;
  gait_shares?: Record<string, number>;
  rein_changes?: ReinSegment[];
  distance_m?: number;
  exercise_id?: string;
  feeling?: Feeling;
  focus_rating?: 1 | 2 | 3;
  note?: string;
  visible_to_rider?: boolean;
};

export type FinishForm = {
  activity: Activity;
  minutes: number;
  startedAt?: string;
  gaitShares?: Record<string, number>;
  reinChanges?: ReinSegment[];
  distanceM?: number;
  /** The planned exercise, with how many checklist rows exist and which are ticked. */
  exercise?: { id: string; rows: number; checked: readonly number[] };
  feeling: Feeling | null;
  note: string;
  /** Only set when the person can decide it (owner with a rider). */
  visibleToRider?: boolean;
};

/**
 * Body of `POST /horses/{id}/sessions`. The exercise is only recorded when at least one row is
 * ticked; the feeling doubles as the rating of that exercise (`focusFromFeeling`).
 */
export function buildSessionBody(form: FinishForm): FinishBody {
  const practised = !!form.exercise && (form.exercise.rows === 0 || form.exercise.checked.length > 0);
  const focus = practised ? focusFromFeeling(form.feeling) : undefined;
  const note = form.note.trim();
  return {
    activity: form.activity,
    minutes: form.minutes,
    ...(form.startedAt ? { started_at: form.startedAt } : {}),
    ...(form.gaitShares ? { gait_shares: form.gaitShares } : {}),
    ...(form.reinChanges ? { rein_changes: form.reinChanges } : {}),
    ...(form.distanceM ? { distance_m: form.distanceM } : {}),
    ...(practised && form.exercise ? { exercise_id: form.exercise.id } : {}),
    ...(form.feeling ? { feeling: form.feeling } : {}),
    ...(focus ? { focus_rating: focus } : {}),
    ...(note ? { note } : {}),
    ...(form.visibleToRider !== undefined ? { visible_to_rider: form.visibleToRider } : {}),
  };
}

/** Title of the saved screen: "Halle, 47 Min." */
export function savedTitle(activity: string, minutes: number): string {
  return `${activityLabel(activity)}, ${Math.round(minutes)} Min.`;
}
