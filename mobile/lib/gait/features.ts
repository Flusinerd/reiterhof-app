import { amplitudeSpectrum, nextPow2 } from "./fft.ts";
import type { Features, Sample } from "./types.ts";

export const SAMPLE_RATE = 50;

/** Band searched for the dominant frequency (Hz). */
const PEAK_MIN_HZ = 0.6;
const PEAK_MAX_HZ = 3.5;
/** Band used to compute the regularity (Hz). */
const BAND_MIN_HZ = 0.5;
const BAND_MAX_HZ = 4.0;

/**
 * Linear resampling of x/y/z to a uniform grid of `n` points starting at
 * `startMs`. Returns null when the samples do not cover the window well
 * (fewer than 60 % of the expected samples, or a gap longer than 500 ms), so
 * that dropouts never produce a bogus classification.
 */
export function resampleWindow(
  samples: readonly Sample[],
  startMs: number,
  durationMs: number,
  rate: number,
): { x: Float64Array; y: Float64Array; z: Float64Array } | null {
  const n = Math.round((durationMs * rate) / 1000);
  const inside = samples.filter((s) => s.t >= startMs - 100 && s.t <= startMs + durationMs + 100);
  if (n < 8 || inside.length < 0.6 * n) return null;
  for (let i = 1; i < inside.length; i++) {
    if (inside[i]!.t - inside[i - 1]!.t > 500) return null;
  }
  const x = new Float64Array(n);
  const y = new Float64Array(n);
  const z = new Float64Array(n);
  let j = 0;
  for (let i = 0; i < n; i++) {
    const t = startMs + (i * 1000) / rate;
    while (j < inside.length - 2 && inside[j + 1]!.t < t) j++;
    const a = inside[j]!;
    const b = inside[Math.min(j + 1, inside.length - 1)]!;
    const span = b.t - a.t;
    const f = span > 0 ? Math.min(1, Math.max(0, (t - a.t) / span)) : 0;
    x[i] = a.x + (b.x - a.x) * f;
    y[i] = a.y + (b.y - a.y) * f;
    z[i] = a.z + (b.z - a.z) * f;
  }
  return { x, y, z };
}

function mean(a: ArrayLike<number>): number {
  let s = 0;
  for (let i = 0; i < a.length; i++) s += a[i]!;
  return a.length ? s / a.length : 0;
}

function rms(a: ArrayLike<number>): number {
  let s = 0;
  for (let i = 0; i < a.length; i++) s += a[i]! * a[i]!;
  return a.length ? Math.sqrt(s / a.length) : 0;
}

/** Removes the least-squares line from `a` in place. */
function detrend(a: Float64Array): void {
  const n = a.length;
  const tMean = (n - 1) / 2;
  let num = 0;
  let den = 0;
  const m = mean(a);
  for (let i = 0; i < n; i++) {
    num += (i - tMean) * (a[i]! - m);
    den += (i - tMean) * (i - tMean);
  }
  const slope = den > 0 ? num / den : 0;
  for (let i = 0; i < n; i++) a[i] = a[i]! - m - slope * (i - tMean);
}

/**
 * Splits acceleration into a vertical part (along the estimated gravity
 * direction) and the horizontal remainder. Gravity is the mean vector of the
 * window, i.e. a high-pass at about 1 / window length; this needs no knowledge
 * of how the phone is oriented.
 */
export function splitGravity(
  x: Float64Array,
  y: Float64Array,
  z: Float64Array,
): { vertical: Float64Array; horizontal: Float64Array } {
  const n = x.length;
  const gx = mean(x);
  const gy = mean(y);
  const gz = mean(z);
  const norm = Math.hypot(gx, gy, gz);
  const ux = norm > 1e-6 ? gx / norm : 0;
  const uy = norm > 1e-6 ? gy / norm : 0;
  const uz = norm > 1e-6 ? gz / norm : 1;
  const vertical = new Float64Array(n);
  const horizontal = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    const dx = x[i]! - gx;
    const dy = y[i]! - gy;
    const dz = z[i]! - gz;
    const v = dx * ux + dy * uy + dz * uz;
    vertical[i] = v;
    horizontal[i] = Math.hypot(dx - v * ux, dy - v * uy, dz - v * uz);
  }
  return { vertical, horizontal };
}

