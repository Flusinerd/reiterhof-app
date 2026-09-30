import assert from "node:assert/strict";
import { test } from "node:test";

import {
  acceptLabel,
  addDays,
  addMonths,
  buildRule,
  CREATABLE_TYPES,
  describeRule,
  eventRange,
  filterParams,
  formatDay,
  monthGrid,
  monthTitle,
  formatInstant,
  formatWhen,
  helperCountText,
  helperStatusText,
  isRequestType,
  normalizeTime,
  parseDate,
  payloadDetails,
  relativeDay,
  reminderAt,
  requestTitle,
  statusBadge,
  taskChips,
  taskLabel,
  toIsoDate,
  toQuery,
  typeMeta,
  REQUEST_TYPES,
} from "./requests.ts";

test("every type has a German label, an icon and an accept text", () => {
  assert.equal(REQUEST_TYPES.length, 7);
  for (const t of REQUEST_TYPES) {
    assert.ok(t.label.length > 0 && t.icon.length > 0 && t.hint.length > 0, t.type);
  }
  assert.equal(typeMeta("show_helper").label, "Turniertrottel");
  assert.equal(typeMeta("exercise").label, "Bewegen");
  assert.equal(typeMeta("nope").type, "other");
  assert.equal(typeMeta(null).type, "other");
  assert.equal(acceptLabel("ride_share"), "Ich komme mit");
  assert.equal(acceptLabel("show_helper"), "Ich komme mit");
  assert.equal(acceptLabel("feed_or_turnout"), "Mach ich");
  assert.ok(!CREATABLE_TYPES.some((t) => t.type === "blanket"));
  assert.ok(isRequestType("blanket") && !isRequestType("party") && !isRequestType(undefined));
});

test("formats the day and time like 'Mi, 2. Okt · 18:00'", () => {
  assert.equal(formatDay("2026-10-02"), "Fr, 2. Okt");
  assert.equal(formatDay("2026-09-30"), "Mi, 30. Sep");
  assert.equal(formatWhen({ date: "2026-09-30", time_from: "18:00" }), "Mi, 30. Sep · 18:00");
  assert.equal(formatWhen({ date: "2026-09-30", time_from: "18:00", time_to: "19:30" }), "Mi, 30. Sep · 18:00–19:30");
  assert.equal(formatWhen({ date: "2026-03-04" }), "Mi, 4. Mär");
  assert.equal(formatWhen({ date: "2026-10-05", date_end: "2026-10-07" }), "Mo, 5. Okt – Mi, 7. Okt");
  assert.equal(formatWhen({ date: "2026-10-05", date_end: "2026-10-07", time_from: "08:00" }), "Mo, 5. Okt – Mi, 7. Okt · 08:00");
  assert.equal(formatWhen({ date: "2026-10-05", date_end: "2026-10-05" }), "Mo, 5. Okt");
  assert.equal(formatDay("kaputt"), "kaputt");
});

test("date helpers do not depend on the time zone", () => {
  assert.deepEqual(parseDate("2026-02-28"), { year: 2026, month: 2, day: 28, weekday: 6 });
  assert.equal(parseDate("2026-02-30"), null);
  assert.equal(parseDate("2026-2-3"), null);
  assert.equal(addDays("2026-02-27", 2), "2026-03-01");
  assert.equal(addDays("2026-10-24", 2), "2026-10-26"); // DST switch
  assert.equal(addDays("2026-01-01", -1), "2025-12-31");
  assert.equal(toIsoDate(new Date(2026, 0, 5, 23, 59)), "2026-01-05");
  assert.equal(relativeDay("2026-10-02", "2026-10-02"), "Heute");
  assert.equal(relativeDay("2026-10-03", "2026-10-02"), "Morgen");
  assert.equal(relativeDay("2026-10-05", "2026-10-02"), "Mo, 5. Okt");
});

test("normalizeTime accepts common spellings", () => {
  assert.equal(normalizeTime("18:30"), "18:30");
  assert.equal(normalizeTime("1830"), "18:30");
  assert.equal(normalizeTime("830"), "08:30");
  assert.equal(normalizeTime("18.30"), "18:30");
  assert.equal(normalizeTime("8"), "08:00");
  assert.equal(normalizeTime(" 7:5 "), "07:05");
  for (const bad of ["", "24:00", "12:60", "abc", "1:2:3", "12345"]) {
    assert.equal(normalizeTime(bad), null, bad);
  }
});

