// Pure platform helpers (no React Native imports, unit-tested).
// lib/platform.ts binds them to `Platform.OS`; lib/api.ts uses `resolveApiUrl`.

export type PlatformOS = "ios" | "android" | "web" | "windows" | "macos";

/** What the current platform can do; screens ask this instead of comparing `Platform.OS`. */
export type Capabilities = {
  /** Automatic check-in/out when the phone enters or leaves the stable (OS geofencing). */
  geofence: boolean;
  /** "Mit Apple anmelden" (native Sign in with Apple; the web flow is not set up). */
  appleSignIn: boolean;
  /** GPS recording that continues with a locked screen. */
  backgroundLocation: boolean;
  /** Insert an event into the device calendar; without it the app offers an ICS file. */
  deviceCalendar: boolean;
  /** Session token in the keychain/keystore; on web it is localStorage. */
  secureStorage: boolean;
  /** Motion permission has to be requested from a user gesture (iOS Safari). */
  motionNeedsGesture: boolean;
};

const NATIVE: Capabilities = {
  geofence: true,
  appleSignIn: true,
  backgroundLocation: true,
  deviceCalendar: true,
  secureStorage: true,
  motionNeedsGesture: false,
};

const WEB: Capabilities = {
  geofence: false,
  appleSignIn: false,
  backgroundLocation: false,
  deviceCalendar: false,
  secureStorage: false,
  motionNeedsGesture: true,
};

export function capabilitiesFor(os: PlatformOS): Capabilities {
  if (os === "ios") return { ...NATIVE };
  if (os === "android") return { ...NATIVE, appleSignIn: false };
  // web, and the windows/macos targets that do not exist for this app
  return { ...WEB };
}

export const DEFAULT_NATIVE_API_URL = "http://10.0.2.2:8080";

export type ApiUrlInput = {
  os: PlatformOS;
  /** `expo.extra.apiUrl` (native default, e.g. https://api.stallfunk.de). */
  configUrl?: unknown;
  /** Build-time override (`STALLFUNK_API_URL` / `EXPO_PUBLIC_API_URL`), used on every platform when set. */
  override?: unknown;
  /** `window.location.origin` on web. */
  origin?: string | null;
};

const clean = (v: unknown): string | null => {
  const t = typeof v === "string" ? v.trim() : "";
  return t ? t.replace(/\/+$/, "") : null;
};

/**
 * Base URL of the API. Native: the override, else `expo.extra.apiUrl`. Web: the override,
 * else the origin of the page (the PWA is served from the same host that proxies `/api`),
 * else the configured URL when there is no window (build tooling).
 */
export function resolveApiUrl({ os, configUrl, override, origin }: ApiUrlInput): string {
  const forced = clean(override);
  if (forced) return forced;
  if (os === "web") return clean(origin) ?? clean(configUrl) ?? DEFAULT_NATIVE_API_URL;
  return clean(configUrl) ?? DEFAULT_NATIVE_API_URL;
}
