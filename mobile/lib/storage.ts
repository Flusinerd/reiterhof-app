import * as SecureStore from "expo-secure-store";

/**
 * Small persistent key/value store for values that belong to this device: the session token,
 * the geofence opt-in, the "consent sheet was shown" flags. Native: the platform keychain/keystore
 * (expo-secure-store). Web: `lib/storage.web.ts`, which uses localStorage.
 * All three calls may reject; callers decide what a failure means.
 */
export function getItem(key: string): Promise<string | null> {
  return SecureStore.getItemAsync(key);
}

export function setItem(key: string, value: string): Promise<void> {
  return SecureStore.setItemAsync(key, value);
}

export function deleteItem(key: string): Promise<void> {
  return SecureStore.deleteItemAsync(key);
}