test("eventRange builds timed and all-day events", () => {
  const timed = eventRange({ date: "2026-10-02", time_from: "17:00", time_to: "18:30" });
  assert.ok(timed && !timed.allDay);
  assert.equal(timed.start.getHours(), 17);
  assert.equal(timed.end.getHours(), 18);
  assert.equal(timed.end.getMinutes(), 30);
  const noEnd = eventRange({ date: "2026-10-02", time_from: "17:00" });
  assert.ok(noEnd);
  assert.equal(noEnd.end.getTime() - noEnd.start.getTime(), 3_600_000);
  const range = eventRange({ date: "2026-10-05", date_end: "2026-10-07" });
  assert.ok(range && range.allDay);
  assert.equal(range.start.getDate(), 5);
  assert.equal(range.end.getDate(), 8); // exclusive end
  assert.equal(eventRange({ date: "x" }), null);
});

test("reminderAt computes local instants", () => {
  assert.equal(reminderAt("default", "2026-10-02", "18:00"), null);
  const morning = reminderAt("morning", "2026-10-02", null);
  assert.ok(morning);
  assert.equal(new Date(morning).getHours(), 7);
  assert.equal(new Date(morning).getDate(), 2);
  const two = reminderAt("two_hours", "2026-10-02", "09:30");
  assert.ok(two);
  assert.equal(new Date(two).getHours(), 7);
  assert.equal(new Date(two).getMinutes(), 30);
  assert.equal(reminderAt("two_hours", "2026-10-02", null), null);
  assert.equal(reminderAt("morning", "kaputt", null), null);
});

test("recurrence rules are built and described in German", () => {
  assert.equal(buildRule("none", "2026-09-30"), "");
  assert.equal(buildRule("daily", "2026-09-30"), "FREQ=DAILY");
  assert.equal(buildRule("weekly", "2026-09-30"), "FREQ=WEEKLY;BYDAY=WE");
  assert.equal(buildRule("weekly", "2026-10-04"), "FREQ=WEEKLY;BYDAY=SU");

  assert.equal(describeRule("FREQ=WEEKLY;BYDAY=WE"), "jeden Mittwoch");
  assert.equal(describeRule("FREQ=DAILY"), "täglich");
  assert.equal(describeRule("FREQ=WEEKLY;BYDAY=FR,MO"), "jeden Montag und Freitag");
  assert.equal(describeRule("FREQ=WEEKLY;BYDAY=SU,FR,MO"), "jeden Montag, Freitag und Sonntag");
  assert.equal(describeRule("FREQ=WEEKLY", "2026-09-30"), "jeden Mittwoch");
  assert.equal(describeRule("FREQ=WEEKLY"), "jede Woche");
  assert.equal(describeRule("FREQ=WEEKLY;BYDAY=WE;UNTIL=20261231"), "jeden Mittwoch bis 31. Dez");
  assert.equal(describeRule("RRULE:FREQ=DAILY;UNTIL=20261003T000000Z"), "täglich bis 3. Okt");
  assert.equal(describeRule(null), "");
  assert.equal(describeRule("FREQ=MONTHLY"), "");
});

test("helper counts read naturally", () => {
  assert.equal(helperCountText(1, 2), "1 von 2 Helfern");
  assert.equal(helperCountText(0, 1), "0 von 1 Helfer");
  assert.equal(helperCountText(1, 3, "ride_share"), "1 von 3 Plätzen");
  assert.equal(helperCountText(0, 1, "ride_share"), "0 von 1 Platz");
  assert.equal(helperStatusText(0, 2), "Noch 2 Helfer gesucht");
  assert.equal(helperStatusText(1, 2), "Noch 1 Helfer gesucht");
  assert.equal(helperStatusText(2, 2), "Alle Helfer gefunden");
  assert.equal(helperStatusText(1, 2, "ride_share"), "Noch 1 Platz frei");
  assert.equal(helperStatusText(2, 2, "ride_share"), "Alle Plätze belegt");
});

test("status badges", () => {
  assert.deepEqual(statusBadge("open"), { label: "Offen", variant: "accent" });
  assert.equal(statusBadge("assigned").label, "Vergeben");
  assert.equal(statusBadge("done").label, "Erledigt");
  assert.equal(statusBadge("cancelled").variant, "danger");
});

