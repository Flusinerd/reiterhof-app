import assert from "node:assert/strict";
import { test } from "node:test";

import {
  DEFAULT_CALIBRATION,
  FEATURE_ORDER,
  GAITS,
  GaitStream,
  analyzeWindows,
  classifyWindow,
  correctRecords,
  exportWindows,
  extractFeatures,
  fuseWithSpeed,
  gaitFromSpeed,
  gaitShares,
  learnCalibration,
  majorityVote,
  speedConfidence,
  toJsonLines,
} from "./index.ts";
import type { Gait, GpsFix, Sample } from "./index.ts";
import { amplitudeSpectrum, fft } from "./fft.ts";
import { makeRng, synthesize, synthesizeSequence } from "./synth.ts";
import type { SynthOptions } from "./synth.ts";

// ---------------------------------------------------------------- helpers

const MOVING: Gait[] = ["walk", "trot", "canter"];

/** Accuracy of `pick(window)` against the true gait over many synthetic recordings. */
function accuracy(
  gait: Gait,
  seeds: number,
  pick: "gait" | "accelGait",
  extra: Partial<SynthOptions> = {},
  options: Parameters<typeof analyzeWindows>[2] = {},
): number {
  let ok = 0;
  let n = 0;
  for (let seed = 1; seed <= seeds; seed++) {
    const ws = analyzeWindows(synthesize({ ...extra, gait, durationS: 30, seed }), [], options);
    for (const w of ws.slice(2)) {
      // The first windows are skipped: the majority vote has no history yet.
      n++;
      if (w[pick] === gait) ok++;
    }
  }
  return ok / n;
}

/** Vertical-only signal at a given phone orientation (gravity on z). */
function verticalSignal(fn: (tSec: number) => number, seconds: number, startMs = 0): Sample[] {
  const rng = makeRng(99);
  const out: Sample[] = [];
  for (let i = 0; i < seconds * 50; i++) {
    const t = i / 50;
    out.push({
      t: startMs + t * 1000,
      x: 0.01 * (rng() - 0.5),
      y: 0.01 * (rng() - 0.5),
      z: 1 + fn(t) + 0.01 * (rng() - 0.5),
    });
  }
  return out;
}

// -------------------------------------------------------------------- FFT

test("fft finds a sinusoid in the right bin", () => {
  const n = 256;
  const sig = new Float64Array(n);
  for (let i = 0; i < n; i++) sig[i] = 0.5 * Math.sin((2 * Math.PI * 8 * i) / n);
  const re = Float64Array.from(sig);
  const im = new Float64Array(n);
  fft(re, im);
  let peak = 1;
  for (let k = 1; k < n / 2; k++) if (Math.hypot(re[k]!, im[k]!) > Math.hypot(re[peak]!, im[peak]!)) peak = k;
  assert.equal(peak, 8);
  assert.ok(Math.abs(Math.hypot(re[8]!, im[8]!) - 0.5 * (n / 2)) < 1e-9);
});

test("fft of an impulse is flat and rejects bad sizes", () => {
  const re = new Float64Array(16);
  const im = new Float64Array(16);
  re[0] = 1;
  fft(re, im);
  for (let k = 0; k < 16; k++) assert.ok(Math.abs(re[k]! - 1) < 1e-12 && Math.abs(im[k]!) < 1e-12);
  assert.throws(() => fft(new Float64Array(12), new Float64Array(12)));
});

test("amplitudeSpectrum recovers the amplitude of a sinusoid", () => {
  const n = 200;
  const sig = Array.from({ length: n }, (_, i) => 0.4 * Math.sin((2 * Math.PI * 1.5 * i) / 50));
  const spec = amplitudeSpectrum(sig, 512);
  const k = Math.round((1.5 * 512) / 50);
  assert.ok(Math.abs(spec[k]! - 0.4) < 0.03, `amplitude ${spec[k]}`);
});

