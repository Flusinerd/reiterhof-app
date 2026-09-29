import assert from "node:assert/strict";
import { test } from "node:test";

import { COLOR_KEYS, COLOR_PAIRS, colorsForKey, isColorKey, resolveColorKey } from "./color-keys.ts";

test("every color key has a palette entry", () => {
  for (const key of COLOR_KEYS) {
    assert.match(COLOR_PAIRS[key].bg, /^#[0-9a-f]{6}$/);
    assert.match(COLOR_PAIRS[key].fg, /^#[0-9a-f]{6}$/);
  }
});

test("palette matches the design spec", () => {
  assert.deepEqual(colorsForKey("green"), { bg: "#a3c9ad", fg: "#1c3a27" });
  assert.deepEqual(colorsForKey("amber"), { bg: "#f0c380", fg: "#4a2a05" });
  assert.deepEqual(colorsForKey("blue"), { bg: "#b9c7ea", fg: "#1e3560" });
  assert.deepEqual(colorsForKey("rose"), { bg: "#f2b8c6", fg: "#5a1a2c" });
  assert.deepEqual(colorsForKey("violet"), { bg: "#cbb8e8", fg: "#3b1f63" });
  assert.deepEqual(colorsForKey("teal"), { bg: "#9fd6cf", fg: "#0f3f3a" });
  assert.deepEqual(colorsForKey("neutral"), { bg: "#e7e2d9", fg: "#44403c" });
});

test("unknown, empty and missing keys fall back to neutral", () => {
  for (const value of [undefined, null, "", "  ", "pink", "constructor"]) {
    assert.equal(resolveColorKey(value), "neutral");
  }
});

test("keys are normalized", () => {
  assert.equal(resolveColorKey(" Green "), "green");
  assert.equal(resolveColorKey("TEAL"), "teal");
});

test("isColorKey", () => {
  assert.equal(isColorKey("rose"), true);
  assert.equal(isColorKey("Rose"), false);
  assert.equal(isColorKey(3), false);
});
