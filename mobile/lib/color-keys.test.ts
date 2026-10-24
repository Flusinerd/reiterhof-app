import assert from "node:assert/strict";
import { test } from "node:test";

import { COLOR_KEYS, COLOR_PAIRS, DARK_COLOR_PAIRS, colorsForKey, isColorKey, resolveColorKey } from "./color-keys.ts";

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

test("dark pairs cover every key and keep light unchanged", () => {
  for (const key of COLOR_KEYS) {
    assert.match(DARK_COLOR_PAIRS[key].bg, /^#[0-9a-f]{6}$/);
    assert.match(DARK_COLOR_PAIRS[key].fg, /^#[0-9a-f]{6}$/);
    assert.deepEqual(colorsForKey(key), COLOR_PAIRS[key]);
    assert.deepEqual(colorsForKey(key, "light"), COLOR_PAIRS[key]);
    assert.deepEqual(colorsForKey(key, "dark"), DARK_COLOR_PAIRS[key]);
  }
});

test("dark fields are darker than the light ones and the initial is lighter than the field", () => {
  const luma = (hex: string) => {
    const n = parseInt(hex.slice(1), 16);
    return 0.299 * ((n >> 16) & 255) + 0.587 * ((n >> 8) & 255) + 0.114 * (n & 255);
  };
  for (const key of COLOR_KEYS) {
    assert.ok(luma(DARK_COLOR_PAIRS[key].bg) < luma(COLOR_PAIRS[key].bg), key);
    assert.ok(luma(DARK_COLOR_PAIRS[key].fg) > luma(DARK_COLOR_PAIRS[key].bg) + 60, key);
  }
});

test("unknown keys fall back to neutral in the dark scheme too", () => {
  assert.deepEqual(colorsForKey("pink", "dark"), DARK_COLOR_PAIRS.neutral);
});
