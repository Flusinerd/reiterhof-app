// Helpers for the 6-digit sign-in code. Pure (no React Native imports).

import { ApiError } from "./api-core.ts";

export const LOGIN_CODE_LENGTH = 6;

/** Seconds before "resend" is offered again (the server allows 3 mails per address and 15 minutes). */
export const RESEND_COOLDOWN_SECONDS = 60;

/**
 * Keeps the digits of what the user typed or pasted and cuts at 6: "123 456", "123-456",
 * "Code: 123456" and "123456" all become "123456".
 */
export function normalizeLoginCode(input: string): string {
  return input.replace(/\D/g, "").slice(0, LOGIN_CODE_LENGTH);
}

/** True for exactly six digits (leading zeros allowed). */
export function isCompleteLoginCode(code: string): boolean {
  return /^\d{6}$/.test(code);
}

/** Display form while typing: "123456" -> "123 456". */
export function formatLoginCode(input: string): string {
  const code = normalizeLoginCode(input);
  return code.length > 3 ? `${code.slice(0, 3)} ${code.slice(3)}` : code;
}

/** Whole seconds left until `availableAtMs`, never negative. */
export function cooldownRemainingSeconds(availableAtMs: number, nowMs: number): number {
  return Math.max(0, Math.ceil((availableAtMs - nowMs) / 1000));
}

/** Label of the resend button: "Erneut senden" or "Erneut senden in 0:45". */
export function resendLabel(secondsLeft: number): string {
  if (secondsLeft <= 0) return "Erneut senden";
  const s = Math.ceil(secondsLeft);
  return `Erneut senden in ${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

/** German error text for a failed code sign-in. */
export function loginCodeErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.code === "invalid_code") {
      return "Code falsch oder abgelaufen. Prüfe die Ziffern oder fordere einen neuen an.";
    }
    if (err.code === "rate_limited") {
      return "Zu viele Versuche. Warte kurz.";
    }
    if (err.code === "network") {
      return "Keine Verbindung. Versuch es gleich noch mal.";
    }
  }
  return "Etwas ist schiefgelaufen. Versuch es noch mal.";
}
