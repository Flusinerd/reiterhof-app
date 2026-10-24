import assert from "node:assert/strict";
import { test } from "node:test";

import type { Recommendation } from "./blankets.ts";
import {
  buildHorseRows,
  diaryDay,
  diaryView,
  flattenDiary,
  heroEyebrow,
  heroSubline,
  horseWeekStrip,
  photoActionLabel,
  type DiaryLike,
  type PageDay,
  type RowBlanket,
  type RowHealthItem,
} from "./horse-page.ts";

// 2026-10-24 is a Saturday.
const today = "2026-10-24";

function day(date: string, over: Partial<PageDay> = {}): PageDay {
  return {
    date,
    status: "empty",
    is_me: false,
    user: null,
    activity: null,
    minutes: 0,
    exercise: null,
    show: null,
    ...over,
  };
}

const week: PageDay[] = [
  day("2026-10-19", { status: "done", activity: "hall", is_me: true, user: { id: "u1", name: "Jan" } }),
  day("2026-10-20", { status: "done", activity: "hack" }),
  day("2026-10-21"),
  day("2026-10-22", { status: "done", activity: "lunge" }),
  day("2026-10-23", { status: "rest" }),
  day("2026-10-24", { status: "today", activity: "hall", minutes: 45, is_me: true, user: { id: "u1", name: "Jan" }, exercise: { id: "e", title: "Schenkelweichen" } }),
  day("2026-10-25", { status: "planned", show: { name: "Turnier Aachen", tests: [] }, activity: "arena" }),
];

test("hero eyebrow and subline", () => {
  const now = new Date("2026-10-24T12:00:00Z");
  assert.equal(heroEyebrow({ sex: "mare", birth_year: 2015, box: "12" }, now), "Stute · 11 Jahre · Box 12");
  assert.equal(heroEyebrow({ sex: null, birth_year: null, box: null }, now), "");
  assert.equal(heroEyebrow({ sex: "gelding", birth_year: null, box: "16" }, now), "Wallach · Box 16");
  assert.equal(heroSubline({ owner: { name: "Jan" }, is_mine: true, i_ride: false, breed: "Hannoveraner" }), "Dein Pferd · Hannoveraner");
  assert.equal(heroSubline({ owner: { name: "Jan" }, is_mine: false, i_ride: true, breed: null }), "Du reitest · Besitzer: Jan");
  assert.equal(heroSubline({ owner: { name: "Anna" }, is_mine: false, i_ride: false, breed: null }), "Besitzer: Anna");
  assert.equal(heroSubline({ owner: null, is_mine: false, i_ride: false, breed: null }), "");
  assert.equal(photoActionLabel(false), "+ Foto hinzufügen");
  assert.equal(photoActionLabel(true), "Foto ändern");
});

