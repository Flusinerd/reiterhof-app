// Pure model of a tracked session (JAN-63 GPS mode, JAN-65 indoor mode): pause handling,
// GPS fix filtering, statistics, rein time accounting, gait corrections, gait timeline and the
// API payload. No React Native imports (unit-tested with node --test). All times are epoch
// milliseconds; the phone clock is the single time base (samples, fixes, windows).

import { exportWindows, type WindowRecord } from "./gait/export.ts";
import { gaitFromSpeed } from "./gait/fusion.ts";
import type { Gait, GaitWindow } from "./gait/types.ts";
import { elevationGain, haversineM, simplifyToMax, type LatLon } from "./geo.ts";

export type Rein = "left" | "right";
export type TrackMode = "gps" | "indoor";

/** Accepted GPS fix. `seg` numbers the run between two pauses (no distance across runs). */
export type Fix = {
  t: number;
  lat: number;
  lon: number;
  alt?: number;
  /** Speed in m/s as reported by the OS. */
  speed?: number;
  seg: number;
};

/** Raw fix as delivered by expo-location (coords + timestamp), already converted to plain numbers. */
export type RawFix = {
  t: number;
  lat: number;
  lon: number;
  alt?: number | null;
  speed?: number | null;
  accuracy?: number | null;
};

export type Interval = { from: number; to: number | null };

/** "This is trot, not what the detector says": see `applyCorrections`. */
export type Correction = { fromMs: number; wrong: Gait; right: Gait };

export type ReinEntry = { t: number; rein: Rein };

export type TrackState = {
  mode: TrackMode;
  startedAt: number;
  /** Active (not paused) periods; the last one is open while running. */
  intervals: Interval[];
  fixes: Fix[];
  reins: ReinEntry[];
  corrections: Correction[];
  distanceM: number;
};

/** Fixes worse than this (metres) are ignored. */
export const MAX_ACCURACY_M = 35;
/** A fix implying more than this speed (m/s, 90 km/h) since the last one is a GPS jump. */
export const MAX_SPEED_MS = 25;
/** How far back a correction reaches (ms): the gait the user has just been riding. */
export const CORRECTION_LOOKBACK_MS = 6000;

export function createState(mode: TrackMode, startedAt: number, rein: Rein = "left"): TrackState {
  return {
    mode,
    startedAt,
    intervals: [{ from: startedAt, to: null }],
    fixes: [],
    reins: mode === "indoor" ? [{ t: startedAt, rein }] : [],
    corrections: [],
    distanceM: 0,
  };
}

// --- pause and time ------------------------------------------------------------------------

export function isPaused(state: TrackState): boolean {
  return state.intervals.length === 0 || state.intervals[state.intervals.length - 1]!.to !== null;
}

export function pause(state: TrackState, now: number): TrackState {
  if (isPaused(state)) return state;
  const intervals = state.intervals.slice();
  const last = intervals[intervals.length - 1]!;
  intervals[intervals.length - 1] = { from: last.from, to: Math.max(now, last.from) };
  return { ...state, intervals };
}

export function resume(state: TrackState, now: number): TrackState {
  if (!isPaused(state)) return state;
  return { ...state, intervals: [...state.intervals, { from: now, to: null }] };
}

/** Active milliseconds inside [from, to] (open intervals end at `now`). */
export function activeBetween(state: TrackState, from: number, to: number, now: number): number {
  let ms = 0;
  for (const iv of state.intervals) {
    const end = iv.to ?? now;
    const a = Math.max(iv.from, from);
    const b = Math.min(end, to);
    if (b > a) ms += b - a;
  }
  return ms;
}

/** Total active time in ms (pauses excluded). */
export function activeMs(state: TrackState, now: number): number {
  return activeBetween(state, state.startedAt, Math.max(now, state.startedAt), now);
}

/** Index of the current active run (0 before the first pause). */
function currentSeg(state: TrackState): number {
  return Math.max(0, state.intervals.length - 1);
}

// --- GPS fixes -----------------------------------------------------------------------------

/**
 * Adds a fix unless it is unusable: while paused, older than the last fix, inaccurate, or a
 * jump that implies an impossible speed. Distance grows only between fixes of the same run.
 */
