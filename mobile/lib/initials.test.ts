import assert from "node:assert/strict";
import { test } from "node:test";

import { initialOf } from "./initials.ts";

test("uses the upper-cased first letter", () => {
  assert.equal(initialOf("Luna"), "L");
  assert.equal(initialOf("cookie"), "C");
});

test("trims whitespace", () => {
  assert.equal(initialOf("  Nala"), "N");
});

test("handles umlauts and astral characters", () => {
  assert.equal(initialOf("änne"), "Ä");
  assert.equal(initialOf("\u{1F434}x"), "\u{1F434}");
});

test("falls back to ? for empty input", () => {
  for (const value of ["", "   ", null, undefined]) {
    assert.equal(initialOf(value), "?");
  }
});
