import assert from "node:assert/strict";
import { test } from "node:test";

import { GaitStream, gaitShares } from "./gait/index.ts";
import type { Gait, GaitWindow } from "./gait/index.ts";
import { synthesizeSequence } from "./gait/synth.ts";
import {
  activeMs,
  addCorrection,
  addFix,
  apiDistance,
  apiGaitShares,
  applyCorrections,
  buildApiTrack,
  buildApiWindows,
  changeRein,
  createState,
  currentRein,
  gaitAt,
  gaitPoints,
  isPaused,
  pause,
  pointGait,
  reinSegments,
  resume,
  segmentByGait,
  stats,
  targetLabel,
  targetProgress,
  toggleStep,
  type GaitPoint,
  type RawFix,
  type TrackState,
} from "./tracking.ts";

const T0 = 1_700_000_000_000;
const sec = (s: number) => T0 + s * 1000;
const min = (m: number) => sec(m * 60);

/** Fix moving north at `speed` m/s from the start, one per second. */
function fixAt(s: number, speed = 2, extra: Partial<RawFix> = {}): RawFix {
  return { t: sec(s), lat: 52 + (speed * s) / 111_195, lon: 9, accuracy: 5, ...extra };
}

function win(startS: number, gait: Gait, extra: Partial<GaitWindow> = {}): GaitWindow {
  return {
    startMs: sec(startS),
    endMs: sec(startS + 4),
    weightMs: 2000,
    features: { freq: 1, verticalEnergy: 0.2, horizontalEnergy: 0.1, asymmetry: 0, harmonicRatio: 0, regularity: 0.9 },
    accelGait: gait,
    accelConfidence: 0.8,
    fusedGait: gait,
    gait,
    ...extra,
  };
}

/** Windows every 2 s from 0, gait per index. */
function windows(gaits: Gait[]): GaitWindow[] {
  return gaits.map((g, i) => win(i * 2, g));
}

// --- pause and time -----------------------------------------------------------------------

test("pause and resume exclude the paused time", () => {
  let s = createState("gps", sec(0));
  assert.equal(isPaused(s), false);
  s = pause(s, sec(60));
  assert.equal(isPaused(s), true);
  assert.equal(activeMs(s, sec(600)), 60_000, "paused time does not run");
  s = resume(s, sec(300));
  assert.equal(activeMs(s, sec(360)), 120_000);
  // idempotent
  assert.equal(pause(pause(s, sec(400)), sec(500)).intervals.length, 2);
  assert.equal(resume(s, sec(400)), s);
  assert.equal(activeMs(s, sec(0) - 5000), 0);
});

// --- fixes ------------------------------------------------------------------------------------

test("distance and average speed follow the accepted fixes", () => {
  let s = createState("gps", sec(0));
  for (let i = 0; i <= 60; i++) s = addFix(s, fixAt(i, 2));
  const st = stats(s, sec(60));
  assert.ok(Math.abs(st.distanceM - 120) < 1, String(st.distanceM));
  assert.ok(Math.abs(st.avgSpeedKmh - 7.2) < 0.2, String(st.avgSpeedKmh));
  assert.equal(st.activeSeconds, 60);
});

test("bad fixes are dropped: accuracy, jumps, old timestamps, paused", () => {
  let s = createState("gps", sec(0));
  s = addFix(s, fixAt(0));
  s = addFix(s, fixAt(1, 2, { accuracy: 80 }));
  assert.equal(s.fixes.length, 1, "inaccurate");
  s = addFix(s, { t: sec(2), lat: 52.1, lon: 9, accuracy: 5 });
  assert.equal(s.fixes.length, 1, "jump of 11 km in 2 s");
  s = addFix(s, fixAt(3));
  s = addFix(s, fixAt(3));
  s = addFix(s, fixAt(2));
  assert.equal(s.fixes.length, 2, "duplicate and out-of-order timestamps");
  s = pause(s, sec(4));
  s = addFix(s, fixAt(5));
  assert.equal(s.fixes.length, 2, "paused");
  s = resume(s, sec(10));
  s = addFix(s, fixAt(8));
  assert.equal(s.fixes.length, 2, "fix recorded before the resume");
  s = addFix(s, { t: sec(11), lat: NaN, lon: 9 });
  assert.equal(s.fixes.length, 2, "NaN");
});

