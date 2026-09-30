// Pure helpers of the reha plan feature (no React Native imports, unit-tested with node --test):
// the minute ramp and phase dates (same rules as backend/internal/reha), German labels and the
// conversion between the API shape and the editable form of app/reha/edit.tsx.
import { ACTIVITIES, activityLabel, addDays, formatDate, isValidDate, type Activity } from "./training.ts";

/** A phase activity is a training activity or "rest" (Boxenruhe: no exercise at all). */
export type PhaseActivity = Activity | "rest";

export const PHASE_ACTIVITIES: readonly { value: PhaseActivity; label: string }[] = [
  ...ACTIVITIES.map((value) => ({ value, label: activityLabel(value) })),
  { value: "rest", label: "Boxenruhe" },
];

export function phaseActivityLabel(activity: string): string {
  return activity === "rest" ? "Boxenruhe" : activityLabel(activity);
}

// Limits, the same as the backend validates.
export const MAX_PHASES = 20;
export const MAX_PHASE_DAYS = 365;
export const MAX_TOTAL_DAYS = 730;
export const MAX_MINUTES = 240;

// --- ramp and dates --------------------------------------------------------------------------

/**
 * Allowed minutes on day `dayInPhase` (0-based) of a phase of `days` days: rises linearly from
 * `min` on the first to `max` on the last day, rounded half up. A one-day phase allows `max`.
 * Example: 10 -> 20 over 9 days gives 10, 11, 13, 14, 15, 16, 18, 19, 20.
 */
export function rampMinutes(min: number, max: number, dayInPhase: number, days: number): number {
  if (days <= 1 || max <= min) return max;
  if (dayInPhase <= 0) return min;
  if (dayInPhase >= days - 1) return max;
  const steps = days - 1;
  const num = min * steps + (max - min) * dayInPhase;
  return Math.floor((2 * num + steps) / (2 * steps));
}

export type DatedPhase<T> = T & { start: string; end: string };

/** Places phases one after the other, beginning on `startDate` (YYYY-MM-DD, inclusive ends). */
export function placePhases<T extends { days: number }>(startDate: string, phases: readonly T[]): DatedPhase<T>[] {
  let day = startDate;
  return phases.map((p) => {
    const n = Math.max(0, Math.floor(p.days));
    const placed = { ...p, start: day, end: addDays(day, n - 1) };
    day = addDays(day, n);
    return placed;
  });
}

export function totalDays(phases: readonly { days: number }[]): number {
  return phases.reduce((sum, p) => sum + (Number.isFinite(p.days) ? p.days : 0), 0);
}

/** Last day of the plan (inclusive). */
export function planEnd(startDate: string, phases: readonly { days: number }[]): string {
  return addDays(startDate, totalDays(phases) - 1);
}

// --- labels ----------------------------------------------------------------------------------

export function daysText(n: number): string {
  return n === 1 ? "1 Tag" : `${n} Tage`;
}

/** "10–20 Min." or "15 Min."; a rest phase has no minutes. */
export function minutesRange(min: number, max: number, rest = false): string {
  if (rest) return "keine Bewegung";
  return min === max ? `${max} Min.` : `${min}–${max} Min.`;
}

/** "22.03. – 30.03." */
export function dateRange(start: string, end: string): string {
  const short = (d: string) => `${d.slice(8, 10)}.${d.slice(5, 7)}.`;
  return start === end ? short(start) : `${short(start)} – ${short(end)}`;
}

export type PhaseLike = { activity: string; days: number; min_minutes: number; max_minutes: number };

/** "Bodenarbeit · 10–20 Min. · 9 Tage" */
export function phaseSummary(p: PhaseLike): string {
  const rest = p.activity === "rest";
  const parts = [rest ? "Boxenruhe" : phaseActivityLabel(p.activity)];
  if (!rest) parts.push(minutesRange(p.min_minutes, p.max_minutes));
  parts.push(daysText(p.days));
  return parts.join(" · ");
}

/** The reha line of a week day: "Schritt führen: Bodenarbeit 14 Min. · erledigt". */
export function weekRehaText(r: { phase: string; activity: string; rest: boolean; minutes: number; done: boolean }): string {
  const what = r.rest ? "keine Bewegung" : `${phaseActivityLabel(r.activity)} ${r.minutes} Min.`;
  return `${r.phase}: ${what}${r.done ? " · erledigt" : ""}`;
}

