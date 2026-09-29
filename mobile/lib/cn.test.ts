import assert from "node:assert/strict";
import { test } from "node:test";

import { cn } from "./cn.ts";

test("joins and drops falsy values", () => {
  assert.equal(cn("a", false, undefined, null, "b"), "a b");
});

test("later utilities win", () => {
  assert.equal(cn("p-5", "p-2"), "p-2");
  assert.equal(cn("text-muted", "text-primary"), "text-primary");
});

test("custom font sizes do not clash with text colors", () => {
  assert.equal(cn("text-title", "text-foreground"), "text-title text-foreground");
  assert.equal(cn("text-body", "text-hero"), "text-hero");
});

test("custom font families conflict with each other", () => {
  assert.equal(cn("font-sans", "font-display"), "font-display");
});
