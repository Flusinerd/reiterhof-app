// Pure consent helpers (no React Native imports, unit-tested). Backend: internal/privacy,
// docs/domains/privacy.md. The explanation texts here are UI copy (German); the full legal
// text is docs/legal/datenschutz.md, bundled through consent-legal-texts.ts.

export type ConsentKind = "location_geofence" | "location_tracking" | "presence_sharing" | "photos" | "push";

/** Same order as the server (`privacy.Kinds()`). */
export const CONSENT_KINDS: readonly ConsentKind[] = [
  "location_geofence",
  "location_tracking",
  "presence_sharing",
  "photos",
  "push",
];

/** One entry of `GET /api/v1/me/consents`. */
export type ConsentItem = {
  kind: ConsentKind;
  granted: boolean;
  version: string | null;
  granted_at: string | null;
  revoked_at: string | null;
  current_version: string;
  up_to_date: boolean;
};

export type ConsentList = { current_version: string; items: ConsentItem[] };

export type ConsentCopy = {
  /** Short name in lists and switches. */
  label: string;
  /** Sheet title, phrased as a question. */
  title: string;
  /** One plain sentence: what for. */
  summary: string;
  /** What exactly happens with the data. */
  points: string[];
  /** What is not possible without the consent. */
  declined: string;
  /** Text of the confirming button. */
  accept: string;
};

export const CONSENT_COPY: Record<ConsentKind, ConsentCopy> = {
  location_geofence: {
    label: "Automatisch im Stall anmelden",
    title: "Automatisch im Stall anmelden?",
    summary: "Die App meldet dich an und ab, wenn du im Stall ankommst oder gehst.",
    points: [
      "Dein Handy überwacht einen Bereich um den Stall, auch bei geschlossener App (Standort „Immer“).",
      "Dein Standort bleibt auf dem Handy. Der Server erfährt nur „angekommen“ und „gegangen“.",
      "Du kannst das jederzeit ausschalten.",
    ],
    declined: "Ohne Erlaubnis meldest du dich mit „Bin da“ selbst an.",
    accept: "Erlauben",
  },
  location_tracking: {
    label: "Strecke beim Reiten aufzeichnen",
    title: "Deine Strecke beim Reiten aufzeichnen?",
    summary: "Die App misst Strecke und Gangarten beim Reiten.",
    points: [
      "Positionspunkte werden nur gespeichert, solange die Aufzeichnung läuft.",
      "Sie gehören zur Einheit. Sehen können sie Besitzer, Reitbeteiligungen des Pferdes und Admins.",
      "Nach 12 Monaten werden sie gelöscht. Dauer und Strecke bleiben.",
    ],
    declined: "Ohne Erlaubnis trägst du Einheiten von Hand ein.",
    accept: "Erlauben",
  },
  presence_sharing: {
    label: "Anwesenheit zeigen",
    title: "Zeigen, wenn du im Stall bist?",
    summary: "Andere sehen, ob und wann du im Stall bist.",
    points: [
      "Was andere sehen, bestimmst du mit „Wer sieht mich?“: alles, nur den Tag oder nichts.",
      "Besuche werden nach 12 Monaten gelöscht.",
    ],
    declined: "Ohne Erlaubnis bist du für andere unsichtbar.",
    accept: "Erlauben",
  },
  photos: {
    label: "Fotos und Kamera",
    title: "Kamera und Fotos nutzen?",
    summary: "Fotos aufnehmen oder auswählen, z. B. für Pferdeakte und Auffälligkeiten.",
    points: [
      "Fotos liegen auf einem Server in Deutschland und sind für Mitglieder deines Stalls abrufbar. Dokumente sehen nur Besitzer, Reitbeteiligungen und Admins.",
      "Fotos können Metadaten wie den Aufnahmeort enthalten. Fotografiere niemanden ohne Einverständnis.",
    ],
    declined: "Ohne Erlaubnis keine Foto- oder Dokument-Uploads.",
    accept: "Erlauben",
  },
  push: {
    label: "Benachrichtigungen",
    title: "Benachrichtigungen erhalten?",
    summary: "Erinnerungen an Termine, neue Anfragen und Deckenwechsel.",
    points: [
      "Dein Handy sendet einen Geräte-Schlüssel (Push-Token) an den Server.",
      "Nachrichten laufen über die Push-Dienste von Expo, Apple und Google (in der Web-App über den Push-Dienst deines Browsers).",
      "Einzelne Erinnerungen kannst du abschalten.",
    ],
    declined: "Ohne Erlaubnis bekommst du keine Benachrichtigungen.",
    accept: "Erlauben",
  },
};

/** The item of a kind; a missing item counts as never granted. */
export function findConsent(items: readonly ConsentItem[] | undefined, kind: ConsentKind): ConsentItem | undefined {
  return items?.find((i) => i.kind === kind);
}

/** True while the consent is granted (whatever text version it was given for). */
export function isGranted(items: readonly ConsentItem[] | undefined, kind: ConsentKind): boolean {
  return findConsent(items, kind)?.granted === true;
}

/**
 * True when the explanation sheet has to be shown before the feature is used: never granted,
 * revoked, or granted for an older text version.
 */
export function needsPrompt(items: readonly ConsentItem[] | undefined, kind: ConsentKind): boolean {
  const c = findConsent(items, kind);
  return !c || !c.granted || !c.up_to_date;
}

/** Items in display order, one per known kind (unknown kinds from a newer server are dropped). */
export function orderedConsents(items: readonly ConsentItem[] | undefined): ConsentItem[] {
  const out: ConsentItem[] = [];
  for (const kind of CONSENT_KINDS) {
    const c = findConsent(items, kind);
    if (c) out.push(c);
  }
  return out;
}

/** German text for a refused account deletion, by server error code; null for other codes. */
export function deleteBlockedMessage(code: string): string | null {
  switch (code) {
    case "owns_horses":
      return "Du bist noch Besitzer eines Pferdes. Übergib es an ein anderes Mitglied (Admins können den Besitzer ändern) oder lass es löschen.";
    case "last_admin":
      return "Du bist der letzte Admin. Ernenne zuerst einen anderen Admin.";
    default:
      return null;
  }
}

/** German text for a refused consent change; null for other codes. */
export function consentErrorMessage(code: string): string | null {
  return code === "version_mismatch"
    ? "Datenschutztext geändert. Aktualisiere die App und versuch es noch mal."
    : null;
}

/** File name of the data export, e.g. `reiterhof-export-2026-09-30.json` (UTC date). */
export function exportFileName(now: Date): string {
  return `reiterhof-export-${now.toISOString().slice(0, 10)}.json`;
}

/** Pretty-printed JSON for the export file. */
export function formatExport(data: unknown): string {
  return JSON.stringify(data, null, 2);
}
