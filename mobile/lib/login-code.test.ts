import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import {
  cooldownRemainingSeconds,
  formatLoginCode,
  isCompleteLoginCode,
  loginCodeErrorMessage,
  normalizeLoginCode,
  resendLabel,
} from "./login-code.ts";

test("normalizeLoginCode keeps digits and cuts at six", () => {
  assert.equal(normalizeLoginCode("123456"), "123456");
  assert.equal(normalizeLoginCode("123 456"), "123456");
  assert.equal(normalizeLoginCode(" 123-456 "), "123456");
  assert.equal(normalizeLoginCode("Code: 123 456\n"), "123456");
  assert.equal(normalizeLoginCode("12345678"), "123456");
  assert.equal(normalizeLoginCode("abc"), "");
  assert.equal(normalizeLoginCode("012 345"), "012345");
});

test("isCompleteLoginCode", () => {
  assert.equal(isCompleteLoginCode("000123"), true);
  assert.equal(isCompleteLoginCode("12345"), false);
  assert.equal(isCompleteLoginCode("1234567"), false);
  assert.equal(isCompleteLoginCode("12 456"), false);
});

test("formatLoginCode groups by three", () => {
  assert.equal(formatLoginCode("12"), "12");
  assert.equal(formatLoginCode("123"), "123");
  assert.equal(formatLoginCode("1234"), "123 4");
  assert.equal(formatLoginCode("123456"), "123 456");
  assert.equal(formatLoginCode("123 456 789"), "123 456");
});

test("cooldown", () => {
  assert.equal(cooldownRemainingSeconds(60_000, 0), 60);
  assert.equal(cooldownRemainingSeconds(60_000, 59_001), 1);
  assert.equal(cooldownRemainingSeconds(60_000, 60_000), 0);
  assert.equal(cooldownRemainingSeconds(60_000, 90_000), 0);
});

test("resendLabel", () => {
  assert.equal(resendLabel(0), "Erneut senden");
  assert.equal(resendLabel(-3), "Erneut senden");
  assert.equal(resendLabel(45), "Erneut senden in 0:45");
  assert.equal(resendLabel(60), "Erneut senden in 1:00");
  assert.equal(resendLabel(5), "Erneut senden in 0:05");
});

test("loginCodeErrorMessage", () => {
  assert.match(loginCodeErrorMessage(new ApiError(401, "invalid_code", "x")), /Code falsch/);
  assert.match(loginCodeErrorMessage(new ApiError(429, "rate_limited", "x")), /Zu viele Versuche/);
  assert.match(loginCodeErrorMessage(new ApiError(0, "network", "x")), /Keine Verbindung/);
  assert.match(loginCodeErrorMessage(new Error("boom")), /schiefgelaufen/);
});
