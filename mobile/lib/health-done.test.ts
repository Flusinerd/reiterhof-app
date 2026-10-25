import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import {
  canMarkDone,
  doneLabel,
  doneState,
  endCourseSheet,
  freezeOrder,
  healthErrorMessage,
  inOrder,
  isCourse,
  type DoneItem,
} from "./health-done.ts";

const today = "2026-10-25";

function item(over: Partial<DoneItem> = {}): DoneItem {
  return {
    id: "i1",
    kind: "deworming",
    label: "Wurmkur",
    due_date: "2026-10-01",
    interval_count: 3,
    daily_time: null,
    last_done_on: null,
    last_done_by: null,
    ...over,
  };
}

test("a medication (kind or daily time) is a course, everything else an appointment", () => {
  assert.equal(isCourse(item()), false);
  assert.equal(isCourse(item({ kind: "medication" })), true);
  assert.equal(isCourse(item({ kind: "vet", daily_time: "08:00" })), true);
});

test("the button: Erledigt for appointments, Gabe beenden for a course; a course needs an end date", () => {
  assert.equal(doneLabel(item()), "Erledigt");
  assert.equal(doneLabel(item({ kind: "medication", daily_time: "08:00" })), "Gabe beenden");
  assert.equal(canMarkDone(item()), true);
  assert.equal(canMarkDone(item({ due_date: null, interval_count: 2 })), true); // the interval starts from today
  assert.equal(canMarkDone(item({ due_date: null, interval_count: null })), false);
  assert.equal(canMarkDone(item({ kind: "medication", daily_time: "08:00", due_date: "2026-10-30", interval_count: null })), true);
  // Without an end date a course never ends: no button (also not with an interval it cannot have).
  assert.equal(canMarkDone(item({ kind: "medication", daily_time: "08:00", due_date: null, interval_count: null })), false);
});

test("the sheet before a course is ended names the medication and what happens", () => {
  assert.deepEqual(endCourseSheet({ label: "Equioxx" }), {
    title: "Gabe beenden?",
    description: "Equioxx endet heute. Du bekommst dafür keine Erinnerungen mehr.",
    confirm: "Gabe beenden",
  });
  assert.match(endCourseSheet({ label: "" }).description, /^Das Medikament endet heute/);
});

test("done today: Heute erledigt with the next appointment", () => {
  const done = item({ due_date: "2027-01-23", last_done_on: today, last_done_by: { id: "me", name: "Jan" } });
  assert.deepEqual(doneState(done, today, "me"), { title: "Heute erledigt", detail: "Nächster Termin: 23.01.2027" });
  // Somebody else did it: named.
  assert.equal(doneState(done, today, "other")?.title, "Heute erledigt von Jan");
  // A one-off appointment is finished.
  assert.equal(doneState(item({ due_date: null, interval_count: null, last_done_on: today }), today)?.detail, "Abgeschlossen");
});

test("done on another day or never: no success state", () => {
  assert.equal(doneState(item({ last_done_on: "2026-10-24" }), today), null);
  assert.equal(doneState(item({ last_done_on: null }), today), null);
  assert.equal(doneState(item(), today), null);
});

test("a course that was ended today", () => {
  const course = item({ kind: "medication", daily_time: "08:00", due_date: today, interval_count: null, last_done_on: today });
  assert.deepEqual(doneState(course, today, "me"), { title: "Gabe beendet", detail: "Endet heute. Keine Erinnerungen mehr." });
  assert.equal(doneState({ ...course, last_done_by: { id: "x", name: "Anna" } }, today, "me")?.title, "Gabe beendet von Anna");
  // The end date was earlier already (the course was over): it says so.
  assert.equal(doneState({ ...course, due_date: "2026-10-20" }, today)?.detail, "Endet am 20.10.2026. Keine Erinnerungen mehr.");
});

test("error texts of erledigt and its undo", () => {
  assert.match(healthErrorMessage(new ApiError(409, "due_date_changed", "x")), /inzwischen geändert/);
  assert.match(healthErrorMessage(new ApiError(409, "undo_expired", "x")), /nicht mehr rückgängig/);
  assert.equal(healthErrorMessage(new ApiError(0, "network", "x")), "Keine Verbindung. Versuch es gleich noch mal.");
  assert.match(healthErrorMessage(new Error("boom")), /schiefgelaufen/);
});

test("freezeOrder keeps every row where it was and puts new ones after their predecessor", () => {
  // First visit: the order of the server.
  assert.deepEqual(freezeOrder([], ["a", "b", "c"]), ["a", "b", "c"]);
  // The server re-sorted after "a" was done (it moved to the end): the frozen order does not follow.
  assert.deepEqual(freezeOrder(["a", "b", "c"], ["b", "c", "a"]), ["a", "b", "c"]);
  // A new appointment lands after the one before it in the server's order.
  assert.deepEqual(freezeOrder(["a", "b", "c"], ["b", "n", "c", "a"]), ["a", "b", "n", "c"]);
  // A new first row goes to the top; several new ones keep their order.
  assert.deepEqual(freezeOrder(["a", "b"], ["n1", "a", "b", "n2", "n3"]), ["n1", "a", "b", "n2", "n3"]);
  // Deleted rows drop out.
  assert.deepEqual(freezeOrder(["a", "b", "c"], ["a", "c"]), ["a", "c"]);
});

test("inOrder sorts by the frozen order, unknown ids last", () => {
  const rows = [{ id: "b" }, { id: "c" }, { id: "a" }, { id: "z" }];
  assert.deepEqual(inOrder(rows, ["a", "b", "c"]).map((r) => r.id), ["a", "b", "c", "z"]);
  assert.deepEqual(inOrder(rows, []).map((r) => r.id), ["b", "c", "a", "z"]);
});
