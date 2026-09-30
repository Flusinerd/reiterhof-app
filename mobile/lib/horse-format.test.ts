import assert from "node:assert/strict";
import { test } from "node:test";

import {
  appointmentCompanionRoute,
  horseRoutes,
  ageFromBirthYear,
  ageText,
  documentKindLabel,
  dueDetailText,
  dueLevel,
  dueText,
  formatDate,
  healthKindLabel,
  joinParts,
  parseGermanDate,
  parseTime,
  parseWholeNumber,
  RIDER_RULES,
  ruleLabel,
  rulesSummary,
  sexLabel,
  telUrl,
  toggleRule,
} from "./horse-format.ts";

const now = new Date(2026, 9, 1);

test("age from birth year", () => {
  assert.equal(ageFromBirthYear(2015, now), 11);
  assert.equal(ageFromBirthYear(2026, now), 0);
  assert.equal(ageFromBirthYear(2030, now), null);
  assert.equal(ageFromBirthYear(null, now), null);
  assert.equal(ageFromBirthYear(undefined, now), null);
  assert.equal(ageFromBirthYear(0, now), null);
});

test("age text", () => {
  assert.equal(ageText(2015, now), "11 Jahre");
  assert.equal(ageText(2025, now), "1 Jahr");
  assert.equal(ageText(2026, now), "unter 1 Jahr");
  assert.equal(ageText(null, now), "");
});

test("sex labels", () => {
  assert.equal(sexLabel("mare"), "Stute");
  assert.equal(sexLabel("gelding"), "Wallach");
  assert.equal(sexLabel("stallion"), "Hengst");
  assert.equal(sexLabel(null), "");
  assert.equal(sexLabel("unicorn"), "");
});

test("joinParts skips empty parts", () => {
  assert.equal(joinParts(["Stute", "", null, undefined, false, "11 Jahre", "  "]), "Stute · 11 Jahre");
  assert.equal(joinParts([]), "");
});

test("due text", () => {
  assert.equal(dueText(-1), "überfällig");
  assert.equal(dueText(-30), "überfällig");
  assert.equal(dueText(0), "heute");
  assert.equal(dueText(1), "morgen");
  assert.equal(dueText(3), "in 3 Tagen");
  assert.equal(dueText(14), "in 14 Tagen");
  assert.equal(dueText(21), "in 3 Wochen");
  assert.equal(dueText(120), "in 4 Monaten");
  assert.equal(dueText(null), "kein Termin");
  assert.equal(dueText(undefined), "kein Termin");
});

test("due detail text", () => {
  assert.equal(dueDetailText(-1), "seit 1 Tag überfällig");
  assert.equal(dueDetailText(-11), "seit 11 Tagen überfällig");
  assert.equal(dueDetailText(6), "in 6 Tagen");
});

test("due level", () => {
  assert.equal(dueLevel(null), "none");
  assert.equal(dueLevel(-1), "overdue");
  assert.equal(dueLevel(0), "soon");
  assert.equal(dueLevel(7), "soon");
  assert.equal(dueLevel(8), "ok");
});

test("date formatting and parsing", () => {
  assert.equal(formatDate("2026-10-08"), "08.10.2026");
  assert.equal(formatDate(null), "");
  assert.equal(parseGermanDate("08.10.2026"), "2026-10-08");
  assert.equal(parseGermanDate(" 8.1.2027 "), "2027-01-08");
  assert.equal(parseGermanDate("8.10.26"), "2026-10-08");
  assert.equal(parseGermanDate("31.02.2026"), null);
  assert.equal(parseGermanDate("2026-10-08"), null);
  assert.equal(parseGermanDate(""), null);
  assert.equal(parseGermanDate("29.02.2028"), "2028-02-29");
});

test("time and number parsing", () => {
  assert.equal(parseTime("8:30"), "08:30");
  assert.equal(parseTime("08.05"), "08:05");
  assert.equal(parseTime("24:00"), null);
  assert.equal(parseTime("8:75"), null);
  assert.equal(parseTime("abc"), null);
  assert.equal(parseWholeNumber(" 42 "), 42);
  assert.equal(parseWholeNumber("4.2"), null);
  assert.equal(parseWholeNumber(""), null);
  assert.equal(parseWholeNumber("-3"), null);
});

test("kind labels", () => {
  assert.equal(healthKindLabel("farrier"), "Hufschmied");
  assert.equal(healthKindLabel("other-thing"), "other-thing");
  assert.equal(documentKindLabel("vaccination_record"), "Impfpass");
  assert.equal(documentKindLabel("x"), "x");
});

test("rider rule labels match the backend list", () => {
  assert.deepEqual(
    RIDER_RULES.map((r) => r.key),
    ["ride", "groom", "log_sessions", "report_observations", "take_week_slots", "hack_alone", "shows"],
  );
  assert.equal(ruleLabel("hack_alone"), "Allein ausreiten");
  assert.equal(ruleLabel("mystery"), "mystery");
  assert.equal(rulesSummary([]), "Keine Freigaben");
  assert.equal(rulesSummary(["groom", "ride"]), "Reiten, Pflegen");
  assert.equal(rulesSummary(["shows", "mystery"]), "Turniere, mystery");
});

test("toggleRule keeps canonical order", () => {
  assert.deepEqual(toggleRule(["shows"], "ride"), ["ride", "shows"]);
  assert.deepEqual(toggleRule(["ride", "shows"], "ride"), ["shows"]);
  assert.deepEqual(toggleRule([], "hack_alone"), ["hack_alone"]);
});

test("routes", () => {
  assert.equal(horseRoutes.detail("abc"), "/horses/abc");
  assert.equal(horseRoutes.emergency("abc"), "/horses/abc/emergency");
  assert.equal(horseRoutes.trainingProfile("abc"), "/horses/abc/training-profile");
  assert.equal(appointmentCompanionRoute("abc"), "/requests/new?type=appointment_companion&horse=abc");
});

test("tel url", () => {
  assert.equal(telUrl("+49 170 1234567"), "tel:+491701234567");
  assert.equal(telUrl("0170/123 45-67"), "tel:01701234567");
  assert.equal(telUrl("  "), null);
  assert.equal(telUrl("abc"), null);
  assert.equal(telUrl(null), null);
});
