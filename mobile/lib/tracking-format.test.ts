import assert from "node:assert/strict";
import { test } from "node:test";

import { formatDistance, formatElevation, formatReinMinutes, formatSpeed, gaitLabel, reinLabel } from "./tracking-format.ts";

test("German formats", () => {
  assert.equal(formatDistance(850.4), "850 m");
  assert.equal(formatDistance(7421), "7,42 km");
  assert.equal(formatDistance(0), "0 m");
  assert.equal(formatSpeed(6.84), "6,8 km/h");
  assert.equal(formatElevation(41.6), "+42 m");
  assert.equal(formatReinMinutes(12.9), "12 Min.");
  assert.equal(gaitLabel("canter"), "Galopp");
  assert.equal(gaitLabel("halt"), "Halt");
  assert.equal(reinLabel("left"), "Linke Hand");
  assert.equal(reinLabel("right"), "Rechte Hand");
});
