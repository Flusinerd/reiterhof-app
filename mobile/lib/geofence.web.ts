import type { GeofenceFailure, GeofenceStable } from "./geofence-core";

/**
 * Web stub of `lib/geofence.ts`. A browser cannot watch a region in the background, so there
 * is no automatic check-in/out on web: the presence screen shows only the manual buttons and
 * a note (`app/presence/index.tsx`). Same exports as the native module, all "unsupported".
 */

export const GEOFENCE_TASK = "reiterhof-geofence";

export async function isGeofenceSupported(): Promise<boolean> {
  return false;
}

export async function isGeofenceEnabled(): Promise<boolean> {
  return false;
}

export type EnableResult = { ok: true } | { ok: false; reason: GeofenceFailure };

export async function enableGeofence(_stable: GeofenceStable | null | undefined): Promise<EnableResult> {
  return { ok: false, reason: "unsupported" };
}

export async function disableGeofence(): Promise<void> {}

export async function syncGeofence(_stable: GeofenceStable | null | undefined): Promise<void> {}

/** Same shape as the native hook; `supported` is always false. */
export function useGeofence(_stable: GeofenceStable | null | undefined) {
  return {
    supported: false,
    enabled: false,
    busy: false,
    failure: null as GeofenceFailure | null,
    toggle: async (_on: boolean): Promise<void> => {},
  };
}
