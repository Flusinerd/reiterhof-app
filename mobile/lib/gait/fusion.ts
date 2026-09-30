import type { Classification, Gait } from "./types.ts";

/** Speed limits between the gaits, km/h (outdoors, from the spec). */
export const SPEED_HALT_MAX = 1.5;
export const SPEED_WALK_MAX = 7;
export const SPEED_TROT_MAX = 16;

/** Distance from a class boundary (km/h) at which the speed is fully trusted. */
const FULL_TRUST_KMH = 3;

/** Gait implied by GPS speed alone: <1.5 halt, <7 walk, 7-16 trot, >16 canter. */
export function gaitFromSpeed(speedKmh: number): Gait {
  if (speedKmh < SPEED_HALT_MAX) return "halt";
  if (speedKmh < SPEED_WALK_MAX) return "walk";
  if (speedKmh <= SPEED_TROT_MAX) return "trot";
  return "canter";
}

/** How much the speed can be trusted for its class: 0 on a boundary, 1 deep inside. */
export function speedConfidence(speedKmh: number): number {
  const gait = gaitFromSpeed(speedKmh);
  let d: number;
  switch (gait) {
    case "halt":
      // Speed 0 is the safest evidence for standing still.
      return Math.min(1, (SPEED_HALT_MAX - speedKmh) / SPEED_HALT_MAX);
    case "walk":
      d = Math.min(speedKmh - SPEED_HALT_MAX, SPEED_WALK_MAX - speedKmh);
      break;
    case "trot":
      d = Math.min(speedKmh - SPEED_WALK_MAX, SPEED_TROT_MAX - speedKmh);
      break;
    default:
      d = speedKmh - SPEED_TROT_MAX;
  }
  return Math.max(0, Math.min(1, d / FULL_TRUST_KMH));
}

/**
 * Fusion rule (outdoors, with a GPS speed):
 * - No usable speed (undefined, NaN, negative): the accelerometer decides.
 * - Both sources agree: that gait.
 * - Otherwise the source with the higher confidence wins. The accelerometer
 *   confidence is the classifier margin (0..1); the speed confidence grows
 *   linearly with the distance to the class boundary and is 1 from 3 km/h
 *   inside the class. Ties go to the accelerometer.
 *
 * So speed wins when the accelerometer is ambiguous (e.g. the trot/canter
 * overlap, or a frequency near a boundary) and the speed is clearly inside a
 * class, but a confident accelerometer result is kept when the speed sits
 * close to a boundary or the GPS fix is wrong.
 */
export function fuseWithSpeed(accel: Classification, speedKmh: number | undefined): Gait {
  if (speedKmh === undefined || !Number.isFinite(speedKmh) || speedKmh < 0) return accel.gait;
  const fromSpeed = gaitFromSpeed(speedKmh);
  if (fromSpeed === accel.gait) return accel.gait;
  return speedConfidence(speedKmh) > accel.confidence ? fromSpeed : accel.gait;
}
