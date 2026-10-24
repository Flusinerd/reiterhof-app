import assert from "node:assert/strict";
import { test } from "node:test";

import type { TodayHorse } from "./blankets.ts";
import type { ReminderItem } from "./reminders.ts";
import { buildToday, nextAction, nextEyebrow, reminderText, todaySummary, type PlanDay, type TodayInput } from "./today.ts";

const tz = "Europe/Berlin";

function day(over: Partial<PlanDay>): PlanDay {
  return {
    date: "2026-10-24",
    status: "today",
    is_me: true,
    user: { id: "u1", name: "Jan" },
    activity: "hall",
    minutes: 45,
    exercise: { id: "e1", title: "Schenkelweichen" },
    show: null,
    ...over,
  };
}

function horse(over: Partial<TodayHorse>): TodayHorse {
  return {
    horse: { id: "h1", name: "Luna", box: "12", color_key: "green" },
    recommendation: {
      status: "blanket",
      rule_index: 0,
      blanket: { id: "b1", horse_id: "h1", name: "Regendecke 100 g" } as TodayHorse["recommendation"]["blanket"],
      note: "",
    },
    state: null,
    done: false,
    is_mine: true,
    ...over,
  };
}

function reminder(over: Partial<ReminderItem>): ReminderItem {
  return {
    id: "r1",
    kind: "health_due",
    title: "Fällig: Luna",
    body: "Wurmkur",
    due_at: "2026-10-15T00:00:00+02:00",
    all_day: true,
    sent_at: null,
    screen: "/horses/h1/health",
    computed: true,
    dismissible: false,
    ...over,
  };
}

function input(over: Partial<TodayInput>): TodayInput {
  return { today: "2026-10-24", nowClock: "16:20", timeZone: tz, plans: [], blankets: null, reminders: [], ...over };
}

test("own training of today is the next task", () => {
  const view = buildToday(
    input({
      plans: [{ horseId: "h1", horseName: "Luna", days: [day({}), day({ date: "2026-10-25", status: "planned" })] }],
      blankets: { reminderTime: "20:30", horses: [horse({})] },
    }),
  );
  assert.equal(view.next?.kind, "training");
  assert.equal(view.next?.title, "Luna · Halle");
  assert.equal(view.next?.detail, "45 Min. · Schenkelweichen · mit dir");
  assert.equal(nextAction(view.next!), "Training starten");
  assert.equal(nextEyebrow(view.next!), "Heute");
  assert.deepEqual(
    view.today.map((e) => [e.kind, e.time, e.title]),
    [["blanket", "20:30", "Decke für Luna"]],
  );
  assert.equal(view.tomorrow.length, 1);
});

test("training of the rider is information, not my task; done blankets are not next", () => {
  const view = buildToday(
    input({
      plans: [{ horseId: "h1", horseName: "Luna", days: [day({ is_me: false, user: { id: "u2", name: "Mia" } })] }],
      blankets: { reminderTime: "20:30", horses: [horse({ done: true, state: { changed_by: { id: "u2", name: "Mia" } } as TodayHorse["state"] })] },
    }),
  );
  assert.equal(view.next, null);
  assert.equal(view.today.find((e) => e.kind === "training")?.detail, "45 Min. · Schenkelweichen · Mia");
  assert.equal(view.today.find((e) => e.kind === "blanket")?.detail, "Regendecke 100 g · erledigt von Mia");
  assert.equal(todaySummary(view), "Für heute ist alles erledigt.");
});

test("without training the blanket at the reminder time is next; overdue items are danger", () => {
  const view = buildToday(
    input({
      blankets: { reminderTime: "20:30", horses: [horse({})] },
      reminders: [reminder({}), reminder({ id: "r2", kind: "training_plan", due_at: "2026-10-24T19:00:00+02:00", all_day: false })],
    }),
  );
  assert.equal(view.next?.kind, "blanket");
  assert.equal(nextEyebrow(view.next!), "Als Nächstes · 20:30");
  assert.equal(nextAction(view.next!), "Runde starten");
  assert.equal(view.next?.screen, "/blankets/round");
  assert.equal(view.today.length, 1, "training_plan reminders are left out");
  assert.equal(view.today[0].tone, "danger");
  assert.equal(view.today[0].time, null);
  assert.equal(todaySummary(view), "2 Dinge stehen heute an.");
});

test("after the blanket time the overdue item comes next", () => {
  const view = buildToday(
    input({ nowClock: "21:00", blankets: { reminderTime: "20:30", horses: [horse({})] }, reminders: [reminder({})] }),
  );
  assert.equal(view.next?.tone, "danger");
  assert.equal(nextEyebrow(view.next!), "Überfällig");
});

test("shows with time and classes, timed entries first", () => {
  const view = buildToday(
    input({
      plans: [
        {
          horseId: "h1",
          horseName: "Luna",
          days: [
            day({
              date: "2026-10-25",
              status: "planned",
              show: { name: "Turnier Lünen", time: "08:30", tests: [{ name: "A-Dressur", time: "09:30" }, { name: "L-Dressur" }] },
            }),
          ],
        },
      ],
      reminders: [reminder({ id: "m", kind: "medication", due_at: "2026-10-25T07:00:00+01:00", all_day: false, title: "Medikament: Luna", body: "Pergolid 1 mg, 07:00 Uhr." })],
    }),
  );
  assert.deepEqual(
    view.tomorrow.map((e) => [e.time, e.title, e.detail]),
    [
      ["07:00", "Pergolid 1 mg · Luna", "Medikament"],
      ["08:30", "Turnier Lünen · Luna", "09:30 A-Dressur · L-Dressur"],
    ],
  );
});

test("rest days, empty days and reminders after tomorrow are left out", () => {
  const view = buildToday(
    input({
      plans: [{ horseId: "h1", horseName: "Luna", days: [day({ status: "rest", activity: null }), day({ date: "2026-10-25", status: "empty", activity: null })] }],
      reminders: [reminder({ due_at: "2026-10-28T00:00:00+01:00" })],
    }),
  );
  assert.deepEqual(view, { next: null, today: [], tomorrow: [] });
});

test("reminder texts lead with the thing", () => {
  assert.deepEqual(reminderText("health_due", "Fällig: Luna", "Wurmkur: seit 10 Tagen überfällig (15.10.2026)."), {
    title: "Wurmkur · Luna",
    detail: "seit 10 Tagen überfällig (15.10.2026)",
  });
  assert.deepEqual(reminderText("medication", "Medikament: Fanta", "Pergolid 1 mg, 08:00 Uhr."), {
    title: "Pergolid 1 mg · Fanta",
    detail: "Medikament",
  });
  assert.deepEqual(reminderText("helper", "Anfrage morgen", "Fanta rausstellen."), {
    title: "Anfrage morgen",
    detail: "Fanta rausstellen",
  });
  const view = buildToday(input({ reminders: [reminder({ body: "Wurmkur: seit 9 Tagen überfällig (15.10.2026)." })] }));
  assert.equal(nextAction(view.next!), "Zum Termin");
});