test("no distance is added across a pause", () => {
  let s = createState("gps", sec(0));
  s = addFix(s, fixAt(0, 2));
  s = addFix(s, fixAt(10, 2));
  const before = s.distanceM;
  s = pause(s, sec(11));
  s = resume(s, sec(600));
  // 5 km away after the pause (a lift home): only the run after the resume counts
  s = addFix(s, { t: sec(601), lat: 52.05, lon: 9, accuracy: 5 });
  assert.equal(s.distanceM, before);
  assert.equal(s.fixes[s.fixes.length - 1]!.seg, 1);
  s = addFix(s, { t: sec(602), lat: 52.05002, lon: 9, accuracy: 5 });
  assert.ok(s.distanceM > before && s.distanceM < before + 5);
});

test("elevation gain in stats uses smoothed altitudes", () => {
  let s = createState("gps", sec(0));
  for (let i = 0; i < 100; i++) s = addFix(s, fixAt(i, 2, { alt: 100 + (i < 50 ? i * 0.6 : 30 - (i - 50) * 0.6) }));
  const gain = stats(s, sec(100)).elevationGainM;
  assert.ok(gain > 25 && gain <= 30, String(gain));
});

// --- rein accounting -----------------------------------------------------------------------------

test("rein segments account time per rein and the changes", () => {
  let s = createState("indoor", sec(0), "left");
  assert.equal(currentRein(s), "left");
  s = changeRein(s, min(10));
  s = changeRein(s, min(22));
  const seg = reinSegments(s, min(30));
  assert.deepEqual(seg, [
    { rein: "left", minutes: 10 },
    { rein: "right", minutes: 12 },
    { rein: "left", minutes: 8 },
  ]);
});

test("rein time excludes pauses", () => {
  let s = createState("indoor", sec(0), "left");
  s = pause(s, min(5));
  s = resume(s, min(15)); // 10 min pause on the left rein
  s = changeRein(s, min(20));
  const seg = reinSegments(s, min(30));
  assert.deepEqual(seg, [
    { rein: "left", minutes: 10 },
    { rein: "right", minutes: 10 },
  ]);
  // changing rein while paused is ignored
  s = pause(s, min(31));
  assert.equal(changeRein(s, min(32)), s);
});

test("a change at the very start or a mistaken tap folds into its neighbour", () => {
  let s = createState("indoor", sec(0), "left");
  s = changeRein(s, sec(1)); // "I start on the right rein"
  s = changeRein(s, min(10));
  assert.deepEqual(reinSegments(s, min(20)), [
    { rein: "right", minutes: 10 },
    { rein: "left", minutes: 10 },
  ]);

  let t = createState("indoor", sec(0), "left");
  t = changeRein(t, min(5));
  t = changeRein(t, min(5) + 2000); // flipped back after 2 s
  assert.deepEqual(reinSegments(t, min(10)), [{ rein: "left", minutes: 10 }]);
});

test("rein changes are only tracked indoors", () => {
  const s = createState("gps", sec(0));
  assert.equal(currentRein(s), null);
  assert.equal(changeRein(s, sec(10)), s);
  assert.deepEqual(reinSegments(s, sec(100)), []);
});

// --- gait timeline ---------------------------------------------------------------------------------

test("gaitAt finds the window around a time", () => {
  const ws = windows(["walk", "walk", "trot", "trot", "canter"]);
  // window i covers [i*2+1, i*2+3) s
  assert.equal(gaitAt(ws, sec(0.5)), null);
  assert.equal(gaitAt(ws, sec(2)), "walk");
  assert.equal(gaitAt(ws, sec(6)), "trot");
  assert.equal(gaitAt(ws, sec(10)), "canter");
  assert.equal(gaitAt(ws, sec(20)), null);
  assert.equal(gaitAt([], sec(1)), null);
  // labels win
  assert.equal(gaitAt([win(0, "walk", { label: "trot" })], sec(2)), "trot");
});

test("point gait falls back to OS speed, then to the fix distance", () => {
  const fix = { t: sec(50), lat: 52, lon: 9, seg: 0, speed: 3 }; // 10.8 km/h
  assert.equal(pointGait([], fix, undefined), "trot");
  const noSpeed = { t: sec(51), lat: 52 + 1.5 / 111_195, lon: 9, seg: 0 };
  const prev = { t: sec(50), lat: 52, lon: 9, seg: 0 };
  assert.equal(pointGait([], noSpeed, prev), "walk");
  assert.equal(pointGait([], noSpeed, undefined), "halt");
  assert.equal(pointGait(windows(["canter", "canter"]), { ...fix, t: sec(2) }, undefined), "canter");
});

