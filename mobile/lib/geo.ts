// Pure geo helpers for the GPS tracker (JAN-63): distance, elevation gain, polyline
// simplification. No React Native imports (unit-tested with node --test).

export type LatLon = { lat: number; lon: number };

const EARTH_RADIUS_M = 6_371_008.8;

const rad = (deg: number) => (deg * Math.PI) / 180;

/** Great-circle distance in metres (haversine). */
export function haversineM(a: LatLon, b: LatLon): number {
  const dLat = rad(b.lat - a.lat);
  const dLon = rad(b.lon - a.lon);
  const h = Math.sin(dLat / 2) ** 2 + Math.cos(rad(a.lat)) * Math.cos(rad(b.lat)) * Math.sin(dLon / 2) ** 2;
  return 2 * EARTH_RADIUS_M * Math.asin(Math.min(1, Math.sqrt(h)));
}

/** Length of a polyline in metres. */
export function pathLengthM(points: readonly LatLon[]): number {
  let sum = 0;
  for (let i = 1; i < points.length; i++) sum += haversineM(points[i - 1]!, points[i]!);
  return sum;
}

export type ElevationOptions = {
  /** Moving-average window (samples, odd is best) applied before counting. Default 5. */
  smoothing?: number;
  /** Minimum rise in metres that counts as climbing (filters GPS noise). Default 3. */
  threshold?: number;
};

/** Centered moving average; the ends use the samples that exist. */
export function movingAverage(values: readonly number[], window: number): number[] {
  const half = Math.max(0, Math.floor(window / 2));
  return values.map((_, i) => {
    const from = Math.max(0, i - half);
    const to = Math.min(values.length - 1, i + half);
    let sum = 0;
    for (let j = from; j <= to; j++) sum += values[j]!;
    return sum / (to - from + 1);
  });
}

/**
 * Total ascent in metres from a series of altitudes. The series is smoothed with a moving
 * average and then counted with hysteresis: a rise only counts once it exceeds `threshold`
 * above the last reference level, and a drop below the reference lowers the reference.
 */
export function elevationGain(altitudes: readonly number[], options: ElevationOptions = {}): number {
  const clean = altitudes.filter((a) => Number.isFinite(a));
  if (clean.length < 2) return 0;
  const threshold = options.threshold ?? 3;
  const smooth = movingAverage(clean, options.smoothing ?? 5);
  let ref = smooth[0]!;
  let gain = 0;
  for (const v of smooth) {
    if (v - ref >= threshold) {
      gain += v - ref;
      ref = v;
    } else if (ref - v >= threshold) {
      ref = v;
    } else if (v < ref) {
      // small dip: follow it down so the next climb is measured from the real low point
      ref = v;
    }
  }
  return gain;
}

// --- simplification ------------------------------------------------------------------------

type XY = { x: number; y: number };

/** Local equirectangular projection around the first point, in metres. */
function project(points: readonly LatLon[]): XY[] {
  const origin = points[0];
  if (!origin) return [];
  const kx = Math.cos(rad(origin.lat)) * 111_320;
  const ky = 110_540;
  return points.map((p) => ({ x: (p.lon - origin.lon) * kx, y: (p.lat - origin.lat) * ky }));
}

function segmentDistance(p: XY, a: XY, b: XY): number {
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const len2 = dx * dx + dy * dy;
  if (len2 === 0) return Math.hypot(p.x - a.x, p.y - a.y);
  const t = Math.max(0, Math.min(1, ((p.x - a.x) * dx + (p.y - a.y) * dy) / len2));
  return Math.hypot(p.x - (a.x + t * dx), p.y - (a.y + t * dy));
}

/**
 * Douglas-Peucker simplification. Keeps the first and last point and every point whose
 * index is in `keep` (e.g. gait changes), and drops points closer than `toleranceM` to the
 * line between kept neighbours. Iterative, so long tracks cannot overflow the stack.
 * Returns the kept points in their original order (same objects).
 */
export function simplifyPolyline<T extends LatLon>(
  points: readonly T[],
  toleranceM: number,
  keep?: (index: number) => boolean,
): T[] {
  if (points.length <= 2) return [...points];
  const xy = project(points);
  const kept = new Array<boolean>(points.length).fill(false);
  kept[0] = true;
  kept[points.length - 1] = true;
  if (keep) for (let i = 0; i < points.length; i++) if (keep(i)) kept[i] = true;

  // Simplify each stretch between two forced points independently.
  const forced: number[] = [];
  kept.forEach((k, i) => {
    if (k) forced.push(i);
  });
  const stack: [number, number][] = [];
  for (let i = 1; i < forced.length; i++) stack.push([forced[i - 1]!, forced[i]!]);
  while (stack.length > 0) {
    const [first, last] = stack.pop()!;
    if (last - first < 2) continue;
    let maxD = -1;
    let index = -1;
    for (let i = first + 1; i < last; i++) {
      const d = segmentDistance(xy[i]!, xy[first]!, xy[last]!);
      if (d > maxD) {
        maxD = d;
        index = i;
      }
    }
    if (maxD > toleranceM && index > 0) {
      kept[index] = true;
      stack.push([first, index], [index, last]);
    }
  }
  return points.filter((_, i) => kept[i]);
}

/**
 * Simplifies until at most `maxPoints` remain by doubling the tolerance (start 2 m). Forced
 * points (`keep`) are always retained, so the result can exceed `maxPoints` only if there are
 * more forced points than that; then an even thinning of the result caps it.
 */
export function simplifyToMax<T extends LatLon>(
  points: readonly T[],
  maxPoints: number,
  keep?: (index: number) => boolean,
  startToleranceM = 2,
): T[] {
  let tolerance = startToleranceM;
  let out = simplifyPolyline(points, tolerance, keep);
  for (let i = 0; out.length > maxPoints && i < 20; i++) {
    tolerance *= 2;
    out = simplifyPolyline(points, tolerance, keep);
  }
  if (out.length > maxPoints) {
    const step = out.length / maxPoints;
    const thinned: T[] = [];
    for (let i = 0; i < maxPoints - 1; i++) thinned.push(out[Math.floor(i * step)]!);
    thinned.push(out[out.length - 1]!);
    out = thinned;
  }
  return out;
}
