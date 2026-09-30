import type { Gait, GaitWindow } from "./types.ts";

/** Order of the numbers in `WindowRecord.f`. */
export const FEATURE_ORDER = [
  "freq",
  "verticalEnergy",
  "horizontalEnergy",
  "asymmetry",
  "harmonicRatio",
  "regularity",
] as const;

/**
 * Compact record of one window for storage and later model training.
 * A 4 s window with all fields is roughly 100 bytes as JSON.
 */
export type WindowRecord = {
  /** Window start (ms). */
  t: number;
  /** Features in `FEATURE_ORDER`, rounded to 3 decimals. */
  f: number[];
  /** Predicted gait (after fusion and smoothing). */
  p: Gait;
  /** Accelerometer-only prediction. */
  a: Gait;
  /** GPS speed in km/h, if available (1 decimal). */
  v?: number;
  /** Gait corrected by the user; the training label when present. */
  c?: Gait;
};

function round(x: number, digits: number): number {
  const m = 10 ** digits;
  return Math.round(x * m) / m;
}

export function exportWindows(windows: readonly GaitWindow[]): WindowRecord[] {
  return windows.map((w) => {
    const r: WindowRecord = {
      t: w.startMs,
      f: FEATURE_ORDER.map((k) => round(w.features[k], 3)),
      p: w.gait,
      a: w.accelGait,
    };
    if (w.speedKmh !== undefined) r.v = round(w.speedKmh, 1);
    if (w.label !== undefined) r.c = w.label;
    return r;
  });
}

/** Sets the corrected label on all records starting in [fromMs, toMs). Returns new records. */
export function correctRecords(
  records: readonly WindowRecord[],
  fromMs: number,
  toMs: number,
  label: Gait,
): WindowRecord[] {
  return records.map((r) => (r.t >= fromMs && r.t < toMs ? { ...r, c: label } : r));
}

/** One JSON object per line, ready to append to a file or upload. */
export function toJsonLines(records: readonly WindowRecord[]): string {
  return records.map((r) => JSON.stringify(r)).join("\n");
}
