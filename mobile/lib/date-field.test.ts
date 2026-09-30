import assert from "node:assert/strict";
import { test } from "node:test";

import { dateToIso, isoToDate } from "./date-field.ts";

test("isoToDate parses into a local calendar day", () => {
  const d = isoToDate("2026-05-17");
  assert.equal(d.getFullYear(), 2026);
  assert.equal(d.getMonth(), 4);
  assert.equal(d.getDate(), 17);
});

test("dateToIso zero-pads month and day", () => {
  assert.equal(dateToIso(new Date(2026, 0, 5)), "2026-01-05");
  assert.equal(dateToIso(new Date(2026, 11, 31)), "2026-12-31");
});

test("isoToDate and dateToIso round trip", () => {
  for (const v of ["2026-01-01", "2026-05-17", "2026-12-31", "2028-02-29"]) {
    assert.equal(dateToIso(isoToDate(v)), v);
  }
});

test("isoToDate falls back to today for empty or invalid input", () => {
  const today = dateToIso(new Date());
  for (const v of ["", "abc", "2026-13-01", "2026-02-30", "17.05.2026"]) {
    assert.equal(dateToIso(isoToDate(v)), today, v);
  }
});

test("the calendar day does not shift on DST change days", () => {
  for (const v of ["2026-03-29", "2026-10-25", "2026-03-28", "2026-10-26"]) {
    assert.equal(dateToIso(isoToDate(v)), v);
  }
});

test("dateToIso keeps the local day around midnight", () => {
  assert.equal(dateToIso(new Date(2026, 2, 29, 0, 0, 0)), "2026-03-29");
  assert.equal(dateToIso(new Date(2026, 2, 29, 23, 59, 59)), "2026-03-29");
});
