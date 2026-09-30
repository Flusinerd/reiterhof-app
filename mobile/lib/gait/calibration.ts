import { DEFAULT_CALIBRATION, asymmetryScore } from "./classify.ts";
import { SAMPLE_RATE, extractFeatures } from "./features.ts";
import type { Calibration, Features, Gait, Sample } from "./types.ts";

/**
 * Typical vertical RMS (g) per gait for the default placement. Taken from the
 * synthetic signal model, so treat them as an assumption until real data exists.
 */
export const REFERENCE_ENERGY: Record<Exclude<Gait, "halt">, number> = {
  walk: 0.1,
  trot: 0.39,
  canter: 0.35,
};

/** A short recording with a known gait, e.g. 20-30 s of walking. */
export type LabeledSegment = { gait: Gait; samples: readonly Sample[] };

function quantile(values: number[], q: number): number {
  const v = [...values].sort((a, b) => a - b);
  const i = (v.length - 1) * q;
  const lo = Math.floor(i);
  const hi = Math.ceil(i);
  return v[lo]! + (v[hi]! - v[lo]!) * (i - lo);
}

function clamp(x: number, lo: number, hi: number): number {
  return Math.max(lo, Math.min(hi, x));
}

/**
 * Learns a calibration from short labelled recordings made with the phone in
 * its usual place (pocket, arm, ...). Segments of any subset of gaits work;
 * whatever cannot be estimated keeps the value of `base`.
 *
 * - `energyScale`: median of (measured / reference vertical energy) over the
 *   moving gaits provided.
 * - `haltEnergy`: geometric mean between the 95th percentile of the halt
 *   energies and the 5th percentile of the moving energies (needs a halt
 *   segment and a moving one); otherwise the default, which scales with
 *   `energyScale`.
 * - `walkTrotFreq`: midpoint of the median walk and trot frequency.
 * - `asymmetryThreshold`: midpoint of the median trot and canter shape scores.
 *
 * Throws if no segment yields a single analysable window.
 */
export function learnCalibration(
  segments: readonly LabeledSegment[],
  base: Calibration = DEFAULT_CALIBRATION,
  windowMs = 4000,
  hopMs = 2000,
): Calibration {
  const byGait = new Map<Gait, Features[]>();
  for (const seg of segments) {
    const list = byGait.get(seg.gait) ?? [];
    const first = seg.samples[0]?.t ?? 0;
    const last = seg.samples[seg.samples.length - 1]?.t ?? 0;
    for (let start = first; start + windowMs <= last; start += hopMs) {
      const f = extractFeatures(seg.samples, start, windowMs, SAMPLE_RATE);
      if (f) list.push(f);
    }
    byGait.set(seg.gait, list);
  }
  const total = [...byGait.values()].reduce((n, l) => n + l.length, 0);
  if (total === 0) throw new Error("learnCalibration: no analysable window in the segments");

  const med = (g: Gait, pick: (f: Features) => number): number | undefined => {
    const l = byGait.get(g);
    return l && l.length > 0 ? quantile(l.map(pick), 0.5) : undefined;
  };

  const cal: Calibration = { ...base };

  const ratios: number[] = [];
  for (const g of ["walk", "trot", "canter"] as const) {
    const e = med(g, (f) => f.verticalEnergy);
    if (e !== undefined) ratios.push(e / REFERENCE_ENERGY[g]);
  }
  if (ratios.length > 0) cal.energyScale = clamp(quantile(ratios, 0.5), 0.1, 5);

  const halt = byGait.get("halt") ?? [];
  const moving = (["walk", "trot", "canter"] as const).flatMap((g) => byGait.get(g) ?? []);
  if (halt.length > 0 && moving.length > 0) {
    const haltTop = quantile(halt.map((f) => f.verticalEnergy), 0.95);
    const movingLow = quantile(moving.map((f) => f.verticalEnergy), 0.05);
    if (movingLow > haltTop) {
      cal.haltEnergy = Math.sqrt(haltTop * movingLow) / cal.energyScale;
    }
  }

  const walkF = med("walk", (f) => f.freq);
  const trotF = med("trot", (f) => f.freq);
  if (walkF !== undefined && trotF !== undefined && trotF > walkF) {
    cal.walkTrotFreq = clamp((walkF + trotF) / 2, 1.1, 1.4);
  }
  const trotA = med("trot", asymmetryScore);
  const canterA = med("canter", asymmetryScore);
  if (trotA !== undefined && canterA !== undefined && canterA > trotA) {
    cal.asymmetryThreshold = clamp((trotA + canterA) / 2, 0.15, 0.8);
  }
  return cal;
}
