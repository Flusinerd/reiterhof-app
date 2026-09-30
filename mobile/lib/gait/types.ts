// Shared types of the gait detector. Pure TypeScript, no React Native imports.

/** Accelerometer sample as delivered by expo-sensors: g units, t in ms. */
export type Sample = { t: number; x: number; y: number; z: number };

/** GPS speed fix. */
export type GpsFix = { t: number; speedKmh: number };

export type Gait = "halt" | "walk" | "trot" | "canter";

export const GAITS: readonly Gait[] = ["halt", "walk", "trot", "canter"];

/** Features of one analysis window. */
export type Features = {
  /** Dominant stride/step frequency in Hz (0 when the window is not periodic). */
  freq: number;
  /** RMS of the gravity-free vertical acceleration, in g. */
  verticalEnergy: number;
  /** RMS of the gravity-free horizontal acceleration, in g. */
  horizontalEnergy: number;
  /**
   * Alternating-peak asymmetry in 0..1: mean of `1 - small/large` over
   * consecutive peaks of the vertical signal. About 0 for the symmetric
   * two-beat trot, large when peak heights alternate (canter).
   */
  asymmetry: number;
  /** Amplitude of the 2nd harmonic relative to the fundamental (0..~1). */
  harmonicRatio: number;
  /** Share of the band power (0.5-4 Hz) that lies in the dominant peak, 0..1. */
  regularity: number;
};

/**
 * Per-user/placement tuning. All energy thresholds are applied to
 * `verticalEnergy / energyScale`, so a single factor adapts the detector to a
 * sensor position (pocket vs arm) with different signal amplitude.
 */
export type Calibration = {
  /** Multiplier for the amplitude: 1 = default placement. */
  energyScale: number;
  /** Normalised vertical RMS (g) below which the horse is standing. */
  haltEnergy: number;
  /** Below this normalised frequency (Hz) a moving horse is walking. */
  walkTrotFreq: number;
  /** Lower edge (Hz) of the trot/canter overlap zone; below it: trot. */
  trotCanterLow: number;
  /** Upper edge (Hz) of the overlap zone; above it: canter. */
  trotCanterHigh: number;
  /** In the overlap zone: asymmetry above this means canter. */
  asymmetryThreshold: number;
  /** Below this regularity the movement is not a gait (treated as halt). */
  minRegularity: number;
};

/** Result of classifying one window from the accelerometer alone. */
export type Classification = { gait: Gait; confidence: number };

/** One analysed window. */
export type GaitWindow = {
  /** Window start (ms, same clock as the samples). */
  startMs: number;
  endMs: number;
  /** Time this window contributes to summaries (hop length, so overlap is not double counted). */
  weightMs: number;
  features: Features;
  /** Accelerometer-only prediction. */
  accelGait: Gait;
  accelConfidence: number;
  /** Mean GPS speed in the window, if a fix was available. */
  speedKmh?: number;
  /** After GPS fusion, before smoothing. */
  fusedGait: Gait;
  /** Final prediction after majority-vote smoothing. */
  gait: Gait;
  /** User-corrected label, if any (see export.ts). */
  label?: Gait;
};
