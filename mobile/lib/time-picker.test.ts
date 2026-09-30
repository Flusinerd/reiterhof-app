import assert from "node:assert/strict";
import { test } from "node:test";

import { dateToTime, timeToDate } from "./time-picker.ts";

test("timeToDate parses HH:MM into local hours and minutes", () => {
  const d = timeToDate("07:05");
  assert.equal(d.getHours(), 7);
  assert.equal(d.getMinutes(), 5);
});

test("timeToDate falls back to now for empty or invalid input", () => {
  for (const v of ["", "25:00", "abc"]) {
    assert.ok(Math.abs(timeToDate(v).getTime() - Date.now()) < 5000);
  }
});

test("dateToTime zero-pads", () => {
  assert.equal(dateToTime(new Date(2026, 0, 1, 9, 3)), "09:03");
  assert.equal(dateToTime(timeToDate("18:30")), "18:30");
});
