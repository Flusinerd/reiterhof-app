// "Erledigt" on a health item (docs/domains/horses.md): what the button is called, when it is offered,
// how the acknowledged item looks, in which order the rows stay while the user is on the screen. Pure,
// no React Native imports (unit-tested). German UI text.

import { ApiError, errorMessage } from "./api-core.ts";
import { formatDate } from "./horse-format.ts";

/** The parts of a health item (GET /horses/{id}/health) these helpers read. */
export type DoneItem = {
  id: string;
  kind: string;
  label: string;
  /** YYYY-MM-DD. For a medication course the last day of the course. */
  due_date: string | null;
  interval_count: number | null;
  daily_time: string | null;
  /** The stable-local day of the last "erledigt", YYYY-MM-DD. */
  last_done_on?: string | null;
  last_done_by?: { id: string; name: string } | null;
};

/**
 * A medication course: kind medication or a daily reminder time. Its due date is the last day of the course,
 * so "erledigt" ends it ("Gabe beenden") instead of moving a date (same rule as the backend,
 * `kind = 'medication' OR daily_time IS NOT NULL`).
 */
export function isCourse(item: Pick<DoneItem, "kind" | "daily_time">): boolean {
  return item.kind === "medication" || !!item.daily_time;
}

/** An appointment can be marked done with a date or an interval; a course only with an end date (without one it never ends). */
export function canMarkDone(item: DoneItem): boolean {
  return isCourse(item) ? !!item.due_date : !!(item.due_date || item.interval_count);
}

/** The label of the button: "Erledigt", for a course "Gabe beenden". */
export function doneLabel(item: Pick<DoneItem, "kind" | "daily_time">): string {
  return isCourse(item) ? "Gabe beenden" : "Erledigt";
}

/** Texts of the sheet that asks before a course is ended. */
export function endCourseSheet(item: Pick<DoneItem, "label">): { title: string; description: string; confirm: string } {
  return {
    title: "Gabe beenden?",
    description: `${item.label || "Das Medikament"} endet heute. Du bekommst dafür keine Erinnerungen mehr.`,
    confirm: "Gabe beenden",
  };
}

/** What the row shows instead of the button once the item was done today: a check mark, a title and a line below. */
export type DoneState = { title: string; detail: string };

/**
 * The success state of an item that was marked done today, `null` for any other item. Somebody else's
 * "erledigt" names them ("Heute erledigt von Anna"), your own does not.
 */
export function doneState(item: DoneItem, today: string, meId?: string): DoneState | null {
  if (!item.last_done_on || item.last_done_on !== today) return null;
  const by = item.last_done_by && item.last_done_by.id !== meId ? item.last_done_by.name : "";
  if (isCourse(item)) {
    const ends = !item.due_date || item.due_date === today ? "Endet heute" : `Endet am ${formatDate(item.due_date)}`;
    return { title: by ? `Gabe beendet von ${by}` : "Gabe beendet", detail: `${ends}. Keine Erinnerungen mehr.` };
  }
  return {
    title: by ? `Heute erledigt von ${by}` : "Heute erledigt",
    detail: item.due_date ? `Nächster Termin: ${formatDate(item.due_date)}` : "Abgeschlossen",
  };
}

/** German text for the errors of "erledigt" and its undo; everything else as usual. */
export function healthErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === "due_date_changed") return "Der Termin wurde inzwischen geändert. Die Liste ist aktuell, versuch es noch mal.";
    if (err.code === "undo_expired") return "Das lässt sich nicht mehr rückgängig machen.";
  }
  return errorMessage(err);
}

/**
 * The order the rows keep while the user is on the screen. The list comes sorted by due date, so an item
 * that was done jumps to a new place under the finger; the frozen order keeps every id where it was and
 * puts an id that appears (a new appointment) after its predecessor in the server's order. Ids that are
 * gone drop out. Start with an empty `previous` on every new visit.
 */
export function freezeOrder(previous: readonly string[], current: readonly string[]): string[] {
  const present = new Set(current);
  const out = previous.filter((id) => present.has(id));
  const known = new Set(out);
  current.forEach((id, i) => {
    if (known.has(id)) return;
    const before = i > 0 ? out.indexOf(current[i - 1] as string) : -1;
    out.splice(before + 1, 0, id);
    known.add(id);
  });
  return out;
}

/** The items in the frozen order of `order` (ids missing from it go last, in their own order). */
export function inOrder<T extends { id: string }>(items: readonly T[], order: readonly string[]): T[] {
  const rank = new Map(order.map((id, i) => [id, i]));
  return [...items].sort((a, b) => (rank.get(a.id) ?? order.length) - (rank.get(b.id) ?? order.length));
}