// --------------------------------------------------------------- features

test("dominant frequency is accurate for any phone orientation", () => {
  for (let seed = 1; seed <= 30; seed++) {
    const samples = synthesize({ gait: "trot", durationS: 8, seed });
    const f = extractFeatures(samples, 0)!;
    // The synthetic trot lies within 1.35..1.75 Hz (+-3 % modulation).
    assert.ok(f.freq > 1.28 && f.freq < 1.82, `seed ${seed}: ${f.freq}`);
  }
  const f = extractFeatures(verticalSignal((t) => 0.5 * Math.sin(2 * Math.PI * 1.4 * t), 8), 0)!;
  assert.ok(Math.abs(f.freq - 1.4) < 0.05, `freq ${f.freq}`);
  assert.ok(Math.abs(f.verticalEnergy - 0.5 / Math.SQRT2) < 0.03);
  assert.ok(f.regularity > 0.9);
});

test("features are null when the window is not covered by samples", () => {
  const samples = synthesize({ gait: "walk", durationS: 8, seed: 1 });
  const gap = samples.filter((s) => s.t < 1500 || s.t > 6500);
  assert.equal(extractFeatures(gap, 0), null);
  assert.equal(extractFeatures([], 0), null);
  assert.ok(extractFeatures(samples, 0) !== null);
});

// --------------------------------------------------------- classification

test("accelerometer-only accuracy on synthetic data is high for every gait", () => {
  for (const g of GAITS) {
    const acc = accuracy(g, 40, "accelGait");
    assert.ok(acc >= 0.93, `${g}: accuracy ${acc}`);
  }
});

test("smoothed accuracy is at least as good as raw", () => {
  let raw = 0;
  let smooth = 0;
  for (const g of GAITS) {
    raw += accuracy(g, 20, "accelGait");
    smooth += accuracy(g, 20, "gait");
  }
  assert.ok(smooth / 4 >= 0.95, `smoothed ${smooth / 4}`);
  assert.ok(smooth >= raw - 1e-9);
});

test("trot and canter in the overlap range are separated by shape, not frequency", () => {
  // Same 1.7 Hz, symmetric vs. strong 2nd harmonic.
  const trot = extractFeatures(verticalSignal((t) => 0.5 * Math.sin(2 * Math.PI * 1.7 * t), 8), 0)!;
  const canter = extractFeatures(
    verticalSignal((t) => 0.4 * Math.sin(2 * Math.PI * 1.7 * t) + 0.28 * Math.sin(4 * Math.PI * 1.7 * t + 1), 8),
    0,
  )!;
  assert.equal(classifyWindow(trot, DEFAULT_CALIBRATION).gait, "trot");
  assert.equal(classifyWindow(canter, DEFAULT_CALIBRATION).gait, "canter");
});

test("aperiodic shaking is not mistaken for a gait", () => {
  const rng = makeRng(5);
  // Broadband noise with an energy well above the halt threshold (e.g. mucking out).
  const samples = verticalSignal(() => 0.3 * (rng() - 0.5) * 2, 30);
  const ws = analyzeWindows(samples);
  assert.ok(ws.length > 10);
  const halts = ws.filter((w) => w.accelGait === "halt").length;
  // Broadband noise can occasionally look peaky in a 4 s window (documented limitation).
  assert.ok(halts / ws.length >= 0.6, `halt share ${halts / ws.length}`);
});

test("a walk->trot->canter->halt sequence is followed with few transitions", () => {
  const seq = synthesizeSequence(
    [
      ["walk", 40],
      ["trot", 40],
      ["canter", 40],
      ["halt", 40],
    ],
    3,
  );
  const ws = analyzeWindows(seq);
  const changes = ws.filter((w, i) => i > 0 && w.gait !== ws[i - 1]!.gait).length;
  assert.equal(changes, 3, `gaits: ${ws.map((w) => w.gait[0]).join("")}`);
  // Each segment is recognised in its second half at the latest.
  const at = (sec: number) => ws.find((w) => w.startMs >= sec * 1000)!.gait;
  assert.equal(at(30), "walk");
  assert.equal(at(70), "trot");
  assert.equal(at(110), "canter");
  assert.equal(at(150), "halt");
});

