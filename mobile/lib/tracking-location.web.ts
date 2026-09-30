import { currentFixListener, startFailureText as nativeText, type StartResult } from "./tracking-location-shared";
import { fixFromPosition, geolocationFailure } from "./web-sensors-core";

export { setFixListener, type StartResult } from "./tracking-location-shared";

/**
 * Web variant of the ride tracker's location module: `navigator.geolocation.watchPosition`.
 *
 * A browser tab can only receive positions while the page is visible and the screen is on
 * (iOS suspends a PWA when the screen locks), so `background` is always false and the GPS
 * screen tells the rider to keep the screen on (`lib/tracking-sensors.web.ts` requests a
 * Screen Wake Lock). Fixes go straight to the listener of the tracking screen; nothing is
 * buffered while no screen listens.
 */

export const TRACKING_TASK = "reiterhof-tracking";

/** How long `startTracking` waits for the first fix before it reports success anyway. */
const FIRST_FIX_WAIT_MS = 20_000;

let watchId: number | null = null;

function clearWatch(): void {
  if (watchId !== null && typeof navigator !== "undefined" && navigator.geolocation) {
    navigator.geolocation.clearWatch(watchId);
  }
  watchId = null;
}

async function permissionState(): Promise<PermissionState | null> {
  try {
    if (!navigator.permissions?.query) return null;
    return (await navigator.permissions.query({ name: "geolocation" })).state;
  } catch {
    return null;
  }
}

/**
 * Starts the watch. Resolves when the first fix arrived, the browser refused, or (with the
 * permission already granted) at once. While the permission prompt is open it waits for the answer.
 */
export async function startTracking(): Promise<StartResult> {
  if (typeof navigator === "undefined" || !navigator.geolocation) return { ok: false, reason: "unsupported" };
  const state = await permissionState();
  if (state === "denied") return { ok: false, reason: "blocked" };
  clearWatch();
  return new Promise<StartResult>((resolve) => {
    let settled = false;
    const settle = (result: StartResult) => {
      if (settled) return;
      settled = true;
      clearTimeout(timer);
      resolve(result);
    };
    const timer = setTimeout(() => settle({ ok: true, background: false }), FIRST_FIX_WAIT_MS);
    try {
      watchId = navigator.geolocation.watchPosition(
        (position) => {
          currentFixListener()?.([fixFromPosition(position)]);
          settle({ ok: true, background: false });
        },
        (error) => {
          const reason = geolocationFailure(error.code);
          // A timeout between fixes is normal; only a refusal at the start is an error.
          if (error.code !== 3) settle({ ok: false, reason });
          if (error.code === 1) clearWatch();
        },
        { enableHighAccuracy: true, maximumAge: 0, timeout: 30_000 },
      );
    } catch {
      settle({ ok: false, reason: "error" });
      return;
    }
    if (state === "granted") settle({ ok: true, background: false });
  });
}

export async function stopTracking(): Promise<void> {
  clearWatch();
}

/** Same texts as native, except that a blocked permission is allowed again in the browser settings. */
export function startFailureText(reason: Exclude<StartResult, { ok: true }>["reason"]): string {
  if (reason === "blocked") {
    return "Der Standortzugriff ist blockiert. Erlaube ihn in den iPhone-Einstellungen unter „Datenschutz & Sicherheit“ und „Ortungsdienste“ für Safari-Websites bzw. Stallfunk und lade die Seite neu.";
  }
  return nativeText(reason);
}
