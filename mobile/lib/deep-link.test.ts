import assert from "node:assert/strict";
import { test } from "node:test";

import { isPlausibleToken, parseVerifyLink } from "./deep-link.ts";

const TOKEN = "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789-_AbCdE";

test("parses the app link", () => {
  assert.equal(parseVerifyLink(`stallfunk://auth/verify?token=${TOKEN}`), TOKEN);
});

test("parses the https fallback link", () => {
  assert.equal(parseVerifyLink(`https://api.example.org/auth/verify?token=${TOKEN}`), TOKEN);
});

test("finds the token among other parameters", () => {
  assert.equal(parseVerifyLink(`stallfunk://auth/verify?a=1&token=${TOKEN}&b=2`), TOKEN);
});

test("rejects other links and bad tokens", () => {
  assert.equal(parseVerifyLink(`stallfunk://other/path?token=${TOKEN}`), null);
  assert.equal(parseVerifyLink("stallfunk://auth/verify"), null);
  assert.equal(parseVerifyLink("stallfunk://auth/verify?token=short"), null);
  assert.equal(parseVerifyLink("stallfunk://auth/verify?token=%E0%A4%A"), null);
  assert.equal(parseVerifyLink(`ftp://x/auth/verify?token=${TOKEN}`), null);
});

test("isPlausibleToken", () => {
  assert.equal(isPlausibleToken(TOKEN), true);
  assert.equal(isPlausibleToken(undefined), false);
  assert.equal(isPlausibleToken("has space has space has space"), false);
});
