// Pure helpers for the browser sensors (no DOM or React Native imports, unit-tested).
// lib/tracking-sensors.web.ts and lib/tracking-location.web.ts use them.

/** Standard gravity. `devicemotion` reports m/s², the gait detector expects g. */
export const STANDARD_GRAVITY = 9.80665;

export type Vec3 = { x: number; y: number; z: number };

/**
 * Converts `DeviceMotionEvent.accelerationIncludingGravity` (m/s²) to the g units of the
 * `GaitStream`. Returns null when the browser delivers no usable values (null, NaN).
 * The detector estimates gravity from the samples, so the axis sign convention of the
 * browser (it differs between iOS and Android) does not matter.
 */
export function accelerationToG(
  a: { x: number | null; y: number | null; z: number | null } | null | undefined,
): Vec3 | null {
  if (!a) return null;
  const { x, y, z } = a;
  if (typeof x !== "number" || typeof y !== "number" || typeof z !== "number") return null;
  if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(z)) return null;
  return { x: x / STANDARD_GRAVITY, y: y / STANDARD_GRAVITY, z: z / STANDARD_GRAVITY };
}

/** The part of a `GeolocationPosition` that we read. */
export type GeoPositionLike = {
  timestamp: number;
  coords: {
    latitude: number;
    longitude: number;
    altitude: number | null;
    speed: number | null;
    accuracy: number;
  };
};

export type WebFix = {
  t: number;
  lat: number;
  lon: number;
  alt: number | null;
  speed: number | null;
  accuracy: number | null;
};

/** Same fields as `RawFix` in lib/tracking.ts; `speed` is m/s and may be null. */
export function fixFromPosition(p: GeoPositionLike): WebFix {
  const finite = (v: number | null | undefined) => (typeof v === "number" && Number.isFinite(v) ? v : null);
  return {
    t: p.timestamp,
    lat: p.coords.latitude,
    lon: p.coords.longitude,
    alt: finite(p.coords.altitude),
    speed: finite(p.coords.speed),
    accuracy: finite(p.coords.accuracy),
  };
}

export type WebGeoFailure = "blocked" | "services-off" | "error";

/**
 * Maps `GeolocationPositionError.code` (1 permission denied, 2 position unavailable, 3 timeout)
 * to the failure reasons of the tracker. A browser cannot ask again after a denial, so a denied
 * permission is reported as "blocked" (the text explains where to allow it again).
 */
export function geolocationFailure(code: number): WebGeoFailure {
  if (code === 1) return "blocked";
  if (code === 2) return "services-off";
  return "error";
}
