// Input normalisation for the sign-in forms. Pure (no React Native imports).

/** Trims and lower-cases; returns null if the value is not a plausible email address. */
export function normalizeEmail(input: string): string | null {
  const email = input.trim().toLowerCase();
  if (email.length > 254) return null;
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) ? email : null;
}

/** Upper-cases and drops spaces and dashes, as the backend does: "abcd-efgh" -> "ABCDEFGH". */
export function normalizeInviteCode(input: string): string {
  return input.replace(/[\s-]/g, "").toUpperCase();
}

/** Formats a code for display while typing: "abcdefgh" -> "ABCD-EFGH". */
export function formatInviteCode(input: string): string {
  const code = normalizeInviteCode(input).slice(0, 8);
  return code.length > 4 ? `${code.slice(0, 4)}-${code.slice(4)}` : code;
}
