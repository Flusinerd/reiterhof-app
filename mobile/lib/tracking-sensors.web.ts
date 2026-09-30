import { accelerationToG } from "./web-sensors-core";

/**
 * Web variant of the ride tracker's sensors (same exports as `tracking-sensors.ts`).
 *
 * - Accelerometer: `devicemotion` events (`accelerationIncludingGravity`, m/s²) converted to g and
 *   thinned to the requested interval. iOS Safari (13+) only delivers them after
 *   `DeviceMotionEvent.requestPermission()`, which must be called from a user gesture: the session
 *   hook calls `requestMotionAccess()` first thing inside the tap on "Start". The page must be
 *   served over HTTPS.
 * - Screen Wake Lock: keeps the screen on while recording. A PWA is suspended when the screen
 *   locks, so recording only works with the screen on; the lock is released by the browser when the
 *   page is hidden and requested again when it becomes visible.
 */

export type SensorSubscription = { remove: () => void };

type MotionEventCtor = { requestPermission?: () => Promise<"granted" | "denied"> };

export async function requestMotionAccess(): Promise<boolean> {
  if (typeof window === "undefined" || !("DeviceMotionEvent" in window)) return false;
  const ctor = window.DeviceMotionEvent as unknown as MotionEventCtor;
  if (typeof ctor.requestPermission !== "function") return true; // Android, desktop: no prompt
  try {
    return (await ctor.requestPermission()) === "granted";
  } catch {
    return false; // not called from a gesture, or blocked
  }
}

export async function isAccelerometerAvailable(): Promise<boolean> {
  return typeof window !== "undefined" && "DeviceMotionEvent" in window;
}

export function subscribeAccelerometer(
  onSample: (sample: { x: number; y: number; z: number }) => void,
  intervalMs: number,
): SensorSubscription {
  let last = 0;
  const listener = (event: DeviceMotionEvent) => {
    const now = Date.now();
    if (now - last < intervalMs * 0.8) return; // browsers deliver 60 Hz; the detector wants ~50 Hz
    const g = accelerationToG(event.accelerationIncludingGravity);
    if (!g) return;
    last = now;
    onSample(g);
  };
  window.addEventListener("devicemotion", listener);
  return { remove: () => window.removeEventListener("devicemotion", listener) };
}

// --- Screen Wake Lock ---------------------------------------------------------------------------

type Sentinel = { release: () => Promise<void> };
const wanted = new Set<string>();
let sentinel: Sentinel | null = null;
let listening = false;

async function acquire(): Promise<void> {
  if (wanted.size === 0 || sentinel || typeof navigator === "undefined") return;
  const wakeLock = (navigator as unknown as { wakeLock?: { request: (t: "screen") => Promise<Sentinel> } }).wakeLock;
  if (!wakeLock) return;
  try {
    sentinel = await wakeLock.request("screen");
  } catch {
    sentinel = null; // denied (battery saver, hidden page): recording still works, the screen may dim
  }
}

function onVisibility(): void {
  if (document.visibilityState === "visible") {
    sentinel = null; // the browser released the old one when the page was hidden
    void acquire();
  }
}

export async function keepScreenAwake(tag: string): Promise<void> {
  wanted.add(tag);
  if (!listening) {
    listening = true;
    document.addEventListener("visibilitychange", onVisibility);
  }
  await acquire();
}

export async function releaseScreenAwake(tag: string): Promise<void> {
  wanted.delete(tag);
  if (wanted.size > 0) return;
  if (listening) {
    listening = false;
    document.removeEventListener("visibilitychange", onVisibility);
  }
  const s = sentinel;
  sentinel = null;
  try {
    await s?.release();
  } catch {
    // already released
  }
}
