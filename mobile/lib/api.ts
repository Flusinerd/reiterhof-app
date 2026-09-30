import Constants from "expo-constants";
import { Platform } from "react-native";

import { createClient } from "./api-core.ts";
import { resolveApiUrl, type PlatformOS } from "./platform-core.ts";
import { getToken } from "./token";

export { ApiError, errorMessage } from "./api-core.ts";

// Base URL of the cloud API. Native: expo.extra.apiUrl from app.json (on the Android emulator,
// 10.0.2.2 points to the host machine). Web (PWA): the origin of the page, because the host
// that serves the app also proxies /api. STALLFUNK_API_URL (through app.config.js) or
// EXPO_PUBLIC_API_URL override it at build time, e.g. for local development.
// The realtime stream, file URLs and ICS links are built from this value as well.
export const API_URL: string = resolveApiUrl({
  os: Platform.OS as PlatformOS,
  configUrl: Constants.expoConfig?.extra?.apiUrl,
  override: Constants.expoConfig?.extra?.apiUrlOverride || process.env.EXPO_PUBLIC_API_URL,
  origin: typeof window !== "undefined" ? window.location?.origin : undefined,
});

let unauthorizedHandler: (() => void) | null = null;

/** AuthProvider registers itself here to drop the session when the server answers 401. */
export function setUnauthorizedHandler(handler: (() => void) | null) {
  unauthorizedHandler = handler;
}

const client = createClient({
  baseUrl: API_URL,
  getToken,
  onUnauthorized: () => unauthorizedHandler?.(),
});

// --- types (mirror docs/architecture.md, "Authentication and roles") ---------------------

export type PresenceVisibility = "all" | "only_day" | "hidden";

export type User = {
  id: string;
  name: string;
  email: string;
  phone: string | null;
  avatar_color: string | null;
  presence_visibility: PresenceVisibility;
  is_admin: boolean;
  stable_id: string | null;
  /** False while the name is only the part of the email before the "@" (the app asks for it). */
  name_confirmed: boolean;
  /**
   * "confirmed": 16 or older, or a parent consented. "parent_pending": a parent was asked by
   * mail. "unknown": nothing stated yet; the app asks before a stable can be joined (Art. 8 GDPR).
   */
  age_status: AgeStatus;
  /** The parent's address while under 16 (null otherwise). */
  parent_email: string | null;
};

export type AgeStatus = "unknown" | "confirmed" | "parent_pending";

export type Stable = {
  id: string;
  name: string;
  farm_name: string | null;
  city: string | null;
  timezone: string;
  /** Location of the stable for the presence geofence; null until set. */
  lat: number | null;
  lng: number | null;
  geofence_radius_m: number;
};

export type Me = {
  user: User;
  /** null until the user joined a stable. */
  stable: Stable | null;
  roles: { owned_horse_ids: string[]; rider_horse_ids: string[] };
};

export type Session = { token: string; user: User };

export type ProfilePatch = Partial<{
  name: string;
  /** Empty string clears the value. */
  phone: string;
  avatar_color: string;
  presence_visibility: PresenceVisibility;
}>;

// --- endpoints -----------------------------------------------------------------------------

export const api = {
  /** Always resolves, whether or not the address is known (no user enumeration). */
  requestMagicLink: (email: string) => client.post<void>("/api/v1/auth/magic-link", { email }),
  verifyMagicLink: (token: string) => client.post<Session>("/api/v1/auth/verify", { token }),
  /** Signs in with the 6-digit code from the login mail (same session as the link). */
  verifyLoginCode: (email: string, code: string) => client.post<Session>("/api/v1/auth/verify-code", { email, code }),
  signInWithGoogle: (idToken: string) => client.post<Session>("/api/v1/auth/google", { id_token: idToken }),
  signInWithApple: (idToken: string, name?: string) =>
    client.post<Session>("/api/v1/auth/apple", { id_token: idToken, ...(name ? { name } : {}) }),
  logout: () => client.post<void>("/api/v1/auth/logout"),
  me: () => client.get<Me>("/api/v1/me"),
  updateMe: (patch: ProfilePatch) => client.patch<Me>("/api/v1/me", patch),
  joinStable: (code: string) => client.post<Me>("/api/v1/stables/join", { code }),
  /** "Ich bin 16 oder älter." */
  confirmAge: () => client.post<Me>("/api/v1/me/age", { over_16: true }),
  /** Under 16: mails the parent a confirmation link (204). 409 `already_confirmed`, 429 after three mails a day. */
  requestParentalConsent: (parentEmail: string) =>
    client.post<void>("/api/v1/me/parental-consent", { parent_email: parentEmail }),
  createInvite: () => client.post<{ code: string; expires_at: string; max_uses: number }>("/api/v1/stables/invites"),
};

/** Other features build their own calls on this client: `authed.get<Horse[]>("/api/v1/horses")`. */
export const authed = client;

export type Health = { status: string };

export const fetchHealth = () => client.get<Health>("/healthz");