/** Vet checkup relative to today: "Heute", "Morgen", "In 16 Tagen", "Seit 2 Tagen überfällig". */
export function checkupText(inDays: number): string {
  if (inDays === 0) return "Heute";
  if (inDays === 1) return "Morgen";
  if (inDays > 1) return `In ${inDays} Tagen`;
  if (inDays === -1) return "Seit gestern überfällig";
  return `Seit ${-inDays} Tagen überfällig`;
}

export type PlanState = "upcoming" | "running" | "finished";

/** One line about where the plan is today. */
export function planStatusText(state: PlanState, dayIndex: number, total: number, startDate: string): string {
  switch (state) {
    case "upcoming":
      return `Startet am ${formatDate(startDate)}`;
    case "finished":
      return "Alle Phasen abgeschlossen";
    default:
      return `Tag ${dayIndex} von ${total}`;
  }
}

/** Share of the plan that has passed, 0 to 1. */
export function planProgress(state: PlanState, dayIndex: number, total: number): number {
  if (state === "finished") return 1;
  if (state === "upcoming" || total <= 0) return 0;
  return Math.min(1, Math.max(0, dayIndex / total));
}

/** German text for reha-specific API error codes, or null for other codes. */
export function rehaErrorText(code: string): string | null {
  switch (code) {
    case "not_active":
      return "Plan ist schon beendet.";
    case "forbidden":
      return "Keine Berechtigung.";
    case "conflict":
      return "Es gibt schon einen aktiven Reha-Plan.";
    case "not_found":
      return "Reha-Plan nicht gefunden.";
    default:
      return null;
  }
}

// --- editing ---------------------------------------------------------------------------------

export type PhaseDraft = {
  name: string;
  activity: PhaseActivity;
  // Numbers are kept as text while typing.
  days: string;
  minMinutes: string;
  maxMinutes: string;
  conditions: string;
};

export type PlanDraft = {
  diagnosis: string;
  vet: string;
  startDate: string;
  checkupDate: string;
  abortCriteria: string;
  phases: PhaseDraft[];
};

/** A new phase, prefilled from the previous one (a typical plan repeats the activity). */
export function newPhaseDraft(previous?: PhaseDraft): PhaseDraft {
  if (previous && previous.activity !== "rest") {
    return { name: "", activity: previous.activity, days: "7", minMinutes: previous.maxMinutes, maxMinutes: previous.maxMinutes, conditions: "" };
  }
  return { name: "", activity: "walker", days: "7", minMinutes: "10", maxMinutes: "20", conditions: "" };
}

export function emptyPlanDraft(today: string): PlanDraft {
  return { diagnosis: "", vet: "", startDate: today, checkupDate: "", abortCriteria: "", phases: [] };
}

export type PlanLike = {
  diagnosis: string;
  vet: string;
  start_date: string;
  checkup_date: string | null;
  abort_criteria: string;
  phases: { name: string; days: number; activity: string; min_minutes: number; max_minutes: number; conditions: string }[];
};

export function draftFromPlan(plan: PlanLike): PlanDraft {
  return {
    diagnosis: plan.diagnosis,
    vet: plan.vet,
    startDate: plan.start_date,
    checkupDate: plan.checkup_date ?? "",
    abortCriteria: plan.abort_criteria,
    phases: plan.phases.map((p) => ({
      name: p.name,
      activity: p.activity as PhaseActivity,
      days: String(p.days),
      minMinutes: p.activity === "rest" ? "" : String(p.min_minutes),
      maxMinutes: p.activity === "rest" ? "" : String(p.max_minutes),
      conditions: p.conditions,
    })),
  };
}

/** Body of POST /horses/{id}/reha-plans and PATCH /reha-plans/{id} (the PATCH accepts all of it). */
export type PlanInput = {
  diagnosis: string;
  vet: string;
  start_date: string;
  checkup_date: string;
  abort_criteria: string;
  phases: { name: string; days: number; activity: PhaseActivity; min_minutes: number; max_minutes: number; conditions: string }[];
};

export type PhaseCheck = { ok: true; phase: PlanInput["phases"][number] } | { ok: false; error: string };

function wholeNumber(text: string): number | null {
  const t = text.trim();
  return /^\d{1,4}$/.test(t) ? Number(t) : null;
}