/** Peak heights (measured from the window minimum) of a lightly smoothed signal. */
function peakHeights(v: Float64Array, rate: number): number[] {
  const n = v.length;
  // 3-point moving average removes sample noise without moving peaks.
  const s = new Float64Array(n);
  for (let i = 0; i < n; i++) {
    const a = v[Math.max(0, i - 1)]!;
    const b = v[i]!;
    const c = v[Math.min(n - 1, i + 1)]!;
    s[i] = (a + b + c) / 3;
  }
  let min = Infinity;
  for (let i = 0; i < n; i++) min = Math.min(min, s[i]!);
  const level = rms(s) * 0.25;
  const minDist = Math.max(2, Math.round(0.15 * rate));
  const peaks: { i: number; h: number }[] = [];
  for (let i = 1; i < n - 1; i++) {
    if (!(s[i]! > s[i - 1]! && s[i]! >= s[i + 1]!) || s[i]! < level) continue;
    const last = peaks[peaks.length - 1];
    if (last && i - last.i < minDist) {
      if (s[i]! > last.h + min) peaks[peaks.length - 1] = { i, h: s[i]! - min };
    } else {
      peaks.push({ i, h: s[i]! - min });
    }
  }
  return peaks.map((p) => p.h);
}

/** Mean of `1 - small/large` over consecutive peak pairs; 0 with fewer than 4 peaks. */
export function alternatingAsymmetry(v: Float64Array, rate: number): number {
  const h = peakHeights(v, rate);
  if (h.length < 4) return 0;
  let sum = 0;
  for (let i = 1; i < h.length; i++) {
    const hi = Math.max(h[i]!, h[i - 1]!);
    const lo = Math.min(h[i]!, h[i - 1]!);
    sum += hi > 0 ? 1 - lo / hi : 0;
  }
  return sum / (h.length - 1);
}

/**
 * Computes the features of one window. Returns null when the samples do not
 * cover the window (see `resampleWindow`).
 */
export function extractFeatures(
  samples: readonly Sample[],
  startMs: number,
  durationMs = 4000,
  rate = SAMPLE_RATE,
): Features | null {
  const r = resampleWindow(samples, startMs, durationMs, rate);
  if (!r) return null;
  const { vertical, horizontal } = splitGravity(r.x, r.y, r.z);
  detrend(vertical);
  const verticalEnergy = rms(vertical);
  const horizontalEnergy = rms(horizontal);

  const size = nextPow2(vertical.length * 2);
  const spec = amplitudeSpectrum(vertical, size);
  const df = rate / size;
  const lo = Math.max(1, Math.ceil(PEAK_MIN_HZ / df));
  const hi = Math.min(spec.length - 2, Math.floor(PEAK_MAX_HZ / df));
  let peak = lo;
  for (let k = lo; k <= hi; k++) if (spec[k]! > spec[peak]!) peak = k;

  // Parabolic interpolation of the peak position.
  const a = spec[peak - 1]!;
  const b = spec[peak]!;
  const c = spec[peak + 1]!;
  const denom = a - 2 * b + c;
  const shift = denom !== 0 ? (0.5 * (a - c)) / denom : 0;
  const freq = (peak + Math.max(-0.5, Math.min(0.5, shift))) * df;

  // Regularity: power around the peak relative to the band power.
  let bandPower = 0;
  let peakPower = 0;
  const halfWidth = 0.4 / df;
  for (let k = Math.floor(BAND_MIN_HZ / df); k <= Math.ceil(BAND_MAX_HZ / df); k++) {
    const p = spec[k]! * spec[k]!;
    bandPower += p;
    if (Math.abs(k - peak) <= halfWidth) peakPower += p;
  }
  const regularity = bandPower > 0 ? peakPower / bandPower : 0;

  // Second harmonic relative to the fundamental.
  let h2 = 0;
  for (let k = Math.floor((2 * freq - 0.15) / df); k <= Math.ceil((2 * freq + 0.15) / df); k++) {
    if (k > 0 && k < spec.length) h2 = Math.max(h2, spec[k]!);
  }
  const harmonicRatio = spec[peak]! > 0 ? h2 / spec[peak]! : 0;

  return {
    freq,
    verticalEnergy,
    horizontalEnergy,
    asymmetry: alternatingAsymmetry(vertical, rate),
    harmonicRatio,
    regularity,
  };
}