// --- corrections --------------------------------------------------------------------------------------

test("a correction labels the recent wrong windows until the detector changes its mind", () => {
  // detector says trot for a while (but it was walk), then really switches to canter
  const ws = windows(["walk", "trot", "trot", "trot", "trot", "canter", "canter"]);
  // user taps "Schritt" at t = 9 s
  const out = applyCorrections(ws, [{ fromMs: sec(9), wrong: "trot", right: "walk" }]);
  assert.deepEqual(
    out.map((w) => w.label),
    [undefined, "walk", "walk", "walk", "walk", undefined, undefined],
  );
  // the input is not modified
  assert.equal(ws[1]!.label, undefined);
});

test("a correction only reaches back a few seconds", () => {
  const ws = windows(["trot", "trot", "trot", "trot", "trot", "trot"]); // 0..14 s
  const out = applyCorrections(ws, [{ fromMs: sec(14), wrong: "trot", right: "canter" }]);
  // windows ending after 14 - 6 = 8 s: starts >= 5 s -> indexes 3, 4, 5
  assert.deepEqual(
    out.map((w) => w.label),
    [undefined, undefined, undefined, "canter", "canter", "canter"],
  );
});

test("the next correction closes the previous one", () => {
  const ws = windows(["trot", "trot", "trot", "trot", "trot", "trot", "trot", "trot"]);
  const out = applyCorrections(ws, [
    { fromMs: sec(8), wrong: "trot", right: "walk" },
    { fromMs: sec(12), wrong: "trot", right: "canter" },
  ]);
  assert.equal(out[2]!.label, "walk");
  assert.equal(out[7]!.label, "canter");
  assert.equal(out[5]!.label, "walk", "starts at 10 s, before the second tap");
  assert.equal(out[6]!.label, "canter", "starts at the second tap");
});

test("corrections feed the shares and the track", () => {
  const ws = windows(["trot", "trot", "trot", "trot"]);
  const corrected = applyCorrections(ws, [{ fromMs: sec(8), wrong: "trot", right: "walk" }]);
  const shares = gaitShares(corrected);
  assert.equal(shares.shares.walk.percent, 100);
  assert.equal(shares.totalMinutes, (4 * 2000) / 60000);
  let s = createState("gps", sec(0));
  s = addFix(s, fixAt(2, 4)); // OS speed says nothing here; windows decide
  assert.equal(gaitPoints(s, corrected)[0]!.gait, "walk");
  const state = addCorrection(createState("indoor", sec(0)), sec(5), "trot", "walk");
  assert.equal(state.corrections.length, 1);
});

test("end to end with the real detector: a correction relabels synthetic windows", () => {
  const samples = synthesizeSequence(
    [
      ["walk", 30],
      ["trot", 30],
    ],
    5,
    { startMs: T0 },
  );
  const stream = new GaitStream();
  for (const smp of samples) stream.push({ ...smp, t: smp.t + T0 });
  assert.ok(stream.windows.length > 20);
  // the user claims the trot part is canter, tapping at 55 s
  const out = applyCorrections(stream.windows, [{ fromMs: sec(55), wrong: "trot", right: "canter" }]);
  const labelled = out.filter((w) => w.label === "canter");
  assert.ok(labelled.length >= 2 && labelled.length <= 6, String(labelled.length));
  assert.ok(labelled.every((w) => w.gait === "trot" && w.startMs >= sec(45)));
  assert.ok(gaitShares(out).shares.canter.percent > 0);
});

// --- segments and API payload ------------------------------------------------------------------------------

function gp(s: number, gait: Gait, seg = 0): GaitPoint {
  return { lat: 52 + s * 0.0001, lon: 9, t: sec(s), gait, seg };
}

test("segmentByGait closes the joints and breaks at pauses", () => {
  const pts = [gp(0, "walk"), gp(1, "walk"), gp(2, "walk"), gp(3, "trot"), gp(4, "trot"), gp(5, "walk"), gp(6, "walk", 1), gp(7, "walk", 1)];
  const segs = segmentByGait(pts);
  assert.deepEqual(
    segs.map((s) => [s.gait, s.coords.length]),
    [
      ["walk", 3], // 0-1-2
      ["trot", 3], // 2-3-4
      ["walk", 2], // 4-5
      ["walk", 2], // 6-7 after the pause: the stretch 5-6 is not drawn
    ],
  );
  assert.equal(segs[1]!.coords[0]!.lat, pts[2]!.lat);
  assert.deepEqual(segmentByGait([]), []);
  assert.deepEqual(segmentByGait([gp(0, "walk")]), []);
});

