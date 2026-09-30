import assert from "node:assert/strict";
import { test } from "node:test";

import {
  dayDiff,
  formatClock,
  lastSeenLabel,
  localDate,
  sinceLabel,
  usualArrivalLabel,
  visibilityDescription,
  visibilityLabel,
} from "./presence-format.ts";

const BERLIN = "Europe/Berlin";

test("visibility labels", () => {
  assert.equal(visibilityLabel("all"), "Alle");
  assert.equal(visibilityLabel("only_day"), "Nur Tag");
  assert.equal(visibilityLabel("hidden"), "Versteckt");
  assert.match(visibilityDescription("only_day"), /keine Uhrzeiten/);
});

test("formatClock uses the stable time zone (summer and winter time)", () => {
  assert.equal(formatClock("2026-09-30T16:40:00Z", BERLIN), "18:40");
  assert.equal(formatClock("2026-01-15T17:40:00Z", BERLIN), "18:40");
  assert.equal(formatClock("2026-09-30T22:05:00Z", BERLIN), "00:05");
  assert.equal(formatClock("garbage", BERLIN), "");
});

test("sinceLabel", () => {
  assert.equal(sinceLabel("2026-09-30T16:40:00Z", BERLIN), "seit 18:40");
  assert.equal(sinceLabel(null, BERLIN), "heute da");
});

test("localDate crosses midnight in the stable time zone", () => {
  assert.equal(localDate("2026-09-30T22:30:00Z", BERLIN), "2026-10-01");
  assert.equal(localDate("2026-09-30T21:30:00Z", BERLIN), "2026-09-30");
});

test("dayDiff", () => {
  assert.equal(dayDiff("2026-09-29", "2026-09-30"), 1);
  assert.equal(dayDiff("2026-09-30", "2026-09-30"), 0);
  assert.equal(dayDiff("2026-02-27", "2026-03-02"), 3);
});

test("lastSeenLabel", () => {
  const today = "2026-09-30";
  const at = "2026-09-29T15:20:00Z";
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-30", last_seen_at: "2026-09-30T15:20:00Z" }, today, BERLIN), "heute, 17:20");
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-30", last_seen_at: null }, today, BERLIN), "heute");
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-29", last_seen_at: at }, today, BERLIN), "gestern, 17:20");
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-29", last_seen_at: null }, today, BERLIN), "gestern");
  // 2026-09-27 is a Sunday.
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-27", last_seen_at: null }, today, BERLIN), "Sonntag");
  assert.equal(lastSeenLabel({ last_seen_date: "2026-09-12", last_seen_at: null }, today, BERLIN), "12.09.");
});

test("usualArrivalLabel", () => {
  assert.equal(usualArrivalLabel(19), "kommt meist gegen 19 Uhr");
  assert.equal(usualArrivalLabel(0), "kommt meist gegen 0 Uhr");
  assert.equal(usualArrivalLabel(null), null);
  assert.equal(usualArrivalLabel(undefined), null);
  assert.equal(usualArrivalLabel(24), null);
});