test("filters map to API parameters", () => {
  assert.deepEqual(filterParams("open", "2026-10-01"), { status: "open", from: "2026-10-01" });
  assert.deepEqual(filterParams("mine", "2026-10-01"), { mine: true, status: "open,assigned,done" });
  assert.deepEqual(filterParams("helping", "2026-10-01"), { assigned: true, status: "open,assigned", from: "2026-10-01" });
  assert.deepEqual(filterParams("done", "2026-10-01"), { status: "done", limit: 50 });
  assert.equal(toQuery({}), "");
  assert.equal(toQuery({ status: "open,assigned", mine: true, assigned: false, from: undefined, limit: 5 }), "?status=open%2Cassigned&mine=true&limit=5");
});

test("payload details are shown in German", () => {
  assert.deepEqual(
    payloadDetails({
      type: "show_helper",
      payload: { show_name: "Herbstturnier", classes: [{ name: "E-Dressur", time: "09:30" }, { name: "A-Springen" }], ride_along: true },
    }),
    [
      { label: "Turnier", value: "Herbstturnier" },
      { label: "Prüfungen", value: "E-Dressur (09:30), A-Springen" },
      { label: "Mitfahrgelegenheit", value: "Platz im Hänger" },
    ],
  );
  assert.deepEqual(payloadDetails({ type: "ride_share", payload: { destination: "Haltern", departure_time: "07:15", seats_free: 2 } }), [
    { label: "Ziel", value: "Haltern" },
    { label: "Abfahrt", value: "07:15" },
  ]);
  assert.deepEqual(payloadDetails({ type: "exercise", payload: { mode: "lunge", rules_note: "nur Schritt" } }), [
    { label: "Wie", value: "Longieren" },
    { label: "Regeln", value: "nur Schritt" },
  ]);
  assert.deepEqual(payloadDetails({ type: "feed_or_turnout", payload: { what: "bring_in" } }), [{ label: "Aufgabe", value: "Reinholen" }]);
  assert.deepEqual(payloadDetails({ type: "appointment_companion", payload: { with: "vet" } }), [{ label: "Termin bei", value: "Tierarzt" }]);
  assert.deepEqual(payloadDetails({ type: "other", payload: {} }), []);
});

test("formatInstant uses the device time zone", () => {
  assert.equal(formatInstant(new Date(2026, 9, 1, 18, 0).toISOString()), "Do, 1. Okt · 18:00");
  assert.equal(formatInstant(new Date(2026, 9, 1, 7, 5).toISOString()), "Do, 1. Okt · 07:05");
  assert.equal(formatInstant(null), "");
  assert.equal(formatInstant("nonsense"), "");
});

test("titles and task chips", () => {
  assert.equal(requestTitle({ type: "exercise", horse_name: "Luna", payload: {} }), "Bewegen: Luna");
  assert.equal(
    requestTitle({ type: "show_helper", horse_name: "Fanta", payload: { show_name: "Herbstturnier" } }),
    "Turniertrottel · Herbstturnier: Fanta",
  );
  assert.equal(requestTitle({ type: "other", horse_name: null, payload: {} }), "Sonstiges");
  assert.equal(taskLabel("show_helper", "hold_horse"), "Pferd halten");
  assert.equal(taskLabel("show_helper", "unbekannt"), "unbekannt");
  assert.equal(taskLabel("exercise", "hold_horse"), "hold_horse");
  assert.deepEqual(taskChips({ type: "show_helper", tasks: ["film", "fetch_number"] }), ["Filmen", "Startnummer holen"]);
  assert.deepEqual(taskChips({ type: "other", tasks: ["Heu holen"] }), ["Heu holen"]);
});

test("monthGrid lays out weeks Monday first", () => {
  const grid = monthGrid(2026, 10);
  assert.equal(grid.length, 5);
  assert.deepEqual(grid[0], [null, null, null, "2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04"]);
  assert.deepEqual(grid[4], ["2026-10-26", "2026-10-27", "2026-10-28", "2026-10-29", "2026-10-30", "2026-10-31", null]);
  assert.equal(monthGrid(2026, 2).flat().filter(Boolean).length, 28);
  assert.equal(monthGrid(2024, 2).flat().filter(Boolean).length, 29);
});

test("addMonths and monthTitle wrap over year borders", () => {
  assert.deepEqual(addMonths(2026, 12, 1), { year: 2027, month: 1 });
  assert.deepEqual(addMonths(2026, 1, -1), { year: 2025, month: 12 });
  assert.equal(monthTitle(2026, 10), "Oktober 2026");
});
