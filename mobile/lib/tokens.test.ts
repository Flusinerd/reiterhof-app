import assert from "node:assert/strict";
import { test } from "node:test";

import { readFileSync } from "node:fs";

import { colors, flattenPalette, fontFamilies, hexToRgbChannels, layout, palettes } from "./tokens.ts";

test("core colors match the design spec (Abendlicht)", () => {
  assert.equal(colors, palettes.light);
  assert.equal(palettes.light.background, "#f4f0e6");
  assert.equal(palettes.dark.background, "#0e1210");
  assert.equal(palettes.dark.primary.DEFAULT, "#c9a45c");
  assert.equal(palettes.light.gait["canter-light"], "#c2410c");
});

test("both palettes have the same tokens", () => {
  const names = (p: typeof palettes.light) => flattenPalette(p).map(([n]) => n);
  assert.deepEqual(names(palettes.dark), names(palettes.light));
  assert.ok(names(palettes.light).includes("primary-foreground"));
});

test("global.css carries every token of both palettes", () => {
  const css = readFileSync(new URL("../global.css", import.meta.url), "utf8");
  const dark = css.indexOf("prefers-color-scheme: dark");
  assert.ok(dark > 0);
  for (const [scheme, from, to] of [["light", 0, dark], ["dark", dark, css.length]] as const) {
    const part = css.slice(from, to);
    for (const [name, value] of flattenPalette(palettes[scheme])) {
      assert.ok(part.includes(`--color-${name}: ${hexToRgbChannels(value)};`), `${scheme} --color-${name}`);
    }
  }
});

test("hex to rgb channels", () => {
  assert.equal(hexToRgbChannels("#c9a45c"), "201 164 92");
});

test("touch target and page padding", () => {
  assert.equal(layout.minTouchTarget, 44);
  assert.equal(layout.pagePadding, 24);
});

test("no zinc-like greys: all colors are hex values", () => {
  for (const p of [palettes.light, palettes.dark]) {
    for (const [, value] of flattenPalette(p)) assert.match(value, /^#[0-9a-f]{6}$/);
  }
});

test("font families use the expo-google-fonts naming", () => {
  for (const family of Object.values(fontFamilies)) {
    assert.match(family, /^(Geist|Fraunces)_\d{3}[A-Za-z]+(_Italic)?$/);
  }
});