// ------------------------------------------------------------- smoothing

test("majorityVote removes single-window outliers", () => {
  const raw: Gait[] = ["trot", "trot", "walk", "trot", "trot", "canter", "trot"];
  const out = raw.map((_, i) => majorityVote(raw.slice(0, i + 1), 3));
  assert.deepEqual(out.slice(2), ["trot", "trot", "trot", "trot", "trot"]);
});

test("majorityVote follows a real change after two windows and breaks ties by recency", () => {
  assert.equal(majorityVote(["walk", "walk", "trot"], 3), "walk");
  assert.equal(majorityVote(["walk", "walk", "trot", "trot"], 3), "trot");
  assert.equal(majorityVote(["walk", "trot"], 3), "trot"); // tie: newest wins
  assert.equal(majorityVote(["walk", "trot"], 1), "trot");
});

test("smoothing lowers the number of gait changes on noisy input", () => {
  // Trot recorded with heavy sensor noise flickers between classes.
  const noisy = synthesize({ gait: "trot", durationS: 120, seed: 4, noise: 1.0 });
  const changes = (ws: { gait: Gait }[]) => ws.filter((w, i) => i > 0 && w.gait !== ws[i - 1]!.gait).length;
  const raw = analyzeWindows(noisy, [], { smoothing: 1 });
  const smooth = analyzeWindows(noisy, [], { smoothing: 5 });
  assert.ok(changes(raw) > 0, "test signal should flicker without smoothing");
  assert.ok(changes(smooth) < changes(raw), `${changes(smooth)} vs ${changes(raw)}`);
});

// ------------------------------------------------------------- GPS fusion

test("gaitFromSpeed uses the spec limits", () => {
  assert.equal(gaitFromSpeed(0.5), "halt");
  assert.equal(gaitFromSpeed(6.9), "walk");
  assert.equal(gaitFromSpeed(7), "trot");
  assert.equal(gaitFromSpeed(16), "trot");
  assert.equal(gaitFromSpeed(16.1), "canter");
});

test("speedConfidence is 0 on a boundary and 1 well inside a class", () => {
  assert.equal(speedConfidence(7), 0);
  assert.equal(speedConfidence(11), 1);
  assert.ok(speedConfidence(8) > 0 && speedConfidence(8) < 1);
  assert.equal(speedConfidence(0), 1);
  assert.equal(speedConfidence(30), 1);
});

test("fusion: speed wins when the accelerometer is ambiguous", () => {
  assert.equal(fuseWithSpeed({ gait: "trot", confidence: 0.1 }, 22), "canter");
  assert.equal(fuseWithSpeed({ gait: "canter", confidence: 0.1 }, 10), "trot");
  assert.equal(fuseWithSpeed({ gait: "walk", confidence: 0.2 }, 11), "trot");
});

test("fusion: a confident accelerometer wins against a speed near a boundary", () => {
  assert.equal(fuseWithSpeed({ gait: "trot", confidence: 0.9 }, 6.5), "trot");
  assert.equal(fuseWithSpeed({ gait: "canter", confidence: 0.8 }, 15), "canter");
});

test("fusion: clearly wrong accelerometer results are overridden, missing speed is ignored", () => {
  assert.equal(fuseWithSpeed({ gait: "canter", confidence: 0.7 }, 4.5), "walk");
  assert.equal(fuseWithSpeed({ gait: "halt", confidence: 0.9 }, 12), "trot");
  assert.equal(fuseWithSpeed({ gait: "trot", confidence: 0.2 }, undefined), "trot");
  assert.equal(fuseWithSpeed({ gait: "trot", confidence: 0.2 }, Number.NaN), "trot");
  assert.equal(fuseWithSpeed({ gait: "trot", confidence: 0.2 }, -1), "trot");
});

