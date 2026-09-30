// Pure helpers for observations ("Auffälligkeiten", JAN-50 to JAN-52, JAN-54). No React Native
// imports, so they can be unit-tested with node --test. All user-facing strings are German.

import { formatDate } from "./horse-format.ts";

// --- routes --------------------------------------------------------------------------------

/** Detail screen of an observation. */
export function observationRoute(id: string): string {
  return `/observations/${id}`;
}

/** "Melden" screen, optionally with a preselected horse. */
export function newObservationRoute(horseId?: string | null): string {
  return horseId ? `/observations/new?horse=${encodeURIComponent(horseId)}` : "/observations/new";
}

// --- categories ----------------------------------------------------------------------------

export const CATEGORIES = [
  "cough",
  "lameness",
  "injury",
  "not_eating",
  "colic",
  "behavior",
  "blanket_equipment",
  "other",
] as const;
export type Category = (typeof CATEGORIES)[number];

const CATEGORY_LABELS: Record<Category, string> = {
  cough: "Husten",
  lameness: "Lahmheit",
  injury: "Verletzung",
  not_eating: "Frisst nicht",
  colic: "Kolik",
  behavior: "Verhalten",
  blanket_equipment: "Decke/Ausrüstung",
  other: "Sonstiges",
};

/** German label of a category key; unknown keys are returned unchanged, missing ones give "Auffälligkeit". */
export function categoryLabel(key: string | null | undefined): string {
  if (!key) return "Auffälligkeit";
  return CATEGORY_LABELS[key as Category] ?? key;
}

// --- body parts ----------------------------------------------------------------------------

export const BODY_PARTS = [
  "front_left",
  "front_right",
  "hind_left",
  "hind_right",
  "head",
  "back",
  "belly",
  "other",
] as const;
export type BodyPart = (typeof BODY_PARTS)[number];

const BODY_PART_LABELS: Record<BodyPart, string> = {
  front_left: "Vorne links",
  front_right: "Vorne rechts",
  hind_left: "Hinten links",
  hind_right: "Hinten rechts",
  head: "Kopf",
  back: "Rücken",
  belly: "Bauch",
  other: "Sonstiges",
};

/** German label of a body part key; "" without one. */
export function bodyPartLabel(key: string | null | undefined): string {
  if (!key) return "";
  return BODY_PART_LABELS[key as BodyPart] ?? key;
}

/** Short caption for the legs in the pictogram ("VL", "VR", "HL", "HR"); "" for other parts. */
export function bodyPartShort(key: string): string {
  switch (key) {
    case "front_left":
      return "VL";
    case "front_right":
      return "VR";
    case "hind_left":
      return "HL";
    case "hind_right":
      return "HR";
    default:
      return "";
  }
}

// --- urgency and status --------------------------------------------------------------------

export const URGENCIES = ["info", "check", "urgent"] as const;
export type Urgency = (typeof URGENCIES)[number];

const URGENCY_LABELS: Record<Urgency, string> = {
  info: "Info",
  check: "Bitte ansehen",
  urgent: "Dringend",
};

export function urgencyLabel(key: string): string {
  return URGENCY_LABELS[key as Urgency] ?? key;
}

/** One-sentence explanation shown under the urgency choice. */
export function urgencyHint(key: string): string {
  switch (key) {
    case "urgent":
      return "Notfallkarte öffnet sich, alle Anwesenden und die Besitzer werden sofort benachrichtigt.";
    case "check":
      return "Besitzer und Reitbeteiligungen werden benachrichtigt.";
    default:
      return "Besitzer und Reitbeteiligungen werden informiert.";
  }
}

export type BadgeVariant = "neutral" | "primary" | "accent" | "info" | "danger";

export function urgencyBadgeVariant(key: string): BadgeVariant {
  if (key === "urgent") return "danger";
  if (key === "check") return "accent";
  return "neutral";
}

export const STATUSES = ["watch", "done"] as const;
export type Status = (typeof STATUSES)[number];

/** "Beobachten" / "Erledigt". */
export function statusLabel(status: string): string {
  if (status === "watch") return "Beobachten";
  if (status === "done") return "Erledigt";
  return status;
}

export function statusBadgeVariant(status: string): BadgeVariant {
  return status === "done" ? "primary" : "info";
}

/** The status a status button switches to. */
export function nextStatus(status: string): Status {
  return status === "done" ? "watch" : "done";
}

/** Label of the button that switches to `nextStatus(status)`. */
export function statusActionLabel(status: string): string {
  return status === "done" ? "Wieder beobachten" : "Als erledigt markieren";
}

// --- texts ---------------------------------------------------------------------------------

/** "Husten · Kopf" (category and body part, whatever is present). */
export function observationTitle(o: { category: string | null; body_part: string | null }): string {
  const parts = [categoryLabel(o.category), bodyPartLabel(o.body_part)].filter((p) => p !== "");
  return parts.join(" · ");
}

/** "heute", "gestern", "vor 5 Tagen"; from 14 days on the date "08.10.2026". */
export function reportedText(iso: string, now: Date): string {
  const t = new Date(iso);
  if (Number.isNaN(t.getTime())) return "";
  const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOf(now) - startOf(t)) / 86_400_000);
  if (days <= 0) return "heute";
  if (days === 1) return "gestern";
  if (days < 14) return `vor ${days} Tagen`;
  return formatDate(iso);
}

/** Counts open (status "watch") observations. */
export function openCount(list: readonly { status: string }[]): number {
  return list.filter((o) => o.status === "watch").length;
}

/** Text of the bundling hint (JAN-54), e.g. "3 weitere Pferde sind ebenfalls fällig – Termin bündeln?". */
export function bundleText(count: number): string {
  if (count <= 0) return "";
  if (count === 1) return "1 weiteres Pferd ist ebenfalls fällig – Termin bündeln?";
  return `${count} weitere Pferde sind ebenfalls fällig – Termin bündeln?`;
}

/** "Fanta, Luna und Nala" for the names of the other horses. */
export function horseNamesText(names: readonly string[]): string {
  if (names.length <= 1) return names.join("");
  return `${names.slice(0, -1).join(", ")} und ${names[names.length - 1]}`;
}

// --- form ----------------------------------------------------------------------------------

/** Most photos per report (backend limit). */
export const MAX_PHOTOS = 6;

export const MAX_DESCRIPTION = 2000;

export type ReportDraft = {
  horseId: string | null;
  category: string | null;
  description: string;
  photos: number;
};

/** German error message for an incomplete report, or null if it can be sent. */
export function reportProblem(draft: ReportDraft): string | null {
  if (!draft.horseId) return "Bitte wähle ein Pferd aus.";
  if (!draft.category) return "Bitte wähle aus, was dir aufgefallen ist.";
  if (draft.description.trim().length > MAX_DESCRIPTION) return `Die Beschreibung ist zu lang (höchstens ${MAX_DESCRIPTION} Zeichen).`;
  if (draft.photos > MAX_PHOTOS) return `Höchstens ${MAX_PHOTOS} Fotos pro Meldung.`;
  return null;
}

/** Title and text of the confirmation before an urgent report is sent. */
export const URGENT_CONFIRM = {
  title: "Dringend melden?",
  message: "Alle Anwesenden und die Besitzer werden sofort benachrichtigt. Danach öffnet sich die Notfallkarte.",
} as const;
