// Pure geofence helpers (no React Native imports, unit-tested).

/** Below roughly 100 m iOS and Android report region events unreliably. */
export const MIN_RADIUS_M = 100;
export const DEFAULT_RADIUS_M = 150;
export const MAX_RADIUS_M = 2000;

export type GeofenceStable = {
  lat: number | null;
  lng: number | null;
  geofence_radius_m: number | null | undefined;
};

export type GeofenceRegion = {
  identifier: string;
  latitude: number;
  longitude: number;
  radius: number;
  notifyOnEnter: boolean;
  notifyOnExit: boolean;
};

export const REGION_ID = "reiterhof-stable";

/** The region to monitor, or null when the stable has no (valid) coordinates. */
export function regionFor(stable: GeofenceStable | null | undefined): GeofenceRegion | null {
  if (!stable) return null;
  const { lat, lng } = stable;
  if (typeof lat !== "number" || typeof lng !== "number") return null;
  if (!Number.isFinite(lat) || !Number.isFinite(lng) || Math.abs(lat) > 90 || Math.abs(lng) > 180) return null;
  const configured = stable.geofence_radius_m;
  const radius = Math.min(
    MAX_RADIUS_M,
    Math.max(MIN_RADIUS_M, typeof configured === "number" && configured > 0 ? configured : DEFAULT_RADIUS_M),
  );
  return { identifier: REGION_ID, latitude: lat, longitude: lng, radius, notifyOnEnter: true, notifyOnExit: true };
}

export type RegionEvent = "enter" | "exit";

/** Maps the OS event to the API call; null for events we ignore. */
export function actionFor(event: RegionEvent | null | undefined): "check-in" | "check-out" | null {
  if (event === "enter") return "check-in";
  if (event === "exit") return "check-out";
  return null;
}

export type GeofenceFailure =
  | "unsupported"
  | "no_location"
  | "foreground_denied"
  | "background_denied"
  | "error";

/** German explanation for a failed activation. */
export function failureMessage(reason: GeofenceFailure): string {
  switch (reason) {
    case "unsupported":
      return "Die automatische Erkennung funktioniert nur in der App auf dem Handy.";
    case "no_location":
      return "Für diesen Stall ist noch kein Standort hinterlegt.";
    case "foreground_denied":
      return "Ohne Standortzugriff kann die App nicht erkennen, wann du im Stall bist.";
    case "background_denied":
      return "Bitte erlaube den Standortzugriff „Immer“ in den Einstellungen, damit die App auch im Hintergrund erkennt, wann du kommst und gehst.";
    case "error":
      return "Die automatische Erkennung konnte nicht gestartet werden. Bitte versuche es erneut.";
  }
}