function ride(): TrackState {
  let s = createState("gps", sec(0));
  for (let i = 0; i <= 20; i++) s = addFix(s, fixAt(i, 1.5, { speed: 1.5 }));
  s = pause(s, sec(21));
  s = resume(s, sec(100));
  for (let i = 101; i <= 120; i++) s = addFix(s, { t: sec(i), lat: 52.01 + ((i - 101) * 4) / 111_195, lon: 9, accuracy: 5, speed: 4 });
  return s;
}

test("api track is simplified, keeps gait changes and marks the pause bridge", () => {
  const s = ride();
  const track = buildApiTrack(s, []);
  // straight lines: first, last, plus the neighbours of the gait change and of the pause
  assert.ok(track.length < s.fixes.length);
  assert.equal(track[0]!.t, 0);
  assert.equal(track[0]!.g, "walk"); // 1.5 m/s = 5.4 km/h
  const trotStart = track.find((p) => p.g === "trot");
  assert.ok(trotStart);
  const bridge = track.findIndex((p) => p.t === 101);
  assert.ok(bridge > 0);
  assert.equal(track[bridge]!.g, "halt", "first point after the pause bridges it");
  assert.equal(track[bridge + 1]!.g, "trot");
  assert.ok(track.every((p) => p.t >= 0 && Number.isFinite(p.lat)));
  assert.ok(track.every((p, i) => i === 0 || p.t >= track[i - 1]!.t));
  assert.deepEqual(buildApiTrack(createState("gps", sec(0)), []), []);
});

test("api track never exceeds the maximum", () => {
  let s = createState("gps", sec(0));
  for (let i = 0; i < 3000; i++) {
    s = addFix(s, { t: sec(i), lat: 52 + i * 0.00001, lon: 9 + Math.sin(i / 7) * 0.0003, accuracy: 5, speed: 2 });
  }
  const t = buildApiTrack(s, [], 200);
  assert.ok(t.length <= 200 && t.length > 10, String(t.length));
  assert.equal(t[t.length - 1]!.t, 2999);
});

test("api windows are relative to the start and capped keeping corrected ones", () => {
  const ws = Array.from({ length: 50 }, (_, i) => win(i * 2, "walk", i % 10 === 0 ? { label: "trot" } : {}));
  const all = buildApiWindows(ws, sec(0));
  assert.equal(all.length, 50);
  assert.equal(all[3]!.t, 6000);
  assert.equal(all[0]!.f.length, 6);
  assert.equal(all[0]!.c, "trot");
  const capped = buildApiWindows(ws, sec(0), 20);
  assert.equal(capped.length, 20);
  assert.equal(capped.filter((r) => r.c === "trot").length, 5, "corrected windows survive");
  assert.ok(capped.every((r, i) => i === 0 || r.t >= capped[i - 1]!.t));
});

test("gait shares and distance for the API", () => {
  const shares = apiGaitShares({ halt: 10, walk: 60.04, trot: 29.96, canter: 0 });
  const sum = Object.values(shares).reduce((a, b) => a + b, 0);
  assert.ok(sum <= 1);
  assert.equal(shares.walk, 0.6);
  assert.deepEqual(apiGaitShares({ halt: 0, walk: 0, trot: 0, canter: 0 }), {});
  assert.equal(apiDistance(7400.4), 7400);
  assert.equal(apiDistance(0.2), undefined);
  assert.equal(apiDistance(9_000_000), 500_000);
});

// --- target and checklist ----------------------------------------------------------------------------------

test("target progress and label", () => {
  assert.equal(targetProgress(600, 40), 0.25);
  assert.equal(targetProgress(4000, 40), 1);
  assert.equal(targetProgress(100, undefined), 0);
  assert.equal(targetLabel(600, 40), "noch 30 Min.");
  assert.equal(targetLabel(2400, 40), "Ziel erreicht");
  assert.equal(targetLabel(2700, 40), "5 Min. über dem Ziel");
  assert.equal(targetLabel(100, undefined), "");
});

test("checklist toggle keeps the indexes sorted", () => {
  assert.deepEqual(toggleStep([], 2), [2]);
  assert.deepEqual(toggleStep([2], 0), [0, 2]);
  assert.deepEqual(toggleStep([0, 2], 2), [0]);
});
