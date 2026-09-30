// Pure helpers of the week plan (JAN-89): POST /api/v1/horses/{id}/week/plan returns a proposal
// for the open days; the app applies the days with PUT /week/{day}. See docs/domains/training.md.

import type { Activity } from "./training.ts";

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
};

/** The server allows at most 500 characters per note. */
export const MAX_NOTE = 500;

/** Note stored with a planned day, e.g. "KI-Vorschlag: 45 Min. · Nach dem Ausritt …". */
export function planDayNote(day: PlanDay): string {
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
  return { status: "planned", activity: day.activity, user_id: day.user?.id ?? "", note: planDayNote(day) };
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
