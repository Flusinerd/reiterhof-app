import assert from "node:assert/strict";
import { test } from "node:test";

import { colors, fontFamilies, layout } from "./tokens.ts";

test("core colors match the design spec", () => {
  assert.equal(colors.background, "#f6f4ee");
  assert.equal(colors.primary.DEFAULT, "#2d5a3d");
  assert.equal(colors.accent.text, "#9a4d0b");
  assert.equal(colors.danger.soft, "#fde8e8");
  assert.equal(colors.gait["canter-light"], "#c2410c");
});

test("touch target and page padding", () => {
  assert.equal(layout.minTouchTarget, 44);
  assert.equal(layout.pagePadding, 24);
});

test("no zinc-like greys: all colors are hex values", () => {
  const flat = Object.values(colors).flatMap((v) => (typeof v === "string" ? [v] : Object.values(v)));
  for (const value of flat) assert.match(value, /^#[0-9a-f]{6}$/);
});

test("font families use the expo-google-fonts naming", () => {
  for (const family of Object.values(fontFamilies)) {
    assert.match(family, /^(Geist|Fraunces)_\d{3}[A-Za-z]+$/);
  }
});
