import assert from "node:assert/strict";
import { test } from "node:test";

import { formatInviteCode, normalizeEmail, normalizeInviteCode } from "./validation.ts";

test("normalizeEmail", () => {
  assert.equal(normalizeEmail("  Jan@Example.ORG "), "jan@example.org");
  assert.equal(normalizeEmail("nope"), null);
  assert.equal(normalizeEmail("a b@c.de"), null);
  assert.equal(normalizeEmail(""), null);
});

test("invite codes", () => {
  assert.equal(normalizeInviteCode(" abcd-efgh "), "ABCDEFGH");
  assert.equal(formatInviteCode("abcdefgh"), "ABCD-EFGH");
  assert.equal(formatInviteCode("abc"), "ABC");
  assert.equal(formatInviteCode("abcdefghijk"), "ABCD-EFGH");
});
