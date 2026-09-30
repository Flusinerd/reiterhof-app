// Parsing of the magic link. Pure (no React Native imports).

const TOKEN_PATTERN = /^[A-Za-z0-9_-]{20,128}$/;

/** True for something that looks like a login token (base64url, as issued by the backend). */
export function isPlausibleToken(value: string | undefined | null): value is string {
  return typeof value === "string" && TOKEN_PATTERN.test(value);
}

/**
 * Extracts the token from `stallfunk://auth/verify?token=...` or the https fallback
 * `https://host/auth/verify?token=...`. Returns null for anything else.
 */
export function parseVerifyLink(url: string): string | null {
  const match = /^(?:stallfunk:\/\/|https?:\/\/[^/?#]+\/)auth\/verify\?([^#]*)/i.exec(url.trim());
  if (!match) return null;
  for (const pair of match[1].split("&")) {
    const [key, raw = ""] = pair.split("=");
    if (key !== "token") continue;
    let value: string;
    try {
      value = decodeURIComponent(raw);
    } catch {
      return null;
    }
    return isPlausibleToken(value) ? value : null;
  }
  return null;
}
