import * as SecureStore from "expo-secure-store";

// The session token lives in the platform keychain/keystore (expo-secure-store).
// An in-memory copy lets the API client read it synchronously.
const KEY = "reiterhof.session";

let cached: string | null = null;

export function getToken(): string | null {
  return cached;
}

/** Reads the stored token into memory. Call once at start. */
export async function loadToken(): Promise<string | null> {
  try {
    cached = await SecureStore.getItemAsync(KEY);
  } catch {
    cached = null;
  }
  return cached;
}

export async function saveToken(token: string): Promise<void> {
  cached = token;
  await SecureStore.setItemAsync(KEY, token);
}

export async function clearToken(): Promise<void> {
  cached = null;
  try {
    await SecureStore.deleteItemAsync(KEY);
  } catch {
    // nothing stored or keystore unavailable: the in-memory token is gone, which is what matters
  }
}
