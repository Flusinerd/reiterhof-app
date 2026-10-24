import assert from "node:assert/strict";
import { test } from "node:test";

import type { Recommendation, TodayHorse } from "./blankets.ts";
import type { HelpRequest } from "./requests.ts";
import {
  blanketBoard,
  boardLine,
  boardMineLine,
  groupRequestsByDay,
  helpersLine,
  herePeople,
  presenceFaces,
  presenceSentence,
  requestRow,
  requestWhen,
} from "./stable-tab.ts";

// 2026-10-24 is a Saturday.
const today = "2026-10-24";

const person = (name: string, me = false) => ({ name, me });

test("presence sentence names who is at the stable", () => {
  assert.equal(presenceSentence([], true), "Niemand im Stall");
  assert.equal(presenceSentence([], false), "Wird geladen …");
  assert.equal(presenceSentence([person("Anna Meier")]), "Anna ist da");
  assert.equal(presenceSentence([person("Jan", true)]), "Du bist da");
  assert.equal(presenceSentence([person("Anna"), person("Mia")]), "Anna und Mia sind da");
  assert.equal(presenceSentence([person("Anna"), person("Mia"), person("Lea")]), "Anna, Mia und Lea sind da");
  assert.equal(presenceSentence([person("Jan", true), person("Anna")]), "Du und Anna sind da");
  assert.equal(
    presenceSentence([person("Anna"), person("Mia"), person("Lea"), person("Tom")]),
    "Anna, Mia, Lea und eine weitere Person sind da",
  );
  assert.equal(
    presenceSentence([person("Jan", true), person("Anna"), person("Mia"), person("Lea"), person("Tom")]),
    "Du, Anna, Mia und 2 weitere sind da",
  );
});

test("herePeople puts me first, only when checked in", () => {
  const me = { id: "u1", name: "Jan", avatar_color: "blue" };
  const here = [{ user_id: "u2", name: "Anna", avatar_color: "rose" }];
  assert.deepEqual(herePeople(undefined, me), []);
  assert.deepEqual(
    herePeople({ me: { open_visit: null }, here }, me).map((p) => p.name),
    ["Anna"],
  );
  const both = herePeople({ me: { open_visit: { id: "v" } }, here }, me);
  assert.deepEqual(both.map((p) => [p.name, p.me ?? false]), [["Jan", true], ["Anna", false]]);
  assert.deepEqual(herePeople({ me: { open_visit: { id: "v" } }, here: [] }, null), []);
});

test("presence faces are capped", () => {
  assert.deepEqual(presenceFaces([1, 2, 3], 4), { faces: [1, 2, 3], extra: 0 });
  assert.deepEqual(presenceFaces([1, 2, 3, 4, 5, 6], 4), { faces: [1, 2, 3, 4], extra: 2 });
});

const rec: Recommendation = { status: "blanket", rule_index: 0, blanket: null, note: "" };
const horse = (id: string, name: string, done: boolean, mine = false): TodayHorse => ({
  horse: { id, name, box: null, color_key: "green" },
  recommendation: rec,
  state: null,
  done,
  is_mine: mine,
});

test("board: done, open and my open horses; order of the server is kept", () => {
  const board = blanketBoard([horse("a", "Luna", false, true), horse("b", "Fanta", true), horse("c", "Balu", false), horse("d", "Pepe", false)], new Set(["d"]));
  assert.deepEqual(board.map((b) => [b.name, b.status]), [
    ["Luna", "mine_open"],
    ["Fanta", "done"],
    ["Balu", "open"],
    ["Pepe", "mine_open"],
  ]);
  assert.equal(board[0].label, "Luna: dein Pferd, noch offen");
  assert.equal(board[1].label, "Fanta: versorgt");
  assert.equal(board[2].label, "Balu: noch offen");
  assert.equal(boardMineLine(board), "Deine Pferde Luna und Pepe sind noch offen.");
  assert.equal(boardMineLine(board.slice(0, 2)), "Dein Pferd Luna ist noch offen.");
  assert.equal(boardMineLine(board.slice(1, 3)), null);
});

test("board line: open horses and reminder, or all done", () => {
  assert.equal(boardLine({ progress: { done: 3, total: 7 }, reminder_time: "20:30" }), "Noch 4 Pferde offen · Erinnerung um 20:30");
  assert.equal(boardLine({ progress: { done: 6, total: 7 }, reminder_time: "20:30" }), "Noch 1 Pferd offen · Erinnerung um 20:30");
  assert.equal(boardLine({ progress: { done: 7, total: 7 }, reminder_time: "20:30" }), "Alle versorgt");
  assert.equal(boardLine({ progress: { done: 0, total: 0 }, reminder_time: "20:30" }), "Noch keine Pferde");
});

function request(over: Partial<HelpRequest> = {}): HelpRequest {
  return {
    id: "r1",
    type: "feed_or_turnout",
    horse_id: "h1",
    horse_name: "Fanta",
    horse_color_key: "amber",
    created_by: "u2",
    creator_name: "Anna Meier",
    creator_color: "rose",
    date: "2026-10-25",
    date_end: null,
    time_from: "07:00",
    time_to: null,
    location: "",
    description: "Kann jemand Fanta rausstellen?",
    tasks: [],
    helpers_needed: 1,
    status: "open",
    recurring_rule: null,
    series_id: null,
    my_remind_at: null,
    payload: { what: "turnout" },
    created_at: "2026-10-23T10:00:00Z",
    helpers: [],
    helpers_count: 0,
    spots_left: 1,
    is_creator: false,
    is_helper: false,
    can_accept: true,
    ...over,
  };
}

