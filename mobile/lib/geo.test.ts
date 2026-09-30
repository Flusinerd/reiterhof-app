import assert from "node:assert/strict";
import { test } from "node:test";

import { elevationGain, haversineM, movingAverage, pathLengthM, simplifyPolyline, simplifyToMax } from "./geo.ts";

test("haversine matches known distances", () => {
  assert.equal(haversineM({ lat: 52, lon: 9 }, { lat: 52, lon: 9 }), 0);
  // one degree of latitude is about 111.2 km
  const d = haversineM({ lat: 52, lon: 9 }, { lat: 53, lon: 9 });
  assert.ok(Math.abs(d - 111_195) < 200, String(d));
  // Hannover - Hamburg airport is roughly 132 km
  const hh = haversineM({ lat: 52.3759, lon: 9.732 }, { lat: 53.5511, lon: 9.9937 });
  assert.ok(hh > 130_000 && hh < 135_000, String(hh));
  // symmetric
  assert.equal(
    haversineM({ lat: 10, lon: 20 }, { lat: 11, lon: 22 }),
    haversineM({ lat: 11, lon: 22 }, { lat: 10, lon: 20 }),
  );
});

test("longitude degrees shrink with latitude", () => {
  const equator = haversineM({ lat: 0, lon: 0 }, { lat: 0, lon: 1 });
  const north = haversineM({ lat: 60, lon: 0 }, { lat: 60, lon: 1 });
  assert.ok(Math.abs(north / equator - 0.5) < 0.01);
});

test("path length sums the segments", () => {
  const pts = [
    { lat: 52, lon: 9 },
    { lat: 52.001, lon: 9 },
    { lat: 52.002, lon: 9 },
  ];
  assert.ok(Math.abs(pathLengthM(pts) - 222.4) < 1);
  assert.equal(pathLengthM([]), 0);
  assert.equal(pathLengthM([pts[0]!]), 0);
});

test("moving average keeps the length and smooths a spike", () => {
  const out = movingAverage([0, 0, 10, 0, 0], 3);
  assert.equal(out.length, 5);
  assert.ok(Math.abs(out[2]! - 10 / 3) < 1e-9);
});

test("elevation gain ignores GPS noise but counts real climbs", () => {
  // noise of +-1 m around a flat 100 m
  const flat = Array.from({ length: 100 }, (_, i) => 100 + (i % 2 === 0 ? 1 : -1));
  assert.equal(elevationGain(flat), 0);

  // steady climb of 40 m in 0.5 m steps, then descent
  const up = Array.from({ length: 80 }, (_, i) => 100 + i * 0.5);
  const down = Array.from({ length: 80 }, (_, i) => 140 - i * 0.5);
  const gain = elevationGain([...up, ...down]);
  assert.ok(gain > 35 && gain <= 40, String(gain));
  assert.equal(elevationGain(down), 0);

  // two hills count twice
  const twoHills = [...up, ...down, ...up];
  assert.ok(elevationGain(twoHills) > 70);

  assert.equal(elevationGain([]), 0);
  assert.equal(elevationGain([100]), 0);
  assert.equal(elevationGain([100, NaN, 100]), 0);
});

test("elevation gain: noisy climb is not inflated", () => {
  let seed = 7;
  const rnd = () => {
    seed = (seed * 1664525 + 1013904223) % 4294967296;
    return seed / 4294967296 - 0.5;
  };
  // true climb 30 m over 300 samples with +-1.5 m noise
  const series = Array.from({ length: 300 }, (_, i) => 100 + (i / 300) * 30 + rnd() * 3);
  const gain = elevationGain(series);
  assert.ok(gain > 24 && gain < 40, String(gain));
});

test("simplify drops collinear points and keeps corners", () => {
  const line = Array.from({ length: 11 }, (_, i) => ({ lat: 52 + i * 0.0001, lon: 9 }));
  assert.equal(simplifyPolyline(line, 1).length, 2);

  // L shape: the corner has to stay
  const corner = [
    ...Array.from({ length: 6 }, (_, i) => ({ lat: 52 + i * 0.0002, lon: 9 })),
    ...Array.from({ length: 5 }, (_, i) => ({ lat: 52.001, lon: 9 + (i + 1) * 0.0003 })),
  ];
  const out = simplifyPolyline(corner, 1);
  assert.equal(out.length, 3);
  assert.deepEqual(out[1], { lat: 52.001, lon: 9 });
  assert.equal(out[0], corner[0]);
  assert.equal(out[2], corner[corner.length - 1]);
});

test("simplify keeps forced points and works on tiny inputs", () => {
  const line = Array.from({ length: 11 }, (_, i) => ({ lat: 52 + i * 0.0001, lon: 9 }));
  const out = simplifyPolyline(line, 1, (i) => i === 5);
  assert.equal(out.length, 3);
  assert.equal(out[1], line[5]);
  assert.deepEqual(simplifyPolyline([], 1), []);
  assert.equal(simplifyPolyline([line[0]!], 1).length, 1);
  assert.equal(simplifyPolyline([line[0]!, line[1]!], 1).length, 2);
});

test("simplify tolerance is in metres", () => {
  // 5 m sideways wiggle in the middle of a 100 m line
  const pts = [
    { lat: 52, lon: 9 },
    { lat: 52.00045, lon: 9 + 0.00007 }, // ~5 m off the line
    { lat: 52.0009, lon: 9 },
  ];
  assert.equal(simplifyPolyline(pts, 10).length, 2);
  assert.equal(simplifyPolyline(pts, 2).length, 3);
});

test("simplifyToMax caps the length and keeps the ends", () => {
  // a zig-zag with 2000 points
  const pts = Array.from({ length: 2000 }, (_, i) => ({
    lat: 52 + i * 0.00001,
    lon: 9 + (i % 2 === 0 ? 0 : 0.0001) + Math.sin(i / 50) * 0.001,
  }));
  const out = simplifyToMax(pts, 100);
  assert.ok(out.length <= 100, String(out.length));
  assert.ok(out.length >= 2);
  assert.equal(out[0], pts[0]);
  assert.equal(out[out.length - 1], pts[pts.length - 1]);
  // forcing more points than allowed still yields the cap
  const forced = simplifyToMax(pts, 50, () => true);
  assert.equal(forced.length, 50);
  assert.equal(forced[forced.length - 1], pts[pts.length - 1]);
});
