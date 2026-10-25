// Pure parts of deleting a horse for good (no React Native imports, unit-tested in horse-delete.test.ts).
// The backend decides the same way (horses.NameMatches): the typed name must equal the horse's name,
// case-insensitive and trimmed. All user-facing strings are German.

/** True when `typed` is the horse's name: ignores case and surrounding spaces; an empty field never matches. */
export function nameMatches(name: string, typed: string): boolean {
  const t = typed.trim();
  return t !== "" && name.trim().toLowerCase() === t.toLowerCase();
}

/** What disappears with the horse, in the order of the sheet. */
export const DELETE_SCOPE = ["Tagebuch", "Training", "Gesundheit", "Decken", "Reha", "Dokumente", "Fotos", "Aufgaben"] as const;

export type DeleteSheetText = {
  title: string;
  /** "Dabei werden gelöscht: Tagebuch, Training, …" */
  scope: string;
  warning: string;
  fieldLabel: string;
  placeholder: string;
  confirm: string;
  cancel: string;
};

/** The words of the confirm sheet. */
export function deleteSheetText(horseName: string): DeleteSheetText {
  return {
    title: `${horseName} löschen?`,
    scope: `Gelöscht werden alle Daten von ${horseName}: ${DELETE_SCOPE.join(", ")}. Hilfegesuche bleiben ohne Pferd bestehen.`,
    warning: "Das lässt sich nicht rückgängig machen.",
    fieldLabel: "Zum Bestätigen den Namen eingeben",
    placeholder: horseName,
    confirm: "Endgültig löschen",
    cancel: "Abbrechen",
  };
}

/** The result line on the screen the user lands on. */
export function deletedMessage(horseName: string): string {
  return `${horseName} wurde gelöscht.`;
}

/**
 * Horses whose deletion is under way or done in this app. A realtime `horse.changed` for such a horse
 * (the event of our own delete arrives like any other) must not refetch the horse's own queries: they
 * would answer 404 (see `eventKeys` in query-effects.ts).
 */
const deleting = new Set<string>();

export const deletedHorses = {
  mark: (id: string) => void deleting.add(id),
  /** The delete failed: the horse is still there. */
  unmark: (id: string) => void deleting.delete(id),
  has: (id: string) => deleting.has(id),
};