test("fusion in the pipeline: ambiguous shape is resolved by GPS speed", () => {
  // 1.7 Hz with a 2nd harmonic right at the trot/canter threshold.
  const samples = verticalSignal(
    (t) => 0.5 * Math.sin(2 * Math.PI * 1.7 * t) + 0.5 * 0.35 * Math.sin(4 * Math.PI * 1.7 * t + 1),
    40,
  );
  const gps = (kmh: number): GpsFix[] => Array.from({ length: 40 }, (_, i) => ({ t: i * 1000, speedKmh: kmh }));
  const fast = analyzeWindows(samples, gps(22), { smoothing: 1 });
  const slow = analyzeWindows(samples, gps(10), { smoothing: 1 });
  assert.ok(fast.every((w) => w.fusedGait === "canter"), "22 km/h should give canter");
  assert.ok(slow.every((w) => w.fusedGait === "trot"), "10 km/h should give trot");
  assert.ok(fast.every((w) => w.speedKmh === 22));
});

test("GPS speed of a window is the mean of the fixes inside it", () => {
  const samples = synthesize({ gait: "walk", durationS: 12, seed: 2 });
  const gps: GpsFix[] = [
    { t: 100, speedKmh: 4 },
    { t: 1100, speedKmh: 6 },
    { t: 5000, speedKmh: 100 },
  ];
  const ws = analyzeWindows(samples, gps);
  assert.equal(ws[0]!.speedKmh, 5);
});

// ------------------------------------------------------------ calibration

test("calibration adapts to a phone on the arm (weaker signal)", () => {
  const arm = { placementScale: 0.25 };
  const before = ["walk", "halt"].map((g) => accuracy(g as Gait, 20, "accelGait", arm));
  assert.ok(before[0]! < 0.5, `default calibration should miss weak walking: ${before[0]}`);

  const segments = [
    { gait: "halt" as Gait, samples: synthesize({ gait: "halt", durationS: 20, seed: 901, ...arm }) },
    { gait: "walk" as Gait, samples: synthesize({ gait: "walk", durationS: 20, seed: 902, ...arm }) },
    { gait: "trot" as Gait, samples: synthesize({ gait: "trot", durationS: 20, seed: 903, ...arm }) },
    { gait: "canter" as Gait, samples: synthesize({ gait: "canter", durationS: 20, seed: 904, ...arm }) },
  ];
  const cal = learnCalibration(segments);
  assert.ok(cal.energyScale > 0.15 && cal.energyScale < 0.4, `energyScale ${cal.energyScale}`);
  for (const g of GAITS) {
    const acc = accuracy(g, 20, "accelGait", arm, { calibration: cal });
    assert.ok(acc >= 0.9, `${g} with calibration: ${acc}`);
  }
});

test("calibration keeps defaults for what it cannot learn and does not hurt the default placement", () => {
  const cal = learnCalibration([{ gait: "trot", samples: synthesize({ gait: "trot", durationS: 20, seed: 5 }) }]);
  assert.equal(cal.walkTrotFreq, DEFAULT_CALIBRATION.walkTrotFreq);
  assert.equal(cal.asymmetryThreshold, DEFAULT_CALIBRATION.asymmetryThreshold);
  assert.ok(Math.abs(cal.energyScale - 1) < 0.35, `energyScale ${cal.energyScale}`);
  for (const g of MOVING) assert.ok(accuracy(g, 10, "accelGait", {}, { calibration: cal }) >= 0.9);
  assert.throws(() => learnCalibration([{ gait: "walk", samples: [] }]));
});

// ----------------------------------------------------------------- stream

