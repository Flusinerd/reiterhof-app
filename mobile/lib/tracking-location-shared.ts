import type { RawFix } from "./tracking";

// Parts of the ride tracker's location module that native (`tracking-location.ts`) and web
// (`tracking-location.web.ts`) share: the listener of the tracking screen, the result type
// and the German error texts.

type Listener = (fixes: RawFix[]) => void;
let listener: Listener | null = null;

/** The tracking screen registers here to receive the fixes. */
export function setFixListener(next: Listener | null): void {
  listener = next;
}

export function currentFixListener(): Listener | null {
  return listener;
}

export type StartResult =
  | { ok: true; background: boolean }
  | { ok: false; reason: "unsupported" | "denied" | "blocked" | "services-off" | "error" };

/** German text for a failed start. */
export function startFailureText(reason: Exclude<StartResult, { ok: true }>["reason"]): string {
  switch (reason) {
    case "denied":
      return "Ohne Standortzugriff keine Aufzeichnung. Erlaube den Zugriff und versuche es erneut.";
    case "blocked":
      return "Standortzugriff ausgeschaltet. Erlaube ihn in den Telefon-Einstellungen für Stallfunk.";
    case "services-off":
      return "Ortungsdienste sind aus. Schalte sie ein und versuche es erneut.";
    case "unsupported":
      return "Ortung ist auf diesem Gerät nicht verfügbar.";
    default:
      return "Ortung konnte nicht starten.";
  }
}