export function addFix(state: TrackState, raw: RawFix): TrackState {
  if (isPaused(state)) return state;
  const active = state.intervals[state.intervals.length - 1]!;
  if (raw.t < active.from) return state; // recorded before the last resume
  if (!Number.isFinite(raw.lat) || !Number.isFinite(raw.lon)) return state;
  if (raw.accuracy != null && (raw.accuracy < 0 || raw.accuracy > MAX_ACCURACY_M)) return state;
  const seg = currentSeg(state);
  const prev = state.fixes[state.fixes.length - 1];
  let add = 0;
  if (prev) {
    if (raw.t <= prev.t) return state;
    if (prev.seg === seg) {
      add = haversineM(prev, raw);
      const dt = (raw.t - prev.t) / 1000;
      if (add / dt > MAX_SPEED_MS) return state;
    }
  }
  const fix: Fix = { t: raw.t, lat: raw.lat, lon: raw.lon, seg };
  if (raw.alt != null && Number.isFinite(raw.alt)) fix.alt = raw.alt;
  if (raw.speed != null && Number.isFinite(raw.speed) && raw.speed >= 0) fix.speed = raw.speed;
  return { ...state, fixes: [...state.fixes, fix], distanceM: state.distanceM + add };
}

export type Stats = {
  activeSeconds: number;
  distanceM: number;
  /** Average speed over the active time, km/h. */
  avgSpeedKmh: number;
  elevationGainM: number;
};

export function stats(state: TrackState, now: number): Stats {
  const seconds = activeMs(state, now) / 1000;
  const alts = state.fixes.flatMap((f) => (f.alt === undefined ? [] : [f.alt]));
  return {
    activeSeconds: seconds,
    distanceM: state.distanceM,
    avgSpeedKmh: seconds > 0 ? (state.distanceM / seconds) * 3.6 : 0,
    elevationGainM: elevationGain(alts),
  };
}

// --- rein accounting (indoor) ----------------------------------------------------------------

/** Switch to the other rein ("Handwechsel"); returns the same state when nothing changes. */
export function changeRein(state: TrackState, now: number): TrackState {
  if (state.mode !== "indoor" || isPaused(state)) return state;
  const current = currentRein(state);
  if (!current) return state;
  const rein: Rein = current === "left" ? "right" : "left";
  return { ...state, reins: [...state.reins, { t: now, rein }] };
}

export function currentRein(state: TrackState): Rein | null {
  return state.reins[state.reins.length - 1]?.rein ?? null;
}

export type ReinSegment = { rein: Rein; minutes: number };

/**
 * Ordered rein segments in active minutes (pauses do not count). Segments shorter than 5 s
 * are folded into their neighbour (a mistaken tap, or a change right at the start), and
 * neighbours on the same rein are merged, so the number of rein changes is `length - 1`.
 */
export function reinSegments(state: TrackState, now: number): ReinSegment[] {
  const raw: { rein: Rein; ms: number }[] = [];
  state.reins.forEach((entry, i) => {
    const end = state.reins[i + 1]?.t ?? Math.max(now, entry.t);
    raw.push({ rein: entry.rein, ms: activeBetween(state, entry.t, end, now) });
  });
  const MIN_MS = 5000;
  // Drop tiny segments (attribute their time to the previous kept one, else the next one).
  const kept: { rein: Rein; ms: number }[] = [];
  let carry = 0;
  for (const seg of raw) {
    if (seg.ms < MIN_MS && raw.length > 1) {
      if (kept.length > 0) kept[kept.length - 1]!.ms += seg.ms;
      else carry += seg.ms;
      continue;
    }
    kept.push({ rein: seg.rein, ms: seg.ms + carry });
    carry = 0;
  }
  if (kept.length === 0 && raw.length > 0) kept.push({ rein: raw[0]!.rein, ms: raw.reduce((s, r) => s + r.ms, 0) });
  const merged: { rein: Rein; ms: number }[] = [];
  for (const seg of kept) {
    const last = merged[merged.length - 1];
    if (last && last.rein === seg.rein) last.ms += seg.ms;
    else merged.push({ ...seg });
  }
  return merged.map((s) => ({ rein: s.rein, minutes: Math.round((s.ms / 60000) * 100) / 100 }));
}

/** Minutes on the current rein since the last change (active time). */
export function currentReinMinutes(state: TrackState, now: number): number {
  const last = state.reins[state.reins.length - 1];
  if (!last) return 0;
  return activeBetween(state, last.t, Math.max(now, last.t), now) / 60000;
}

// --- gait corrections and timeline -------------------------------------------------------------

/**
 * The user taps the gait button: `right` is the true gait, `wrong` what the detector currently
 * says. Windows from a short lookback until the detector changes its mind count as labelled.
 */
export function addCorrection(state: TrackState, now: number, wrong: Gait, right: Gait): TrackState {
  return { ...state, corrections: [...state.corrections, { fromMs: now, wrong, right }] };
}

/**
 * Sets `label` on the windows covered by a correction: every window that ends after
 * `fromMs - lookback` and predicts `wrong`, up to the first later window that predicts
 * something else (the correction ends there) or the next correction. Windows that start
 * before a later tap belong to the earlier correction. Returns new windows.
 */
