import * as Location from "expo-location";
import * as TaskManager from "expo-task-manager";
import { Platform } from "react-native";

import type { RawFix } from "./tracking";
import { currentFixListener, type StartResult } from "./tracking-location-shared";
import { appendPendingFixes } from "./tracking-store";

export { setFixListener, startFailureText, type StartResult } from "./tracking-location-shared";

/**
 * GPS updates for the ride tracker (JAN-63): one `expo-location` background task that also
 * delivers in the foreground. It is separate from the geofence of the check-in
 * (`lib/geofence.ts`, task `reiterhof-geofence`): different task name, different API
 * (`startLocationUpdatesAsync` vs. geofencing), and it only runs while a ride is tracked.
 *
 * Fixes go to the listener of the tracking screen. When no screen listens (the OS restarted
 * the app process for the task) they are buffered on disk and picked up when the session is
 * restored. Importing this module registers the task in the global scope, which the OS needs
 * to be able to run it while the app is not in the foreground.
 */

export const TRACKING_TASK = "reiterhof-tracking";

const nativeOnly = Platform.OS === "ios" || Platform.OS === "android";

export function toRawFix(l: Location.LocationObject): RawFix {
  return {
    t: l.timestamp,
    lat: l.coords.latitude,
    lon: l.coords.longitude,
    alt: l.coords.altitude,
    speed: l.coords.speed,
    accuracy: l.coords.accuracy,
  };
}

if (nativeOnly) {
  TaskManager.defineTask<{ locations: Location.LocationObject[] }>(TRACKING_TASK, async ({ data, error }) => {
    if (error || !data?.locations?.length) return;
    const fixes = data.locations.map(toRawFix);
    const listener = currentFixListener();
    if (listener) listener(fixes);
    else await appendPendingFixes(fixes);
  });
}

/**
 * Asks for the permissions and starts the updates. Foreground permission is required;
 * background permission is asked for as well, and without it tracking still works while
 * the app stays open (`background: false`).
 */
export async function startTracking(): Promise<StartResult> {
  if (!nativeOnly) return { ok: false, reason: "unsupported" };
  try {
    if (!(await Location.hasServicesEnabledAsync())) return { ok: false, reason: "services-off" };
    const fg = await Location.requestForegroundPermissionsAsync();
    if (fg.status !== "granted") return { ok: false, reason: fg.canAskAgain ? "denied" : "blocked" };
    let background = false;
    try {
      background = (await Location.requestBackgroundPermissionsAsync()).status === "granted";
    } catch {
      background = false;
    }
    if (await Location.hasStartedLocationUpdatesAsync(TRACKING_TASK)) {
      await Location.stopLocationUpdatesAsync(TRACKING_TASK);
    }
    await Location.startLocationUpdatesAsync(TRACKING_TASK, {
      accuracy: Location.Accuracy.BestForNavigation,
      timeInterval: 1000,
      distanceInterval: 2,
      activityType: Location.ActivityType.Fitness,
      pausesUpdatesAutomatically: false,
      showsBackgroundLocationIndicator: true,
      foregroundService: {
        notificationTitle: "Stallfunk",
        notificationBody: "Dein Ausritt wird aufgezeichnet.",
      },
    });
    return { ok: true, background };
  } catch {
    return { ok: false, reason: "error" };
  }
}

export async function stopTracking(): Promise<void> {
  if (!nativeOnly) return;
  try {
    if (await Location.hasStartedLocationUpdatesAsync(TRACKING_TASK)) {
      await Location.stopLocationUpdatesAsync(TRACKING_TASK);
    }
  } catch {
    // already stopped
  }
}
