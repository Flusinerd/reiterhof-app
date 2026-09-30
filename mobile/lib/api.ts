import Constants from "expo-constants";

import { createClient } from "./api-core.ts";
import { getToken } from "./token.ts";

export { ApiError, errorMessage } from "./api-core.ts";

// Base URL of the cloud API. On the Android emulator, 10.0.2.2 points to the
// host machine running the emulator. Override with expo.extra.apiUrl in app.json.
export const API_URL: string =
  (Constants.expoConfig?.extra?.apiUrl as string | undefined) ?? "http://10.0.2.2:8080";

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
};

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
  signInWithGoogle: (idToken: string) => client.post<Session>("/api/v1/auth/google", { id_token: idToken }),
  signInWithApple: (idToken: string, name?: string) =>
    client.post<Session>("/api/v1/auth/apple", { id_token: idToken, ...(name ? { name } : {}) }),
  logout: () => client.post<void>("/api/v1/auth/logout"),
  me: () => client.get<Me>("/api/v1/me"),
  updateMe: (patch: ProfilePatch) => client.patch<Me>("/api/v1/me", patch),
  joinStable: (code: string) => client.post<Me>("/api/v1/stables/join", { code }),
  createInvite: () => client.post<{ code: string; expires_at: string; max_uses: number }>("/api/v1/stables/invites"),
};

/** Other features build their own calls on this client: `authed.get<Horse[]>("/api/v1/horses")`. */
export const authed = client;

export type Health = { status: string };

export const fetchHealth = () => client.get<Health>("/healthz");
