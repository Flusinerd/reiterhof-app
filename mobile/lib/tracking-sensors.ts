import { activateKeepAwakeAsync, deactivateKeepAwake } from "expo-keep-awake";
import { Accelerometer } from "expo-sensors";

/**
 * Native sensor access of the ride tracker: the accelerometer (expo-sensors, g units) and
 * keep-awake. The web variant (`tracking-sensors.web.ts`) uses DeviceMotion and the Screen
 * Wake Lock and has the same exports.
 */

export type SensorSubscription = { remove: () => void };

/**
 * Asks for motion access. Native platforms need no permission for the accelerometer.
 * On web (iOS Safari) this must be called synchronously from a tap.
 */
export async function requestMotionAccess(): Promise<boolean> {
  return true;
}

export async function isAccelerometerAvailable(): Promise<boolean> {
  return Accelerometer.isAvailableAsync();
}

/** Calls `onSample` with acceleration in g, about every `intervalMs`. */
export function subscribeAccelerometer(
  onSample: (sample: { x: number; y: number; z: number }) => void,
  intervalMs: number,
): SensorSubscription {
  Accelerometer.setUpdateInterval(intervalMs);
  return Accelerometer.addListener(({ x, y, z }) => onSample({ x, y, z }));
}

export async function keepScreenAwake(tag: string): Promise<void> {
  await activateKeepAwakeAsync(tag);
}

export async function releaseScreenAwake(tag: string): Promise<void> {
  await deactivateKeepAwake(tag);
}
