import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import {
  blanketHero,
  countItems,
  dayHeading,
  groupByDay,
  isValidReminderTime,
  itemDay,
  kindLabel,
  minutesOf,
  reminderErrorMessage,
  sentLabel,
  stepReminderTime,
  timeLabel,
  type BlanketCheck,
  type ReminderItem,
} from "./reminders.ts";

const BERLIN = "Europe/Berlin";
const TODAY = "2026-09-30";

function item(over: Partial<ReminderItem>): ReminderItem {
  return {
    id: "1",
    kind: "helper",
    title: "Titel",
    body: "",
    due_at: "2026-09-30T06:00:00Z",
    all_day: false,
    sent_at: null,
    screen: "",
    computed: false,
    dismissible: true,
    ...over,
  };
}

test("kind labels", () => {
  assert.equal(kindLabel("last_person"), "Decken");
  assert.equal(kindLabel("medication"), "Medikament");
  assert.equal(kindLabel("training_plan"), "Training");
  assert.equal(kindLabel("something_new"), "Erinnerung");
});

test("dayHeading", () => {
  assert.equal(dayHeading("2026-09-30", TODAY), "Heute");
  assert.equal(dayHeading("2026-10-01", TODAY), "Morgen");
  assert.equal(dayHeading("2026-09-29", TODAY), "Gestern");
  assert.equal(dayHeading("2026-10-03", TODAY), "Samstag, 3. Okt");
  assert.equal(dayHeading("2026-12-31", "2026-12-30"), "Morgen");
  assert.equal(dayHeading("2027-01-01", "2026-12-31"), "Morgen"); // year change
  assert.equal(dayHeading("nonsense", TODAY), "nonsense");
});

test("time labels use the stable time zone", () => {
  assert.equal(timeLabel(item({ due_at: "2026-09-30T06:00:00Z" }), BERLIN), "08:00 Uhr");
  assert.equal(timeLabel(item({ due_at: "2026-01-15T07:00:00Z" }), BERLIN), "08:00 Uhr");
  assert.equal(timeLabel(item({ all_day: true }), BERLIN), "ganztägig");
  assert.equal(sentLabel(item({ sent_at: "2026-09-30T06:01:00Z" }), BERLIN), "Gesendet 08:01 Uhr");
  assert.equal(sentLabel(item({ sent_at: null }), BERLIN), "");
});

test("itemDay: sent time wins, overdue items belong to today", () => {
  assert.equal(itemDay(item({ due_at: "2026-09-30T22:30:00Z" }), BERLIN, TODAY), "2026-10-01"); // 00:30 next day
  assert.equal(itemDay(item({ due_at: "2026-09-28T00:00:00Z", all_day: true }), BERLIN, TODAY), TODAY);
  assert.equal(itemDay(item({ due_at: "2026-09-29T10:00:00Z", sent_at: "2026-09-30T06:00:00Z" }), BERLIN, TODAY), TODAY);
});

test("groupByDay sorts and sections items by local day", () => {
  const items = [
    item({ id: "d", title: "Sa", due_at: "2026-10-03T08:00:00Z" }),
    item({ id: "b", title: "Morgen 14", due_at: "2026-10-01T12:00:00Z" }),
    item({ id: "a", title: "Morgen 10", due_at: "2026-10-01T08:00:00Z" }),
    item({ id: "c", title: "Fr", due_at: "2026-10-01T22:30:00Z" }), // 00:30 on Friday Oct 2
  ];
  const sections = groupByDay(items, BERLIN, TODAY);
  assert.deepEqual(
    sections.map((s) => [s.heading, s.items.map((i) => i.title)]),
    [
      ["Morgen", ["Morgen 10", "Morgen 14"]],
      ["Freitag, 2. Okt", ["Fr"]],
      ["Samstag, 3. Okt", ["Sa"]],
    ],
  );
  assert.deepEqual(groupByDay([], BERLIN, TODAY), []);
});

test("countItems", () => {
  assert.equal(countItems([]), 0);
  assert.equal(
    countItems([
      { key: "today", label: "Heute", items: [item({}), item({ id: "2" })] },
      { key: "week", label: "Diese Woche", items: [item({ id: "3" })] },
    ]),
    3,
  );
});

function check(over: Partial<BlanketCheck>): BlanketCheck {
  return {
    day: TODAY,
    time: "20:30",
    due_at: "2026-09-30T18:30:00Z",
    done: 4,
    total: 7,
    state: "upcoming",
    screen: "/blankets",
    ...over,
  };
}

test("blanketHero", () => {
  const up = blanketHero(check({}));
  assert.equal(up.value, "4/7");
  assert.equal(up.unit, "Pferde versorgt");
  assert.match(up.description, /Erinnerung folgt/);

  assert.match(blanketHero(check({ state: "due" })).description, /3 Pferde noch offen/);
  assert.match(blanketHero(check({ state: "due", done: 6 })).description, /1 Pferd noch offen/);
  assert.match(blanketHero(check({ state: "done", done: 7 })).description, /Alle versorgt/);
  assert.equal(blanketHero(check({ state: "done", done: 1, total: 1 })).unit, "Pferd versorgt");
  const empty = blanketHero(check({ state: "empty", done: 0, total: 0 }));
  assert.equal(empty.value, "–");
  assert.match(empty.description, /Noch keine Pferde/);
});

test("reminder time validation", () => {
  assert.equal(minutesOf("20:30"), 1230);
  assert.equal(minutesOf("24:00"), null);
  assert.equal(minutesOf("8:30"), null);
  for (const ok of ["16:00", "20:30", "22:00"]) assert.equal(isValidReminderTime(ok), true, ok);
  for (const bad of ["15:45", "22:15", "", "abc", "20:60"]) assert.equal(isValidReminderTime(bad), false, bad);
});

test("stepReminderTime moves by quarter hours and stays in range", () => {
  assert.equal(stepReminderTime("20:30", 1), "20:45");
  assert.equal(stepReminderTime("20:30", -1), "20:15");
  assert.equal(stepReminderTime("20:30", 4), "21:30");
  assert.equal(stepReminderTime("21:45", 1), "22:00");
  assert.equal(stepReminderTime("22:00", 1), "22:00");
  assert.equal(stepReminderTime("16:00", -1), "16:00");
  assert.equal(stepReminderTime("16:15", -5), "16:00");
  // off the grid: snap first
  assert.equal(stepReminderTime("20:40", 1), "20:45");
  assert.equal(stepReminderTime("20:40", -1), "20:30");
  // garbage falls back to the lower bound
  assert.equal(stepReminderTime("garbage", 1), "16:15");
});

test("reminderErrorMessage", () => {
  assert.match(reminderErrorMessage(new ApiError(403, "forbidden", "admin only")), /Verwalter/);
  assert.match(reminderErrorMessage(new ApiError(400, "validation_failed", "x")), /16:00 und 22:00/);
  assert.match(reminderErrorMessage(new ApiError(0, "network", "x")), /Keine Verbindung/);
  assert.match(reminderErrorMessage(new Error("boom")), /schiefgelaufen/);
});