test("week strip: done, planned, show, rest and empty days; today is marked", () => {
  const strip = horseWeekStrip(week, today);
  assert.deepEqual(
    strip.map((s) => s.tone),
    ["done", "done", "empty", "done", "rest", "planned", "show"],
  );
  assert.deepEqual(strip.map((s) => s.label), ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"]);
  assert.deepEqual(strip.map((s) => s.isToday), [false, false, false, false, false, true, false]);
  assert.equal(strip[5].accessibilityLabel, "Heute: geplant, Halle");
  assert.equal(strip[6].accessibilityLabel, "Sonntag, 25. Oktober: Turnier Turnier Aachen");
  assert.equal(strip[2].accessibilityLabel, "Mittwoch, 21. Oktober: nichts geplant");
});

const blanket: RowBlanket = {
  recommendation: {
    status: "blanket",
    rule_index: 0,
    blanket: { id: "b", name: "Regendecke 100 g" } as Recommendation["blanket"],
    note: "unter 5 °C und Regen",
  },
  done: false,
  state: null,
  day: today,
};

const overdue: RowHealthItem = { id: "h1", kind: "deworming", label: "Wurmkur", due_date: "2026-10-13", daily_time: null, days_until_due: -11 };
const later: RowHealthItem = { id: "h2", kind: "farrier", label: "Hufschmied", due_date: "2026-11-23", daily_time: null, days_until_due: 30 };

test("rows of an owner: training with start, blanket with change, overdue health with done", () => {
  const rows = buildHorseRows({ horseId: "H", today, days: week, blanket, health: [later, overdue], canManage: true });
  assert.deepEqual(rows.map((r) => r.key), ["training", "blanket", "health"]);
  const [t, b, h] = rows;
  assert.equal(t.eyebrow, "Training · heute mit dir");
  assert.equal(t.title, "Halle 45 Min. · Schenkelweichen");
  assert.deepEqual(t.action, { label: "Starten", kind: "start" });
  assert.equal(t.route, "/training?horse=H");
  assert.equal(b.eyebrow, "Heute Nacht");
  assert.equal(b.title, "Regendecke 100 g");
  assert.equal(b.detail, "unter 5 °C und Regen");
  assert.deepEqual(b.action, { label: "Ändern", kind: "blanket" });
  assert.equal(h.eyebrow, "Gesundheit");
  assert.equal(h.eyebrowTone, "danger");
  assert.equal(h.title, "Wurmkur überfällig");
  assert.equal(h.detail, "seit 11 Tagen");
  assert.deepEqual(h.action, { label: "Erledigt", kind: "done", itemId: "h1" });
});

test("rows of a rider: no blanket change, no done button", () => {
  const rows = buildHorseRows({ horseId: "H", today, days: week, blanket, health: [overdue], canManage: false });
  assert.equal(rows[1].action, undefined);
  assert.equal(rows[2].action, undefined);
  assert.equal(rows[0].action?.kind, "start");
});

test("a day taken by someone else has no start button", () => {
  const days = [day(today, { status: "planned", activity: "hack", minutes: 60, user: { id: "u2", name: "Lea" } })];
  const [t] = buildHorseRows({ horseId: "H", today, days, blanket: null, health: null, canManage: true });
  assert.equal(t.eyebrow, "Training · heute mit Lea");
  assert.equal(t.action, undefined);
  assert.equal(t.route, "/training/week?horse=H");
});

test("an open day can be started by a rider who may take slots, not by one who may not", () => {
  const open = (can: boolean) => [day(today, { status: "open", activity: "hall", minutes: 30, can_take: can })];
  assert.equal(buildHorseRows({ horseId: "H", today, days: open(true), blanket: null, health: null, canManage: false })[0].action?.kind, "start");
  assert.equal(buildHorseRows({ horseId: "H", today, days: open(false), blanket: null, health: null, canManage: false })[0].action, undefined);
});

test("a free day names the next planned day", () => {
  const days = [
    day(today, { status: "empty" }),
    day("2026-10-25", { status: "planned", activity: "hack", minutes: 60, user: { id: "u2", name: "Lea" } }),
  ];
  const [t] = buildHorseRows({ horseId: "H", today, days, blanket: null, health: null, canManage: false });
  assert.equal(t.eyebrow, "Training");
  assert.equal(t.title, "Heute frei");
  assert.equal(t.detail, "Morgen Ausritt mit Lea");
  assert.equal(t.action, undefined);
});

test("a rest day and a show day", () => {
  const rest = buildHorseRows({ horseId: "H", today, days: [day(today, { status: "rest", rest_reason: "after_show" })], blanket: null, health: null, canManage: false })[0];
  assert.equal(rest.title, "Ruhetag nach dem Turnier");
  const show = buildHorseRows({
    horseId: "H",
    today,
    days: [day(today, { show: { name: "Turnier Aachen", time: "09:00", tests: [{ name: "A-Dressur", time: "10:15" }] } })],
    blanket: null,
    health: null,
    canManage: true,
  })[0];
  assert.equal(show.eyebrow, "Turnier · heute");
  assert.equal(show.title, "Turnier Aachen");
  assert.equal(show.detail, "09:00 · 10:15 A-Dressur");
});

test("left out: training without access, blanket without recommendation, health without due dates", () => {
  const noRec: RowBlanket = { ...blanket, recommendation: { status: "no_weather", rule_index: null, blanket: null, note: "" } };
  const medication: RowHealthItem = { id: "m", kind: "medication", label: "Gel", due_date: "2026-10-30", daily_time: "08:00", days_until_due: 6 };
  const undated: RowHealthItem = { id: "u", kind: "vet", label: "", due_date: null, daily_time: null, days_until_due: null };
  assert.deepEqual(buildHorseRows({ horseId: "H", today, days: null, blanket: noRec, health: [medication, undated], canManage: true }), []);
});

test("blanket row: done, no blanket and manual choice", () => {
  const done = buildHorseRows({ horseId: "H", today, days: null, blanket: { ...blanket, done: true, state: { changed_by: { name: "Mia" } } }, health: null, canManage: false })[0];
  assert.equal(done.eyebrow, "Heute Nacht · erledigt");
  assert.equal(done.eyebrowTone, "success");
  assert.equal(done.detail, "Erledigt von Mia");
  const manual = buildHorseRows({
    horseId: "H",
    today,
    days: null,
    blanket: { ...blanket, recommendation: { status: "none", rule_index: null, blanket: null, note: "", manual: true }, override: { day_until: "2026-10-26" } },
    health: null,
    canManage: true,
  })[0];
  assert.equal(manual.title, "Keine Decke");
  assert.equal(manual.detail, "Von Hand gewählt · bis 26.10.2026");
});

test("health row far away has no button; soon has one", () => {
  const far = buildHorseRows({ horseId: "H", today, days: null, blanket: null, health: [later], canManage: true })[0];
  assert.equal(far.title, "Hufschmied in 4 Wochen");
  assert.equal(far.action, undefined);
  assert.equal(far.eyebrowTone, "muted");
  const soon = buildHorseRows({ horseId: "H", today, days: null, blanket: null, health: [{ ...later, days_until_due: 3 }], canManage: true })[0];
  assert.equal(soon.eyebrowTone, "accent");
  assert.equal(soon.action?.kind, "done");
});

test("diary day labels", () => {
  assert.equal(diaryDay("2026-10-24", today), "Heute");
  assert.equal(diaryDay("2026-10-23", today), "Gestern");
  assert.equal(diaryDay("2026-10-22", today), "Donnerstag");
  assert.equal(diaryDay("2026-10-18", today), "Sonntag");
  assert.equal(diaryDay("2026-10-17", today), "17. Oktober");
  assert.equal(diaryDay("2026-09-03", today), "3. September");
  assert.equal(diaryDay("2026-10-25", today), "Heute");
});

const session: DiaryLike = {
  type: "session",
  id: "s1",
  at: "2026-10-23T16:00:00Z",
  day: "2026-10-23",
  user: { id: "u2", name: "Mia" },
  session: { label: "Longe", minutes: 30, note: " Lief locker, links etwas steif. ", feel: "loose", exercise_title: "Schenkelweichen" },
  observation: null,
};

const observation: DiaryLike = {
  type: "observation",
  id: "o1",
  at: "2026-10-22T09:00:00Z",
  day: "2026-10-22",
  user: { id: "u1", name: "Jan" },
  session: null,
  observation: {
    category: "lameness",
    body_part: "hind_left",
    description: "Leichte Schwellung",
    urgency: "check",
    status: "watch",
    media_url: "/api/v1/files/x/y.jpg",
    media_count: 2,
  },
};

test("diary view of a session", () => {
  const v = diaryView(session, { today, meId: "u1" });
  assert.equal(v.when, "Gestern · Mia");
  assert.equal(v.title, "Longe 30 Min.");
  assert.equal(v.meta, "Locker · Schenkelweichen");
  assert.equal(v.quote, "Lief locker, links etwas steif.");
  assert.equal(v.dot, "success");
  assert.equal(v.photoUrl, null);
  assert.equal(v.route, "");
  assert.equal(diaryView({ ...session, session: { ...session.session!, note: null, feel: null, exercise_title: null } }, { today }).meta, undefined);
});

test("diary view of an observation", () => {
  const v = diaryView(observation, { today, meId: "u1" });
  assert.equal(v.when, "Donnerstag · du");
  assert.equal(v.title, "Lahmheit · Hinten links");
  assert.equal(v.body, "Leichte Schwellung");
  assert.equal(v.meta, "Beobachten · Bitte ansehen");
  assert.equal(v.dot, "accent");
  assert.equal(v.photoUrl, "/api/v1/files/x/y.jpg");
  assert.equal(v.photoCount, 2);
  assert.equal(v.route, "/observations/o1");
  const urgent = diaryView({ ...observation, observation: { ...observation.observation!, urgency: "urgent" } }, { today });
  assert.equal(urgent.dot, "danger");
  const done = diaryView({ ...observation, observation: { ...observation.observation!, status: "done", urgency: "info", description: "  " } }, { today });
  assert.equal(done.dot, "muted");
  assert.equal(done.meta, "Erledigt");
  assert.equal(done.body, undefined);
});

test("flattenDiary joins pages and drops doubles", () => {
  const flat = flattenDiary([{ items: [session, observation] }, { items: [observation, { ...session, id: "s2" }] }]);
  assert.deepEqual(flat.map((e) => e.id), ["s1", "o1", "s2"]);
  assert.deepEqual(flattenDiary([]), []);
});