const helper = (id: string, name: string) => ({ user_id: id, name, avatar_color: null, thanked: false, joined_at: "2026-10-23T11:00:00Z" });

test("when: relative day with time, date with time, ranges, grouped lists", () => {
  assert.equal(requestWhen(request({ date: "2026-10-24", time_from: "18:00" }), today), "heute 18:00");
  assert.equal(requestWhen(request({ date: "2026-10-25" }), today), "morgen 07:00");
  assert.equal(requestWhen(request({ date: "2026-10-31", time_from: "07:15", time_to: "09:00" }), today), "Sa, 31. Okt · 07:15–09:00");
  assert.equal(requestWhen(request({ date: "2026-10-31", time_from: null }), today), "Sa, 31. Okt");
  assert.equal(requestWhen(request({ date: "2026-10-26", date_end: "2026-10-28" }), today), "Mo, 26. Okt – Mi, 28. Okt · 07:00");
  assert.equal(requestWhen(request(), today, false), "07:00");
  assert.equal(requestWhen(request({ time_from: null }), today, false), "ganztägig");
});

test("helpers line names the helpers", () => {
  assert.equal(helpersLine(request()), "Noch niemand");
  assert.equal(helpersLine(request({ helpers: [helper("u3", "Jonas Berg")], helpers_count: 1 })), "Jonas hilft");
  assert.equal(helpersLine(request({ helpers: [helper("u3", "Jonas")], helpers_count: 1 }), "u3"), "Du hilfst");
  assert.equal(
    helpersLine(request({ helpers: [helper("u3", "Jonas"), helper("u4", "Mia")], helpers_count: 2, helpers_needed: 3, type: "show_helper" })),
    "Jonas und Mia helfen · 2 von 3 Helfern",
  );
  assert.equal(
    helpersLine(request({ helpers: [helper("u3", "Jonas")], helpers_count: 1, helpers_needed: 3, type: "ride_share" })),
    "Jonas hilft · 1 von 3 Plätzen",
  );
  assert.equal(
    helpersLine(request({ helpers: [helper("u3", "A"), helper("u4", "B"), helper("u5", "C"), helper("u6", "D")], helpers_count: 4, helpers_needed: 4 })),
    "A, B und 2 weitere helfen · 4 von 4 Helfern",
  );
  assert.equal(helpersLine(request({ status: "done" })), "Erledigt");
  assert.equal(helpersLine(request({ status: "cancelled" })), "Abgesagt");
});

test("request row: message, context, action and state", () => {
  const row = requestRow(request({ location: "Paddock" }), today);
  assert.equal(row.name, "Anna");
  assert.equal(row.colorKey, "rose");
  assert.equal(row.when, "morgen 07:00");
  assert.equal(row.text, "Kann jemand Fanta rausstellen?");
  assert.equal(row.context, "Füttern und Rausstellen: Fanta · Paddock");
  assert.equal(row.action, "Mach ich");
  assert.equal(row.helpers, "Noch niemand");
  assert.equal(row.state, "open");
  assert.equal(row.a11y, "Anna, morgen 07:00. Kann jemand Fanta rausstellen?. Noch niemand");

  const own = requestRow(request({ is_creator: true, can_accept: false, description: "", location: "Halle" }), today);
  assert.equal(own.name, "Du");
  assert.equal(own.text, "Füttern und Rausstellen: Fanta");
  assert.equal(own.context, "Halle");
  assert.equal(own.action, null);
  assert.equal(own.state, "mine");

  const ride = requestRow(request({ type: "ride_share", horse_name: null, description: "", payload: { destination: "Haltern" }, can_accept: true }), today);
  assert.equal(ride.text, "Mitfahrgelegenheit nach Haltern");

  const repeating = requestRow(request({ location: "Paddock", recurring_rule: "FREQ=DAILY" }), today);
  assert.equal(repeating.context, "Füttern und Rausstellen: Fanta · Paddock · täglich");

  assert.equal(requestRow(request({ is_helper: true, can_accept: false }), today).state, "helping");
  assert.equal(requestRow(request({ status: "done", can_accept: false }), today).state, "done");
  assert.equal(requestRow(request({ status: "cancelled", can_accept: false }), today).state, "cancelled");
});

test("requests are grouped by day with a heading", () => {
  const groups = groupRequestsByDay(
    [
      request({ id: "a", date: "2026-10-24" }),
      request({ id: "b", date: "2026-10-25" }),
      request({ id: "c", date: "2026-10-25" }),
      request({ id: "d", date: "2026-10-30" }),
    ],
    today,
  );
  assert.deepEqual(groups.map((g) => [g.heading, g.items.map((r) => r.id)]), [
    ["Heute", ["a"]],
    ["Morgen", ["b", "c"]],
    ["Freitag, 30. Okt", ["d"]],
  ]);
});