test("streaming and batch analysis give identical windows", () => {
  const samples = synthesize({ gait: "canter", durationS: 30, seed: 8 });
  const batch = analyzeWindows(samples);
  const stream = new GaitStream();
  let count = 0;
  for (const s of samples) count += stream.push(s).length;
  assert.equal(count, batch.length);
  assert.deepEqual(stream.windows, batch);
  // 30 s, 4 s windows, 2 s hop -> starts 0..24 s -> 13 windows.
  assert.equal(batch.length, 13);
});

test("a sensor dropout skips windows instead of producing garbage", () => {
  const samples = [
    ...synthesize({ gait: "trot", durationS: 20, seed: 1 }),
    ...synthesize({ gait: "trot", durationS: 20, seed: 1, startMs: 120_000 }),
  ];
  const ws = analyzeWindows(samples);
  assert.ok(ws.every((w) => w.endMs <= 20_000 || w.startMs >= 119_900), "no window inside the gap");
  assert.ok(ws.length >= 14);
});

// ----------------------------------------------------------------- export

test("exportWindows produces compact records and corrections", () => {
  const ws = analyzeWindows(synthesize({ gait: "walk", durationS: 20, seed: 6 }), [{ t: 0, speedKmh: 4.44 }]);
  const recs = exportWindows(ws);
  assert.equal(recs.length, ws.length);
  const r = recs[0]!;
  assert.equal(r.t, ws[0]!.startMs);
  assert.equal(r.f.length, FEATURE_ORDER.length);
  assert.equal(r.p, "walk");
  assert.equal(r.v, 4.4);
  assert.equal(r.c, undefined);
  assert.ok(JSON.stringify(r).length < 140, JSON.stringify(r));

  const fixed = correctRecords(recs, 4000, 8000, "trot");
  assert.deepEqual(
    fixed.map((x) => x.c),
    recs.map((x) => (x.t >= 4000 && x.t < 8000 ? "trot" : undefined)),
  );
  assert.equal(recs[2]!.c, undefined, "input is not mutated");
  assert.equal(toJsonLines(fixed).split("\n").length, fixed.length);
});

// ------------------------------------------------------------------ shares

test("gaitShares sums minutes and percentages per gait", () => {
  const seq = synthesizeSequence(
    [
      ["halt", 60],
      ["walk", 120],
      ["trot", 120],
      ["canter", 60],
    ],
    11,
  );
  const s = gaitShares(analyzeWindows(seq));
  const total = Object.values(s.shares).reduce((a, x) => a + x.percent, 0);
  assert.ok(Math.abs(total - 100) < 1e-9);
  // 6 minutes in total; the first window and the smoothing lag shift a few seconds.
  assert.ok(Math.abs(s.totalMinutes - 6) < 0.15, `${s.totalMinutes}`);
  assert.ok(Math.abs(s.shares.walk.minutes - 2) < 0.25, `walk ${s.shares.walk.minutes}`);
  assert.ok(Math.abs(s.shares.trot.minutes - 2) < 0.25, `trot ${s.shares.trot.minutes}`);
  assert.ok(Math.abs(s.shares.canter.percent - 16.7) < 4);

  const riding = gaitShares(analyzeWindows(seq), { excludeHalt: true });
  assert.equal(riding.shares.halt.minutes, 0);
  assert.ok(Math.abs(riding.shares.walk.percent - 40) < 6);
});

test("gaitShares prefers corrected labels and handles empty input", () => {
  const ws = analyzeWindows(synthesize({ gait: "walk", durationS: 20, seed: 12 }));
  ws[0]!.label = "canter";
  const withLabels = gaitShares(ws);
  const without = gaitShares(ws, { useLabels: false });
  assert.ok(withLabels.shares.canter.minutes > 0);
  assert.equal(without.shares.canter.minutes, 0);
  const empty = gaitShares([]);
  assert.equal(empty.totalMinutes, 0);
  assert.equal(empty.shares.walk.percent, 0);
});
