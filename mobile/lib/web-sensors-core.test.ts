import assert from "node:assert/strict";
import { test } from "node:test";

import { accelerationToG, fixFromPosition, geolocationFailure, STANDARD_GRAVITY } from "./web-sensors-core.ts";

test("m/s² become g", () => {
  const g = accelerationToG({ x: 0, y: 0, z: STANDARD_GRAVITY });
  assert.deepEqual(g, { x: 0, y: 0, z: 1 });
  const half = accelerationToG({ x: STANDARD_GRAVITY / 2, y: -STANDARD_GRAVITY, z: 0 });
  assert.deepEqual(half, { x: 0.5, y: -1, z: 0 });
});

test("unusable acceleration gives null", () => {
  assert.equal(accelerationToG(null), null);
  assert.equal(accelerationToG(undefined), null);
  assert.equal(accelerationToG({ x: null, y: 1, z: 1 }), null);
  assert.equal(accelerationToG({ x: NaN, y: 1, z: 1 }), null);
  assert.equal(accelerationToG({ x: 1, y: Infinity, z: 1 }), null);
});

test("a position becomes a raw fix", () => {
  const fix = fixFromPosition({
    timestamp: 1_700_000_000_000,
    coords: { latitude: 52.5, longitude: 13.4, altitude: 40, speed: 2.5, accuracy: 8 },
  });
  assert.deepEqual(fix, { t: 1_700_000_000_000, lat: 52.5, lon: 13.4, alt: 40, speed: 2.5, accuracy: 8 });
});

test("missing or invalid altitude and speed stay null", () => {
  const fix = fixFromPosition({
    timestamp: 1,
    coords: { latitude: 1, longitude: 2, altitude: null, speed: NaN, accuracy: 5 },
  });
  assert.equal(fix.alt, null);
  assert.equal(fix.speed, null);
});

test("geolocation error codes map to tracker failures", () => {
  assert.equal(geolocationFailure(1), "blocked");
  assert.equal(geolocationFailure(2), "services-off");
  assert.equal(geolocationFailure(3), "error");
  assert.equal(geolocationFailure(99), "error");
});
