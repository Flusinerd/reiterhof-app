import assert from "node:assert/strict";
import { test } from "node:test";

import { flash } from "./flash.ts";

test("a result is kept until it is cleared, and listeners hear about both", () => {
  let calls = 0;
  const off = flash.subscribe(() => calls++);
  assert.equal(flash.get(), null);
  flash.set("Luna wurde gelöscht.");
  assert.equal(flash.get(), "Luna wurde gelöscht.");
  flash.clear();
  assert.equal(flash.get(), null);
  assert.equal(calls, 2);
  // Clearing nothing is not a change.
  flash.clear();
  assert.equal(calls, 2);
  off();
  flash.set("x");
  assert.equal(calls, 2);
  flash.clear();
});
