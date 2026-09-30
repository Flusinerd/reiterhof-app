import assert from "node:assert/strict";
import { test } from "node:test";

import { backoffDelay } from "./backoff.ts";

const noJitter = { jitter: 0 };

test("doubles up to the cap", () => {
  assert.deepEqual(
    [0, 1, 2, 3, 4, 5, 6, 10].map((n) => backoffDelay(n, noJitter)),
    [1000, 2000, 4000, 8000, 16000, 30000, 30000, 30000],
  );
});

test("jitter stays within +-25 % of the delay", () => {
  assert.equal(backoffDelay(2, { random: () => 0 }), 3000);
  assert.equal(backoffDelay(2, { random: () => 0.999999 }), 5000);
  assert.equal(backoffDelay(2, { random: () => 0.5 }), 4000);
});

test("ignores negative and huge attempts", () => {
  assert.equal(backoffDelay(-3, noJitter), 1000);
  assert.equal(backoffDelay(10_000, noJitter), 30000);
  assert.equal(backoffDelay(1.9, noJitter), 2000);
});
