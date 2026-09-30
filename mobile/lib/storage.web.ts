/**
 * Web variant of `lib/storage.ts`: localStorage.
 *
 * This is less protected than the keychain/keystore that the native app uses: any script that
 * runs on the origin can read it. The PWA has no third-party scripts, the origin serves only
 * this app, and on iOS an installed PWA has its own storage, separate from Safari's. The session
 * token is a random bearer token that the server can revoke (sign-out, account deletion).
 * localStorage can throw (private mode, storage cleared or blocked): callers already treat
 * that as "nothing stored".
 */
export async function getItem(key: string): Promise<string | null> {
  return window.localStorage.getItem(key);
}

export async function setItem(key: string, value: string): Promise<void> {
  window.localStorage.setItem(key, value);
}

export async function deleteItem(key: string): Promise<void> {
  window.localStorage.removeItem(key);
}