/** Validates one phase form with German messages (`n` is the 1-based position). */
export function checkPhase(d: PhaseDraft, n: number): PhaseCheck {
  const where = `Phase ${n}`;
  const name = d.name.trim();
  if (!name) return { ok: false, error: `${where}: Name fehlt.` };
  if (name.length > 80) return { ok: false, error: `${where}: Name max. 80 Zeichen.` };
  const days = wholeNumber(d.days);
  if (days === null || days < 1 || days > MAX_PHASE_DAYS) {
    return { ok: false, error: `${where}: Dauer 1 bis ${MAX_PHASE_DAYS} Tage.` };
  }
  const conditions = d.conditions.trim();
  if (conditions.length > 500) return { ok: false, error: `${where}: Bedingungen max. 500 Zeichen.` };
  if (d.activity === "rest") {
    return { ok: true, phase: { name, days, activity: "rest", min_minutes: 0, max_minutes: 0, conditions } };
  }
  const min = wholeNumber(d.minMinutes);
  const max = wholeNumber(d.maxMinutes);
  if (min === null || max === null || min < 1 || max > MAX_MINUTES) {
    return { ok: false, error: `${where}: Minuten 1 bis ${MAX_MINUTES}.` };
  }
  if (min > max) return { ok: false, error: `${where}: „Von“ darf nicht über „Bis“ liegen.` };
  return { ok: true, phase: { name, days, activity: d.activity, min_minutes: min, max_minutes: max, conditions } };
}

export type PlanCheck = { ok: true; input: PlanInput } | { ok: false; error: string };

/** Validates the plan form; dates are YYYY-MM-DD, `checkupDate` may be empty. */
export function draftToInput(d: PlanDraft): PlanCheck {
  const diagnosis = d.diagnosis.trim();
  if (!diagnosis) return { ok: false, error: "Diagnose fehlt." };
  if (diagnosis.length > 200) return { ok: false, error: "Diagnose max. 200 Zeichen." };
  const vet = d.vet.trim();
  if (vet.length > 120) return { ok: false, error: "Tierarzt max. 120 Zeichen." };
  const abort = d.abortCriteria.trim();
  if (abort.length > 1000) return { ok: false, error: "Abbruchkriterien max. 1000 Zeichen." };
  if (!isValidDate(d.startDate)) return { ok: false, error: "Startdatum ungültig (JJJJ-MM-TT)." };
  const checkup = d.checkupDate.trim();
  if (checkup) {
    if (!isValidDate(checkup)) return { ok: false, error: "Kontrolldatum ungültig (JJJJ-MM-TT)." };
    if (checkup < d.startDate) return { ok: false, error: "Kontrolle nicht vor dem Start." };
  }
  if (d.phases.length === 0) return { ok: false, error: "Mindestens eine Phase nötig." };
  if (d.phases.length > MAX_PHASES) return { ok: false, error: `Max. ${MAX_PHASES} Phasen.` };
  const phases: PlanInput["phases"] = [];
  for (const [i, p] of d.phases.entries()) {
    const r = checkPhase(p, i + 1);
    if (!r.ok) return r;
    phases.push(r.phase);
  }
  if (totalDays(phases) > MAX_TOTAL_DAYS) {
    return { ok: false, error: `Plan max. ${MAX_TOTAL_DAYS} Tage.` };
  }
  return { ok: true, input: { diagnosis, vet, start_date: d.startDate, checkup_date: checkup, abort_criteria: abort, phases } };
}

/** Moves the item at `index` by `delta` (-1 up, +1 down); out of range returns the list as is. */
export function movePhase<T>(list: readonly T[], index: number, delta: -1 | 1): T[] {
  const to = index + delta;
  if (index < 0 || index >= list.length || to < 0 || to >= list.length) return [...list];
  const next = [...list];
  const [item] = next.splice(index, 1);
  next.splice(to, 0, item as T);
  return next;
}

export function removePhase<T>(list: readonly T[], index: number): T[] {
  return list.filter((_, i) => i !== index);
}

/** Replaces the item at `index`, or appends when `index` is -1 / out of range. */
export function putPhase<T>(list: readonly T[], index: number, item: T): T[] {
  if (index < 0 || index >= list.length) return [...list, item];
  return list.map((x, i) => (i === index ? item : x));
}
