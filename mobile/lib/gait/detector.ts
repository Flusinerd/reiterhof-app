import { DEFAULT_CALIBRATION, classifyWindow } from "./classify.ts";
import { SAMPLE_RATE, extractFeatures } from "./features.ts";
import { fuseWithSpeed } from "./fusion.ts";
import type { Calibration, Gait, GaitWindow, GpsFix, Sample } from "./types.ts";

export type DetectorOptions = {
  /** Window length in ms (default 4000). */
  windowMs?: number;
  /** Distance between window starts in ms (default 2000 = 50 % overlap). */
  hopMs?: number;
  /** Sample rate used to resample each window (default 50 Hz). */
  sampleRate?: number;
  /** Number of windows in the majority vote (default 3, 1 = no smoothing). */
  smoothing?: number;
  calibration?: Calibration;
};

/**
 * Majority vote over the last `n` gaits (the vote includes the newest one).
 * Ties go to the tied gait that occurred most recently.
 */
export function majorityVote(history: readonly Gait[], n: number): Gait {
  const recent = history.slice(-Math.max(1, n));
  const counts = new Map<Gait, number>();
  for (const g of recent) counts.set(g, (counts.get(g) ?? 0) + 1);
  let best = recent[recent.length - 1]!;
  let bestCount = counts.get(best)!;
  for (let i = recent.length - 1; i >= 0; i--) {
    const c = counts.get(recent[i]!)!;
    if (c > bestCount) {
      best = recent[i]!;
      bestCount = c;
    }
  }
  return best;
}

/**
 * Streaming detector: feed samples (and GPS fixes) in time order, get finished
 * windows back. Windows the samples do not cover (sensor dropout) are skipped.
 */
export class GaitStream {
  readonly windows: GaitWindow[] = [];
  private readonly windowMs: number;
  private readonly hopMs: number;
  private readonly rate: number;
  private readonly smoothing: number;
  private readonly cal: Calibration;
  private samples: Sample[] = [];
  private gps: GpsFix[] = [];
  private nextStart: number | null = null;
  private history: Gait[] = [];

  constructor(options: DetectorOptions = {}) {
    this.windowMs = options.windowMs ?? 4000;
    this.hopMs = options.hopMs ?? 2000;
    this.rate = options.sampleRate ?? SAMPLE_RATE;
    this.smoothing = options.smoothing ?? 3;
    this.cal = options.calibration ?? DEFAULT_CALIBRATION;
  }

  /** Adds a GPS fix. Call before the samples of the same instant. */
  pushGps(fix: GpsFix): void {
    this.gps.push(fix);
  }

  /** Adds a sample and returns the windows completed by it. */
  push(sample: Sample): GaitWindow[] {
    this.samples.push(sample);
    if (this.nextStart === null) this.nextStart = sample.t;
    const done: GaitWindow[] = [];
    while (this.nextStart !== null && sample.t >= this.nextStart + this.windowMs) {
      const first = this.samples[0];
      if (first && first.t > this.nextStart + this.hopMs) {
        // Long dropout: jump to the data instead of iterating over empty windows.
        this.nextStart = first.t;
        continue;
      }
      const w = this.analyse(this.nextStart);
      if (w) {
        this.windows.push(w);
        done.push(w);
      }
      this.nextStart += this.hopMs;
      this.trim();
    }
    return done;
  }

  private speedFor(start: number, end: number): number | undefined {
    const inside = this.gps.filter((g) => g.t >= start && g.t <= end);
    if (inside.length > 0) {
      return inside.reduce((s, g) => s + g.speedKmh, 0) / inside.length;
    }
    // GPS fixes arrive at about 1 Hz; accept a recent one when none fell in the window.
    let last: GpsFix | undefined;
    for (const g of this.gps) if (g.t < start && g.t >= start - 3000) last = g;
    return last?.speedKmh;
  }

  private analyse(start: number): GaitWindow | null {
    const features = extractFeatures(this.samples, start, this.windowMs, this.rate);
    if (!features) return null;
    const end = start + this.windowMs;
    const cls = classifyWindow(features, this.cal);
    const speedKmh = this.speedFor(start, end);
    const fusedGait = fuseWithSpeed(cls, speedKmh);
    this.history.push(fusedGait);
    if (this.history.length > 64) this.history.shift();
    const w: GaitWindow = {
      startMs: start,
      endMs: end,
      weightMs: this.hopMs,
      features,
      accelGait: cls.gait,
      accelConfidence: cls.confidence,
      fusedGait,
      gait: majorityVote(this.history, this.smoothing),
    };
    if (speedKmh !== undefined) w.speedKmh = speedKmh;
    return w;
  }

  private trim(): void {
    const keepFrom = (this.nextStart ?? 0) - 200;
    let i = 0;
    while (i < this.samples.length && this.samples[i]!.t < keepFrom) i++;
    if (i > 0) this.samples = this.samples.slice(i);
    let j = 0;
    while (j < this.gps.length && this.gps[j]!.t < keepFrom - 5000) j++;
    if (j > 0) this.gps = this.gps.slice(j);
  }
}

/** Analyses a whole recording (samples and GPS fixes sorted by time). */
export function analyzeWindows(
  samples: readonly Sample[],
  gps: readonly GpsFix[] = [],
  options: DetectorOptions = {},
): GaitWindow[] {
  const stream = new GaitStream(options);
  let g = 0;
  for (const s of samples) {
    while (g < gps.length && gps[g]!.t <= s.t) stream.pushGps(gps[g++]!);
    stream.push(s);
  }
  return stream.windows;
}