export function applyCorrections(
  windows: readonly GaitWindow[],
  corrections: readonly Correction[],
  lookbackMs = CORRECTION_LOOKBACK_MS,
): GaitWindow[] {
  const out = windows.map((w) => ({ ...w }));
  const sorted = [...corrections].sort((a, b) => a.fromMs - b.fromMs);
  const claimed = new Set<GaitWindow>(); // an earlier correction owns the windows before the next tap
  sorted.forEach((c, ci) => {
    const nextFrom = sorted[ci + 1]?.fromMs ?? Infinity;
    let ended = false;
    for (const w of out) {
      if (w.endMs <= c.fromMs - lookbackMs) continue;
      if (w.startMs >= nextFrom) break;
      if (ended) break;
      if (claimed.has(w)) continue;
      if (w.gait !== c.wrong) {
        // Before the tap a different gait means the lookback reached into a previous stretch.
        if (w.startMs >= c.fromMs) ended = true;
        continue;
      }
      w.label = c.right;
      claimed.add(w);
    }
  });
  return out;
}

/** Gait of a window for display and summaries: the corrected label wins. */
export function effectiveGait(w: GaitWindow): Gait {
  return w.label ?? w.gait;
}

/**
 * Gait at time `t` from the windows (each window stands for its hop, centered in the window).
 * Returns null when no window covers `t`. `windows` must be ordered by start.
 */
export function gaitAt(windows: readonly GaitWindow[], t: number): Gait | null {
  let lo = 0;
  let hi = windows.length - 1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    const w = windows[mid]!;
    const center = (w.startMs + w.endMs) / 2;
    const half = w.weightMs / 2;
    if (t < center - half) hi = mid - 1;
    else if (t >= center + half) lo = mid + 1;
    else return effectiveGait(w);
  }
  return null;
}

/**
 * Gait of a GPS point: the windows if they cover it (they include the accelerometer and speed
 * fusion), else the OS speed (e.g. when the accelerometer stopped in the background), else
 * the speed implied by the previous fix.
 */
export function pointGait(windows: readonly GaitWindow[], fix: Fix, prev: Fix | undefined): Gait {
  const fromWindows = gaitAt(windows, fix.t);
  if (fromWindows) return fromWindows;
  if (fix.speed !== undefined) return gaitFromSpeed(fix.speed * 3.6);
  if (prev && prev.seg === fix.seg && fix.t > prev.t) {
    return gaitFromSpeed((haversineM(prev, fix) / ((fix.t - prev.t) / 1000)) * 3.6);
  }
  return "halt";
}

export type GaitPoint = LatLon & { t: number; alt?: number; gait: Gait; seg: number };

/** Track points with a gait each. `windows` are the corrected windows. */
export function gaitPoints(state: TrackState, windows: readonly GaitWindow[]): GaitPoint[] {
  return state.fixes.map((f, i) => {
    const p: GaitPoint = { lat: f.lat, lon: f.lon, t: f.t, gait: pointGait(windows, f, state.fixes[i - 1]), seg: f.seg };
    if (f.alt !== undefined) p.alt = f.alt;
    return p;
  });
}

export type GaitSegment = { gait: Gait; coords: LatLon[] };

/**
 * Splits a track into runs of the same gait for the map polylines. Consecutive segments share
 * their joint, so the line is closed. The stretch between two runs (a pause) is not drawn.
 */
export function segmentByGait(points: readonly GaitPoint[]): GaitSegment[] {
  const out: GaitSegment[] = [];
  let current: GaitSegment | null = null;
  points.forEach((p, i) => {
    const prev = points[i - 1];
    if (!prev || prev.seg !== p.seg) {
      current = null; // first point of a run: nothing to draw yet
      return;
    }
    // the stretch prev -> p is ridden in the gait measured at p
    const coord = { lat: p.lat, lon: p.lon };
    if (current && current.gait === p.gait) {
      current.coords.push(coord);
    } else {
      current = { gait: p.gait, coords: [{ lat: prev.lat, lon: prev.lon }, coord] };
      out.push(current);
    }
  });
  return out;
}

// --- API payload ------------------------------------------------------------------------------

/**
 * Gait shares (percent, summing to 100) from the GPS points alone, weighting each stretch by
 * its duration. Used when no accelerometer windows exist (sensor missing or stopped).
 */
export function pointShares(points: readonly GaitPoint[]): Record<Gait, number> {
  const ms: Record<Gait, number> = { halt: 0, walk: 0, trot: 0, canter: 0 };
  points.forEach((p, i) => {
    const prev = points[i - 1];
    if (prev && prev.seg === p.seg && p.t > prev.t) ms[p.gait] += p.t - prev.t;
  });
  const total = ms.halt + ms.walk + ms.trot + ms.canter;
  const pct = (v: number) => (total > 0 ? (v / total) * 100 : 0);
  return { halt: pct(ms.halt), walk: pct(ms.walk), trot: pct(ms.trot), canter: pct(ms.canter) };
}

