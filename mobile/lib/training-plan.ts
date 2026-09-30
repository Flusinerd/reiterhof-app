// Pure helpers of the week plan (JAN-89, JAN-92): POST /api/v1/horses/{id}/week/plan returns a proposal
// for the open days; the app applies the days with PUT /week/{day}. See docs/domains/training.md.

import type { WeekDay } from "./api/training";
import { activityLabel, formatMinutes, type Activity, type WeekDayLike } from "./training.ts";

/** Whether and how the language model took part (planOut.ai_status). */
export type PlanAIStatus =
  | "used"
  | "not_configured"
  | "no_consent"
  | "owner_under_16"
  | "limit"
  | "failed"
  | "nothing_to_plan";

export type PlanDay = {
  date: string;
  weekday: number;
  activity: Activity | "rest";
  label: string;
  minutes: number;
  intensity: string;
  intensity_label: string;
  reason: string;
  note?: string;
  /** "ai": accepted proposal of the model; "rules": from the fixed rules. */
  source: "ai" | "rules";
  /** Why the rules replaced the model's proposal for this day. */
  replaced?: string;
  /** Somebody already claimed the day without an activity; applying keeps them. */
  user: { id: string; name: string; color_key?: string } | null;
  /** Content of the unit in a few words (only from the model). */
  focus?: string;
  /** Library exercise for the unit; null for hack, walker and rest. */
  exercise: { id: string; title: string } | null;
  /** Level of the unit by its load (JAN-93); absent for rest days. */
  level?: "recovery" | "light" | "normal" | "demanding";
  /** German label of the level, e.g. "aktive Erholung". */
  level_label?: string;
  /** The owner changed the day by hand after the proposal; the model's reason no longer applies. */
  edited?: boolean;
};

export type PlanResponse = {
  horse_id: string;
  start: string;
  end: string;
  source: "ai" | "rules";
  ai_status: PlanAIStatus;
  /** Only the owner can grant the ai_training consent. */
  owner_is_me: boolean;
  days: PlanDay[];
};

/** Body of PUT /week/{day} that stores one planned day. */
export type PlanApplyBody = {
  status: "planned" | "rest";
  activity?: Activity;
  /** Keeps whoever claimed the day; "" = nobody. */
  user_id?: string;
  note: string;
  focus?: string;
  exercise_id?: string;
};

/** The server allows at most 500 characters per note. */
export const MAX_NOTE = 500;

/** Note stored with a planned day, e.g. "KI-Vorschlag: 45 Min. · Nach dem Ausritt …". */
export function planDayNote(day: PlanDay): string {
  if (day.edited) return day.activity === "rest" ? "Plan: Ruhetag" : `Plan: ${formatMinutes(day.minutes)}`;
  const lead = day.source === "ai" ? "KI-Vorschlag" : "Plan";
  const parts = [day.activity === "rest" ? "Ruhetag" : `${day.minutes} Min.`, day.reason].filter((p) => p !== "");
  const note = `${lead}: ${parts.join(" · ")}`;
  return note.length > MAX_NOTE ? `${note.slice(0, MAX_NOTE - 1)}…` : note;
}

/**
 * PUT body for one plan day; a claimed day keeps its person. A rest day proposed for a day
 * somebody claimed is not applied (null): storing it would silently remove their claim.
 */
export function planApplyBody(day: PlanDay): PlanApplyBody | null {
  if (day.activity === "rest") return day.user ? null : { status: "rest", note: planDayNote(day) };
  const body: PlanApplyBody = { status: "planned", activity: day.activity, user_id: day.user?.id ?? "", note: planDayNote(day) };
  // An edited day states focus and exercise even when empty, so that the owner's clearing is stored.
  if (day.focus || day.edited) body.focus = day.focus ?? "";
  if (day.exercise || day.edited) body.exercise_id = day.exercise?.id ?? "";
  return body;
}

/** Hint above the plan; null when the model planned (the rows carry their labels). */
export function planStatusText(status: PlanAIStatus, ownerIsMe: boolean): string | null {
  switch (status) {
    case "used":
      return null;
    case "not_configured":
      return "KI-Vorschläge sind auf dem Server nicht eingerichtet. Der Plan kommt aus den festen Regeln.";
    case "no_consent":
      return ownerIsMe
        ? "Der Plan kommt aus den festen Regeln. Mit deiner Erlaubnis schlägt eine KI die Woche vor."
        : "Der Plan kommt aus den festen Regeln. KI-Vorschläge kann nur der Besitzer des Pferdes erlauben.";
    case "owner_under_16":
      return "KI-Vorschläge gibt es nur für Pferde von Besitzern ab 16 Jahren. Der Plan kommt aus den festen Regeln.";
    case "limit":
      return "Das KI-Kontingent ist für diesen Monat aufgebraucht. Der Plan kommt aus den festen Regeln.";
    case "failed":
      return "Die KI war nicht erreichbar. Der Plan kommt aus den festen Regeln.";
    case "nothing_to_plan":
      return "In dieser Woche ist kein Tag mehr offen.";
  }
}

