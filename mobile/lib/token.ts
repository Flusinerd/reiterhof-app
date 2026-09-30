import * as storage from "./storage";

// The session token lives in the platform keychain/keystore (expo-secure-store; localStorage on web, see lib/storage.web.ts).
// An in-memory copy lets the API client read it synchronously.
const KEY = "reiterhof.session";

let cached: string | null = null;

export function getToken(): string | null {
  return cached;
}

/** Reads the stored token into memory. Call once at start. */
export async function loadToken(): Promise<string | null> {
  try {
    cached = await storage.getItem(KEY);
  } catch {
    cached = null;
  }
  return cached;
}

export async function saveToken(token: string): Promise<void> {
  cached = token;
  await storage.setItem(KEY, token);
}

export async function clearToken(): Promise<void> {
  cached = null;
  try {
    await storage.deleteItem(KEY);
  } catch {
    // nothing stored or keystore unavailable: the in-memory token is gone, which is what matters
  }
}
