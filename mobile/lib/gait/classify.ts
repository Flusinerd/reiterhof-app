import type { Calibration, Classification, Features, Gait } from "./types.ts";

/** Defaults for a phone worn in a trouser pocket / on the body; no learning needed. */
export const DEFAULT_CALIBRATION: Calibration = {
  energyScale: 1,
  haltEnergy: 0.04,
  walkTrotFreq: 1.25,
  trotCanterLow: 1.6,
  trotCanterHigh: 1.85,
  asymmetryThreshold: 0.35,
  minRegularity: 0.5,
};

function clamp01(x: number): number {
  return Math.max(0, Math.min(1, x));
}

/** 0 on the boundary, 1 when `width` or more away from it. */
function margin(x: number, boundary: number, width: number): number {
  return clamp01(Math.abs(x - boundary) / width);
}

/**
 * Shape score used to tell canter from trot: the larger of the alternating-peak
 * asymmetry and the relative 2nd harmonic. The canter waveform within one
 * stride is uneven, which shows up as either alternating peak heights or a
 * strong 2nd harmonic depending on the sensor position.
 */
export function asymmetryScore(f: Features): number {
  return Math.max(f.asymmetry, f.harmonicRatio);
}

/**
 * Classifies one window from the accelerometer features alone.
 *
 * Rules (thresholds from `cal`, energy normalised by `cal.energyScale`):
 * 1. Vertical energy below `haltEnergy`, or no periodic pattern
 *    (`regularity < minRegularity`): halt.
 * 2. Dominant frequency below `walkTrotFreq` (default 1.25 Hz): walk.
 * 3. Above `trotCanterHigh` (1.85 Hz): canter; below `trotCanterLow` (1.6 Hz): trot.
 * 4. In between the two overlap ranges of trot (1.3-1.8) and canter
 *    (1.6-2.2) the shape decides: `asymmetryScore > asymmetryThreshold` is canter.
 *
 * `confidence` (0..1) is the distance of the deciding feature from its nearest
 * decision boundary, used by the GPS fusion and shown as certainty.
 */
export function classifyWindow(f: Features, cal: Calibration): Classification {
  const e = f.verticalEnergy / cal.energyScale;
  if (e < cal.haltEnergy) {
    return { gait: "halt", confidence: margin(e, cal.haltEnergy, cal.haltEnergy * 0.5) };
  }
  if (f.regularity < cal.minRegularity) {
    return { gait: "halt", confidence: margin(f.regularity, cal.minRegularity, 0.15) * 0.8 };
  }
  // Moving: confidence also depends on being clearly above the halt threshold.
  const energyConf = margin(e, cal.haltEnergy, cal.haltEnergy);
  let gait: Gait;
  let conf: number;
  if (f.freq < cal.walkTrotFreq) {
    gait = "walk";
    conf = margin(f.freq, cal.walkTrotFreq, 0.15);
  } else if (f.freq > cal.trotCanterHigh) {
    gait = "canter";
    conf = margin(f.freq, cal.trotCanterHigh, 0.15);
  } else if (f.freq < cal.trotCanterLow) {
    gait = "trot";
    conf = Math.min(margin(f.freq, cal.walkTrotFreq, 0.15), margin(f.freq, cal.trotCanterLow, 0.15));
  } else {
    const s = asymmetryScore(f);
    gait = s > cal.asymmetryThreshold ? "canter" : "trot";
    conf = margin(s, cal.asymmetryThreshold, 0.2);
  }
  return { gait, confidence: Math.min(conf, energyConf) };
}
