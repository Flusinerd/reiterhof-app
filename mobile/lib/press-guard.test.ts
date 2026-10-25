import assert from "node:assert/strict";
import { test } from "node:test";

import { isThenable, runPress, type PressGuard } from "./press-guard.ts";

function setup() {
  const guard: PressGuard = { locked: false };
  const events: string[] = [];
  const hooks = { start: () => events.push("start"), end: () => events.push("end") };
  return { guard, events, hooks };
}

function deferred() {
  let resolve!: (v?: unknown) => void;
  let reject!: (e?: unknown) => void;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

test("a handler that returns a promise locks the button until it settles", async () => {
  const { guard, events, hooks } = setup();
  const d = deferred();
  let calls = 0;
  assert.equal(runPress(guard, () => (calls++, d.promise), hooks), true);
  assert.equal(guard.locked, true);
  // A second press before the promise settles is ignored (the double tap).
  assert.equal(runPress(guard, () => (calls++, d.promise), hooks), false);
  assert.equal(calls, 1);
  assert.deepEqual(events, ["start"]);
  d.resolve();
  await d.promise;
  await Promise.resolve();
  assert.equal(guard.locked, false);
  assert.deepEqual(events, ["start", "end"]);
  // And presses work again afterwards.
  assert.equal(runPress(guard, () => (calls++, undefined), hooks), true);
  assert.equal(calls, 2);
});

test("a rejected promise unlocks the button too, without throwing again", async () => {
  const { guard, events, hooks } = setup();
  const d = deferred();
  runPress(guard, () => d.promise, hooks);
  d.reject(new Error("offline"));
  await d.promise.catch(() => undefined);
  await Promise.resolve();
  assert.equal(guard.locked, false);
  assert.deepEqual(events, ["start", "end"]);
});

test("a plain handler locks nothing and shows no spinner", () => {
  const { guard, events, hooks } = setup();
  let calls = 0;
  assert.equal(runPress(guard, () => void calls++, hooks), true);
  assert.equal(runPress(guard, () => void calls++, hooks), true);
  assert.equal(calls, 2);
  assert.equal(guard.locked, false);
  assert.deepEqual(events, []);
});

test("isThenable", () => {
  assert.equal(isThenable(Promise.resolve()), true);
  assert.equal(isThenable({ then: () => undefined }), true);
  assert.equal(isThenable(undefined), false);
  assert.equal(isThenable(null), false);
  assert.equal(isThenable(() => undefined), false);
  assert.equal(isThenable({ then: 1 }), false);
});