export const MAX_TRACK_POINTS = 5000;

export type ApiTrackPoint = { lat: number; lon: number; t: number; alt?: number; g: Gait };

const round = (v: number, digits: number) => {
  const m = 10 ** digits;
  return Math.round(v * m) / m;
};

/**
 * Simplified track for `POST /sessions` (`track`): Douglas-Peucker (2 m, doubled until it fits
 * `maxPoints`); points next to a gait change or a pause are kept. `t` is seconds since the
 * start, `g` the gait of the stretch that ends at the point (the first point after a pause
 * is "halt": the stretch bridges the pause).
 */
export function buildApiTrack(
  state: TrackState,
  windows: readonly GaitWindow[],
  maxPoints = MAX_TRACK_POINTS,
): ApiTrackPoint[] {
  const pts = gaitPoints(state, windows);
  if (pts.length === 0) return [];
  const keep = (i: number) => {
    const p = pts[i]!;
    const prev = pts[i - 1];
    const next = pts[i + 1];
    return (
      (prev !== undefined && (prev.gait !== p.gait || prev.seg !== p.seg)) ||
      (next !== undefined && (next.gait !== p.gait || next.seg !== p.seg))
    );
  };
  const index = new Map(pts.map((p, i) => [p, i] as const));
  return simplifyToMax(pts, maxPoints, keep).map((p) => {
    const prev = pts[(index.get(p) ?? 0) - 1];
    const out: ApiTrackPoint = {
      lat: round(p.lat, 6),
      lon: round(p.lon, 6),
      t: Math.max(0, round((p.t - state.startedAt) / 1000, 1)),
      g: prev !== undefined && prev.seg !== p.seg ? "halt" : p.gait,
    };
    if (p.alt !== undefined) out.alt = round(p.alt, 1);
    return out;
  });
}

export const MAX_API_WINDOWS = 3000;

/**
 * Raw gait windows for model improvement (`gait_windows`): compact records with `t` in ms
 * since the start. Above `max` windows the corrected ones are kept and the rest is thinned evenly.
 */
export function buildApiWindows(
  windows: readonly GaitWindow[],
  startedAt: number,
  max = MAX_API_WINDOWS,
): WindowRecord[] {
  const records = exportWindows(windows).map((r) => ({ ...r, t: Math.max(0, r.t - startedAt) }));
  if (records.length <= max) return records;
  const labelled = records.filter((r) => r.c !== undefined).slice(0, max);
  const rest = records.filter((r) => r.c === undefined);
  const room = max - labelled.length;
  const step = rest.length / Math.max(1, room);
  const sampled: WindowRecord[] = [];
  for (let i = 0; i < room; i++) sampled.push(rest[Math.floor(i * step)]!);
  return [...labelled, ...sampled].sort((a, b) => a.t - b.t);
}

/** Rounded minutes and gait fractions (0..1) for the finish screen / API from `gaitShares`. */
export function apiGaitShares(percent: Record<Gait, number>): Record<string, number> {
  const out: Record<string, number> = {};
  let sum = 0;
  for (const g of ["halt", "walk", "trot", "canter"] as const) {
    const v = Math.max(0, percent[g] ?? 0) / 100;
    out[g] = Math.floor(v * 1000) / 1000; // floor so the sum never exceeds 1
    sum += out[g]!;
  }
  return sum > 0 ? out : {};
}

/** Distance in whole metres, or undefined when there is nothing worth sending. */
export function apiDistance(distanceM: number): number | undefined {
  const m = Math.round(distanceM);
  return m > 0 ? Math.min(m, 500_000) : undefined;
}

// --- target duration ---------------------------------------------------------------------------

/** Progress 0..1 of the active time against the target (minutes); 0 without a target. */
export function targetProgress(activeSeconds: number, targetMinutes: number | undefined): number {
  if (!targetMinutes || targetMinutes <= 0) return 0;
  return Math.max(0, Math.min(1, activeSeconds / (targetMinutes * 60)));
}

/** "noch 12 Min." / "Ziel erreicht" / "+5 Min." for the target line. */
export function targetLabel(activeSeconds: number, targetMinutes: number | undefined): string {
  if (!targetMinutes || targetMinutes <= 0) return "";
  const diff = Math.round(targetMinutes - activeSeconds / 60);
  if (diff > 0) return `noch ${diff} Min.`;
  if (diff === 0) return "Ziel erreicht";
  return `${-diff} Min. über dem Ziel`;
}

// --- exercise checklist --------------------------------------------------------------------------

/** Toggles a step in the sorted list of checked indexes. */
export function toggleStep(checked: readonly number[], index: number): number[] {
  return checked.includes(index) ? checked.filter((i) => i !== index) : [...checked, index].sort((a, b) => a - b);
}
