import assert from "node:assert/strict";
import { test } from "node:test";

import { createPushGuard } from "./push-once.ts";

test("the second push to the same target within the window is ignored", () => {
  let time = 1000;
  const allow = createPushGuard(600, () => time);
  assert.equal(allow("/requests/1"), true);
  time += 599;
  assert.equal(allow("/requests/1"), false);
});

test("the same target can be opened again once the window is over", () => {
  let time = 1000;
  const allow = createPushGuard(600, () => time);
  assert.equal(allow("/requests/1"), true);
  time += 600;
  assert.equal(allow("/requests/1"), true);
});

test("another target is never held back", () => {
  let time = 1000;
  const allow = createPushGuard(600, () => time);
  assert.equal(allow("/requests/1"), true);
  time += 10;
  assert.equal(allow("/requests/2"), true);
  // Going back to the first one right after is a new decision, not a double tap.
  time += 10;
  assert.equal(allow("/requests/1"), true);
});

test("an ignored push does not extend the window", () => {
  let time = 1000;
  const allow = createPushGuard(600, () => time);
  assert.equal(allow("/a"), true);
  time += 500;
  assert.equal(allow("/a"), false);
  time += 150;
  assert.equal(allow("/a"), true);
});
