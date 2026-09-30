// Formatting for the tracking screens (German UI texts). Pure, unit-tested.

import type { Gait } from "./gait/types.ts";

const GAIT_LABELS: Record<Gait, string> = { halt: "Halt", walk: "Schritt", trot: "Trab", canter: "Galopp" };

export function gaitLabel(gait: Gait): string {
  return GAIT_LABELS[gait];
}

const de = (value: number, digits: number) => value.toFixed(digits).replace(".", ",");

/** "7,42 km" from metres; under a kilometre "850 m". */
export function formatDistance(meters: number): string {
  if (meters < 1000) return `${Math.round(meters)} m`;
  return `${de(meters / 1000, 2)} km`;
}

/** "6,8 km/h". */
export function formatSpeed(kmh: number): string {
  return `${de(kmh, 1)} km/h`;
}

/** "+42 m". */
export function formatElevation(meters: number): string {
  return `+${Math.round(meters)} m`;
}

/** Whole minutes on the current rein: "12 Min.". */
export function formatReinMinutes(minutes: number): string {
  return `${Math.floor(minutes)} Min.`;
}

export function reinLabel(rein: "left" | "right"): string {
  return rein === "left" ? "Linke Hand" : "Rechte Hand";
}
