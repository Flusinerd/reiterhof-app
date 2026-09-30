// Synthetic accelerometer signals for tests and offline experiments.
//
// The model is an assumption, not measured data: sinusoids with harmonics,
// slow frequency/amplitude modulation, sensor noise, a random phone
// orientation and slightly irregular timestamps. See README.md for limits.

import type { Gait, Sample } from "./types.ts";

/** Deterministic PRNG (mulberry32). */
export function makeRng(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

function gauss(rng: () => number): number {
  const u = Math.max(1e-12, rng());
  return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * rng());
}

export type SynthOptions = {
  gait: Gait;
  durationS: number;
  seed: number;
  /** First timestamp in ms. */
  startMs?: number;
  /** Nominal rate in Hz. */
  rate?: number;
  /** Amplitude factor for the sensor placement (1 = default, ~0.3 = on the arm). */
  placementScale?: number;
  /** Sensor noise standard deviation in g. */
  noise?: number;
};

type Params = { f: [number, number]; a1: [number, number]; h2: [number, number]; lateral: number };

// Ranges follow the spec: walk 0.8-1.2 Hz, trot 1.3-1.8 Hz, canter 1.6-2.2 Hz.
// The canter has a strong second harmonic (asymmetric waveform within one stride).
const PARAMS: Record<Exclude<Gait, "halt">, Params> = {
  walk: { f: [0.85, 1.15], a1: [0.1, 0.18], h2: [0.15, 0.35], lateral: 0.6 },
  trot: { f: [1.35, 1.75], a1: [0.4, 0.7], h2: [0.05, 0.18], lateral: 0.3 },
  canter: { f: [1.6, 2.2], a1: [0.3, 0.55], h2: [0.5, 0.75], lateral: 0.4 },
};

function uniform(rng: () => number, [lo, hi]: [number, number]): number {
  return lo + (hi - lo) * rng();
}

export function synthesize(opts: SynthOptions): Sample[] {
  const rate = opts.rate ?? 50;
  const noise = opts.noise ?? 0.02;
  const scale = opts.placementScale ?? 1;
  const start = opts.startMs ?? 0;
  const rng = makeRng(opts.seed);

  // Random phone orientation: gravity direction plus two perpendicular axes.
  let g: [number, number, number] = [gauss(rng), gauss(rng), gauss(rng)];
  const gn = Math.hypot(...g);
  g = [g[0] / gn, g[1] / gn, g[2] / gn];
  const helper: [number, number, number] = Math.abs(g[2]) < 0.9 ? [0, 0, 1] : [1, 0, 0];
  let u1: [number, number, number] = [
    g[1] * helper[2] - g[2] * helper[1],
    g[2] * helper[0] - g[0] * helper[2],
    g[0] * helper[1] - g[1] * helper[0],
  ];
  const un = Math.hypot(...u1);
  u1 = [u1[0] / un, u1[1] / un, u1[2] / un];
  const u2: [number, number, number] = [
    g[1] * u1[2] - g[2] * u1[1],
    g[2] * u1[0] - g[0] * u1[2],
    g[0] * u1[1] - g[1] * u1[0],
  ];

  let f0 = 0;
  let a1 = 0;
  let a2 = 0;
  let lat = 0;
  if (opts.gait !== "halt") {
    const p = PARAMS[opts.gait];
    f0 = uniform(rng, p.f);
    a1 = uniform(rng, p.a1) * scale;
    a2 = a1 * uniform(rng, p.h2);
    lat = a1 * p.lateral;
  }
  const phase = rng() * 2 * Math.PI;
  const phase2 = rng() * 2 * Math.PI;
  const modPhase = rng() * 2 * Math.PI;

  const n = Math.round(opts.durationS * rate);
  const out: Sample[] = [];
  let theta = phase;
  let tPrev = 0;
  for (let i = 0; i < n; i++) {
    const t = i / rate + (rng() - 0.5) * 0.004; // +-2 ms timestamp jitter
    const dt = Math.max(0, t - tPrev);
    tPrev = t;
    // Slow frequency and amplitude modulation, like a real horse and rider.
    const f = f0 * (1 + 0.03 * Math.sin(2 * Math.PI * 0.1 * t + modPhase));
    theta += 2 * Math.PI * f * dt;
    const am = 1 + 0.1 * Math.sin(2 * Math.PI * 0.15 * t + modPhase);
    let vert = 0;
    let side = 0;
    if (opts.gait === "halt") {
      // Breathing/shifting weight and a hand-held wobble.
      vert = 0.008 * Math.sin(2 * Math.PI * 0.3 * t + modPhase);
      side = 0.006 * Math.sin(2 * Math.PI * 0.2 * t);
    } else {
      vert = am * (a1 * Math.sin(theta) + a2 * Math.sin(2 * theta + phase2));
      side = am * lat * Math.sin(theta / 2 + phase2);
    }
    const back = 0.2 * side * Math.sin(theta);
    out.push({
      t: Math.round((start + t * 1000) * 100) / 100,
      x: g[0] * (1 + vert) + u1[0] * side + u2[0] * back + noise * gauss(rng),
      y: g[1] * (1 + vert) + u1[1] * side + u2[1] * back + noise * gauss(rng),
      z: g[2] * (1 + vert) + u1[2] * side + u2[2] * back + noise * gauss(rng),
    });
  }
  return out;
}

/** Concatenates segments in time: [[gait, seconds], ...] with per-segment seeds. */
export function synthesizeSequence(
  segments: readonly (readonly [Gait, number])[],
  seed: number,
  extra: Partial<SynthOptions> = {},
): Sample[] {
  const out: Sample[] = [];
  let start = 0;
  for (const [gait, seconds] of segments) {
    // Each segment gets its own orientation; the phone does not move between
    // gaits in reality, so keep the seed for all segments of one placement.
    const part = synthesize({ ...extra, gait, durationS: seconds, seed: seed * 1000 + 7, startMs: start });
    out.push(...part);
    start += seconds * 1000;
  }
  return out;
}
