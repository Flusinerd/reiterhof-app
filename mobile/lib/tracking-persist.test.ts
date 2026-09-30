import assert from "node:assert/strict";
import { test } from "node:test";

import { SNAPSHOT_VERSION, parseSnapshot, restoreSnapshot, serializeSnapshot, snapshotSummary, type Snapshot } from "./tracking-persist.ts";
import { activeMs, addFix, changeRein, createState, isPaused, reinSegments } from "./tracking.ts";

const T0 = 1_700_000_000_000;
const sec = (s: number) => T0 + s * 1000;

function snapshot(): Snapshot {
  let state = createState("gps", sec(0));
  for (let i = 0; i <= 30; i++) state = addFix(state, { t: sec(i), lat: 52 + (2 * i) / 111_195, lon: 9, accuracy: 5, alt: 100 });
  return { v: SNAPSHOT_VERSION, horse: "h1", activity: "hack", mode: "gps", state, windows: [], checked: [], savedAt: sec(31) };
}

test("snapshots round-trip through JSON", () => {
  const s = snapshot();
  const back = parseSnapshot(serializeSnapshot(s));
  assert.deepEqual(back, s);
});

test("parse rejects garbage, other versions and inconsistent modes", () => {
  assert.equal(parseSnapshot(null), null);
  assert.equal(parseSnapshot(""), null);
  assert.equal(parseSnapshot("{"), null);
  assert.equal(parseSnapshot("[]"), null);
  assert.equal(parseSnapshot(JSON.stringify({ ...snapshot(), v: 99 })), null);
  assert.equal(parseSnapshot(JSON.stringify({ ...snapshot(), mode: "indoor" })), null);
  assert.equal(parseSnapshot(JSON.stringify({ ...snapshot(), windows: null })), null);
  const noState = { ...snapshot() } as Record<string, unknown>;
  delete noState.state;
  assert.equal(parseSnapshot(JSON.stringify(noState)), null);
});

test("restore pauses at the last sign of life", () => {
  const s = snapshot();
  const state = restoreSnapshot(s);
  assert.equal(isPaused(state), true);
  assert.equal(activeMs(state, sec(9999)), 31_000, "the dead time does not count");
});

test("restore first takes the fixes that the background task collected", () => {
  const s = snapshot();
  const pending = [
    { t: sec(32), lat: 52 + 64 / 111_195, lon: 9, accuracy: 5 },
    { t: sec(33), lat: 52 + 66 / 111_195, lon: 9, accuracy: 5 },
  ];
  const state = restoreSnapshot(s, pending);
  assert.equal(state.fixes.length, s.state.fixes.length + 2);
  assert.equal(activeMs(state, sec(9999)), 33_000);
  assert.ok(state.distanceM > s.state.distanceM);
});

test("restore keeps an already paused session and its rein log", () => {
  let state = createState("indoor", sec(0));
  state = changeRein(state, sec(600));
  const s: Snapshot = { v: SNAPSHOT_VERSION, horse: "h", activity: "hall", mode: "indoor", state, windows: [], checked: [1], savedAt: sec(700) };
  const restored = restoreSnapshot(s);
  assert.equal(isPaused(restored), true);
  assert.deepEqual(reinSegments(restored, sec(5000)).map((r) => r.rein), ["left", "right"]);
  assert.equal(restoreSnapshot({ ...s, state: restored }), restored);
});

test("summary names the activity and the running time", () => {
  assert.equal(snapshotSummary(snapshot(), "Ausritt"), "Ausritt, seit 1 Min. unterwegs");
});