/** True when the owner can switch the model on for this plan. */
export function canAskForAI(plan: PlanResponse): boolean {
  return plan.ai_status === "no_consent" && plan.owner_is_me;
}

// --- next step -------------------------------------------------------------------------

/** What the training tab suggests next: set up the profile, or plan the open days of the week. */
export type NextStep = { kind: "profile" } | { kind: "plan"; openDays: number } | null;

/**
 * Days the planner would fill (as in the backend's plan.go): status open, or today/planned
 * without an activity. Done, rest and empty days never count; neither do days before today
 * when the week shows which one is today.
 */
export function openPlanDays<T extends WeekDayLike>(days: readonly T[]): T[] {
  const today = days.find((d) => d.is_today)?.date;
  return days.filter((d) => {
    if (today !== undefined && d.date < today) return false;
    if (d.status === "open") return true;
    return (d.status === "today" || d.status === "planned") && !d.activity;
  });
}

/**
 * The next step for the horse: the profile when there is none (owners only), else planning
 * when a day after today is still open. Undefined data (still loading) gives null.
 */
export function nextStep(
  today: { has_profile: boolean; can_edit: boolean } | undefined,
  week: { days: readonly WeekDayLike[] } | undefined,
): NextStep {
  if (!today || !today.can_edit) return null;
  if (!today.has_profile) return { kind: "profile" };
  if (!week) return null;
  const open = openPlanDays(week.days);
  if (!open.some((d) => !d.is_today)) return null;
  return { kind: "plan", openDays: open.length };
}

// --- editing a day ---------------------------------------------------------------------

/** What the day editor changes: rest or a planned unit. */
export type DayEdit = {
  status: "planned" | "rest";
  activity: Activity | null;
  minutes: number;
  focus: string;
  exercise: { id: string; title: string } | null;
  userId: string | null;
};

/** The draft day after the owner's change; the model's reason and level no longer apply. */
export function editedPlanDay(day: PlanDay, edit: DayEdit): PlanDay {
  const { level: _level, level_label: _levelLabel, ...kept } = day;
  if (edit.status === "rest" || edit.activity === null) {
    return { ...kept, activity: "rest", label: activityLabel("rest"), minutes: 0, focus: "", exercise: null, edited: true };
  }
  return {
    ...kept,
    activity: edit.activity,
    label: activityLabel(edit.activity),
    minutes: edit.minutes,
    focus: edit.focus,
    exercise: edit.exercise,
    edited: true,
  };
}

/** Number of draft days that "Übernehmen" would store. */
export function applicableDays(days: readonly PlanDay[]): number {
  return days.filter((d) => planApplyBody(d) !== null).length;
}

/** Replaces the draft day with the same date, keeping the order. */
export function replaceDraftDay(days: readonly PlanDay[], fresh: PlanDay): PlanDay[] {
  return days.map((d) => (d.date === fresh.date ? fresh : d));
}

/** Body of PUT /week/{day}. */
export type TakeDayBody = {
  status: "planned" | "rest" | "open";
  activity?: Activity;
  user_id?: string;
  note?: string;
  focus?: string;
  exercise_id?: string;
};

/**
 * PUT body for a day edited in the week view. A planned day always carries user_id ("" = nobody),
 * otherwise the server would claim it for the caller.
 */
export function dayEditBody(edit: DayEdit): TakeDayBody {
  if (edit.status === "rest" || edit.activity === null) return { status: "rest", note: "Plan: Ruhetag" };
  return {
    status: "planned",
    activity: edit.activity,
    user_id: edit.userId ?? "",
    note: `Plan: ${formatMinutes(edit.minutes)}`,
    focus: edit.focus,
    exercise_id: edit.exercise?.id ?? "",
  };
}

/** The editor's starting value for a saved week day or a draft day. */
export function dayEditFrom(day: WeekDay | PlanDay): DayEdit {
  if ("status" in day) {
    return {
      status: day.status === "rest" ? "rest" : "planned",
      activity: day.activity,
      minutes: day.minutes,
      focus: day.focus ?? "",
      exercise: day.exercise,
      userId: day.user?.id ?? null,
    };
  }
  const activity = day.activity === "rest" ? null : day.activity;
  return {
    status: activity === null ? "rest" : "planned",
    activity,
    minutes: day.minutes,
    focus: day.focus ?? "",
    exercise: day.exercise,
    userId: day.user?.id ?? null,
  };
}

/** Context for proposing one day again: the other draft days, and the activity not to propose. */
export type ReproposeOptions = {
  day: string;
  draft: { date: string; activity: Activity | "rest"; minutes: number }[];
  exclude: Activity[];
};

export function reproposeOptions(draftDays: readonly PlanDay[], date: string): ReproposeOptions {
  const current = draftDays.find((d) => d.date === date);
  return {
    day: date,
    draft: draftDays
      .filter((d) => d.date !== date)
      .map((d) => ({ date: d.date, activity: d.activity, minutes: d.activity === "rest" ? 0 : d.minutes })),
    exclude: current && current.activity !== "rest" ? [current.activity] : [],
  };
}
