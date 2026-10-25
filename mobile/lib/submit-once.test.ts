import assert from "node:assert/strict";
import { test } from "node:test";

import { createSubmitOnce } from "./submit-once.ts";

test("a call for an id that is still running is ignored", async () => {
  const once = createSubmitOnce(1500, () => 0);
  let release!: () => void;
  const gate = new Promise<void>((resolve) => (release = resolve));
  let calls = 0;
  const first = once.run("a", async () => (calls++, await gate, "first"));
  assert.equal(await once.run("a", async () => (calls++, "second")), undefined);
  assert.equal(calls, 1);
  release();
  assert.equal(await first, "first");
});

test("other ids are independent", async () => {
  const once = createSubmitOnce(1500, () => 0);
  assert.equal(await once.run("a", async () => 1), 1);
  assert.equal(await once.run("b", async () => 2), 2);
});

test("a repeat within the cooldown after a success is ignored, later ones run", async () => {
  let time = 1000;
  const once = createSubmitOnce(1500, () => time);
  assert.equal(await once.run("a", async () => "ok"), "ok");
  time += 1499;
  assert.equal(await once.run("a", async () => "again"), undefined);
  time += 1;
  assert.equal(await once.run("a", async () => "later"), "later");
});

test("a failure starts no cooldown and the error reaches the caller", async () => {
  const once = createSubmitOnce(1500, () => 0);
  await assert.rejects(once.run("a", async () => Promise.reject(new Error("offline"))), /offline/);
  assert.equal(await once.run("a", async () => "retry"), "retry");
});
