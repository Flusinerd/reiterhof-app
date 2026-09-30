import * as Location from "expo-location";
import * as SecureStore from "expo-secure-store";
import * as TaskManager from "expo-task-manager";
import { useCallback, useEffect, useState } from "react";
import { Platform } from "react-native";

import { presenceApi } from "./api/presence";
import { actionFor, regionFor, type GeofenceFailure, type GeofenceStable, type RegionEvent } from "./geofence-core";
import { loadToken } from "./token";

/**
 * Automatic check-in/out when the phone enters/leaves the stable (JAN-27).
 *
 * - Opt-in per device: the switch state is stored locally (secure store), never on the server.
 * - The OS monitors the region (iOS: CLRegion, Android: Geofencing API); the app is woken up
 *   for enter/exit and runs the task below, which calls the presence API with source "geofence".
 * - Everything here is a no-op on web and where the task manager is unavailable.
 * - The location itself never leaves the device; the server only sees the check-in.
 * - The privacy consent (JAN-19) comes later; until then the screen says so.
 *
 * Importing this module (root layout) registers the background task; that has to happen at
 * app start, in the global scope, so the OS can run it while the app is closed.
 */

export const GEOFENCE_TASK = "reiterhof-geofence";
const STORAGE_KEY = "reiterhof.geofence.enabled";

const nativeOnly = Platform.OS === "ios" || Platform.OS === "android";

if (nativeOnly) {
  TaskManager.defineTask<{ eventType: Location.GeofencingEventType; region: Location.LocationRegion }>(
    GEOFENCE_TASK,
    async ({ data, error }) => {
      if (error || !data) return;
      const event: RegionEvent | null =
        data.eventType === Location.GeofencingEventType.Enter
          ? "enter"
          : data.eventType === Location.GeofencingEventType.Exit
            ? "exit"
            : null;
      const action = actionFor(event);
      if (!action) return;
      try {
        // The task can run in a fresh JS context: read the session token first.
        if (!(await loadToken())) return;
        if (action === "check-in") await presenceApi.checkIn("geofence");
        else await presenceApi.checkOut();
      } catch {
        // offline or signed out: the next event or the manual button catches up
      }
    },
  );
}

export async function isGeofenceSupported(): Promise<boolean> {
  if (!nativeOnly) return false;
  try {
    return await TaskManager.isAvailableAsync();
  } catch {
    return false;
  }
}

export async function isGeofenceEnabled(): Promise<boolean> {
  try {
    return (await SecureStore.getItemAsync(STORAGE_KEY)) === "1";
  } catch {
    return false;
  }
}

async function storeEnabled(on: boolean): Promise<void> {
  try {
    if (on) await SecureStore.setItemAsync(STORAGE_KEY, "1");
    else await SecureStore.deleteItemAsync(STORAGE_KEY);
  } catch {
    // not persisted: the switch shows off again after a restart
  }
}

export type EnableResult = { ok: true } | { ok: false; reason: GeofenceFailure };

/** Asks for the permissions (foreground, then background) and starts monitoring. */
export async function enableGeofence(stable: GeofenceStable | null | undefined): Promise<EnableResult> {
  if (!(await isGeofenceSupported())) return { ok: false, reason: "unsupported" };
  const region = regionFor(stable);
  if (!region) return { ok: false, reason: "no_location" };
  try {
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status !== "granted") return { ok: false, reason: "foreground_denied" };
    const bg = await Location.requestBackgroundPermissionsAsync();
    if (bg.status !== "granted") return { ok: false, reason: "background_denied" };
    await Location.startGeofencingAsync(GEOFENCE_TASK, [region]);
    await storeEnabled(true);
    return { ok: true };
  } catch {
    return { ok: false, reason: "error" };
  }
}

export async function disableGeofence(): Promise<void> {
  await storeEnabled(false);
  if (!(await isGeofenceSupported())) return;
  try {
    if (await Location.hasStartedGeofencingAsync(GEOFENCE_TASK)) await Location.stopGeofencingAsync(GEOFENCE_TASK);
  } catch {
    // already stopped
  }
}

/** Re-registers the region after app start (position or radius may have changed). Silent. */
export async function syncGeofence(stable: GeofenceStable | null | undefined): Promise<void> {
  if (!(await isGeofenceSupported()) || !(await isGeofenceEnabled())) return;
  const region = regionFor(stable);
  try {
    const bg = await Location.getBackgroundPermissionsAsync();
    if (!region || bg.status !== "granted") return;
    await Location.startGeofencingAsync(GEOFENCE_TASK, [region]);
  } catch {
    // ignore: the switch stays on, the next start tries again
  }
}

/** State of the geofence switch on this device. */
export function useGeofence(stable: GeofenceStable | null | undefined) {
  const [supported, setSupported] = useState(false);
  const [enabled, setEnabled] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<GeofenceFailure | null>(null);

  useEffect(() => {
    let alive = true;
    void (async () => {
      const [s, e] = await Promise.all([isGeofenceSupported(), isGeofenceEnabled()]);
      if (!alive) return;
      setSupported(s);
      setEnabled(s && e);
    })();
    return () => {
      alive = false;
    };
  }, []);

  const toggle = useCallback(
    async (on: boolean) => {
      setBusy(true);
      setFailure(null);
      try {
        if (on) {
          const res = await enableGeofence(stable);
          setEnabled(res.ok);
          if (!res.ok) setFailure(res.reason);
        } else {
          await disableGeofence();
          setEnabled(false);
        }
      } finally {
        setBusy(false);
      }
    },
    [stable],
  );

  return { supported, enabled, busy, failure, toggle };
}
