import assert from "node:assert/strict";
import { test } from "node:test";

import { ACTIVITIES } from "./training.ts";
import {
  FEELING_OPTIONS,
  buildSessionBody,
  durationCaption,
  exercisePractised,
  feelingLabel,
  finishEyebrow,
  focusFromFeeling,
  notePlaceholder,
  otherPerson,
  parseMinutes,
  placeLabel,
  savedTitle,
  stepsCaption,
  visibleLabel,
} from "./session-finish.ts";

test("feelings match the backend values and carry German labels", () => {
  assert.deepEqual(
    FEELING_OPTIONS.map((f) => [f.value, f.label]),
    [
      ["hard", "Schwer"],
      ["good", "Gut"],
      ["easy", "Sehr locker"],
    ],
  );
  assert.equal(feelingLabel("easy"), "Sehr locker");
  assert.equal(feelingLabel(null), "");
  assert.equal(feelingLabel("great"), "");
});

test("the feeling rates the exercise like the old focus chips", () => {
  assert.equal(focusFromFeeling("hard"), 1);
  assert.equal(focusFromFeeling("good"), 2);
  assert.equal(focusFromFeeling("easy"), 3);
  assert.equal(focusFromFeeling(null), undefined);
});

test("place and duration line", () => {
  for (const a of ACTIVITIES) assert.ok(placeLabel(a).length > 3, a);
  assert.equal(durationCaption("hall", true), "Minuten in der Halle, aus dem Tracking");
  assert.equal(durationCaption("hack", false), "Minuten im Gelände");
  assert.equal(finishEyebrow("Luna"), "Luna · gerade eben");
});

test("typed minutes are whole numbers from 1 to 600", () => {
  assert.equal(parseMinutes("47"), 47);
  assert.equal(parseMinutes(" 5 "), 5);
  assert.equal(parseMinutes("600"), 600);
  assert.equal(parseMinutes("601"), null);
  assert.equal(parseMinutes("0"), null);
  assert.equal(parseMinutes(""), null);
  assert.equal(parseMinutes("4,5"), null);
  assert.equal(parseMinutes("abc"), null);
});

test("the note speaks to the other person on the horse", () => {
  const horse = {
    owner: { id: "jan", name: "Jan Krüger" },
    riders: [
      { user_id: "mia", name: "Mia Berger" },
      { user_id: "tom", name: "Tom" },
    ],
  };
  assert.equal(otherPerson(horse, "jan"), "Mia");
  assert.equal(otherPerson(horse, "mia"), "Jan");
  assert.equal(otherPerson({ owner: horse.owner, riders: [] }, "jan"), null);
  assert.equal(otherPerson({ owner: horse.owner, riders: [{ user_id: "jan", name: "Jan" }] }, "jan"), null);
  assert.equal(otherPerson({ owner: null, riders: horse.riders }, "mia"), "Tom");
  assert.equal(otherPerson(undefined, "jan"), null);
  assert.equal(otherPerson(horse, undefined), null);
  assert.equal(notePlaceholder("Mia"), "Notiz für Mia …");
  assert.equal(notePlaceholder(null), "Notiz …");
  assert.equal(visibleLabel("Mia"), "Mia sieht die Einheit");
});

test("the exercise starts out ticked from the tracker, or from the plan without tracker data", () => {
  assert.equal(exercisePractised([0, 2]), true);
  assert.equal(exercisePractised([]), false);
  assert.equal(exercisePractised(undefined), true);
  assert.equal(stepsCaption([0, 2], 4), "2 von 4 Schritten erledigt");
  assert.equal(stepsCaption([0, 1, 2, 3, 4], 4), "4 von 4 Schritten erledigt");
  assert.equal(stepsCaption([], 4), "0 von 4 Schritten erledigt");
  assert.equal(stepsCaption(undefined, 4), null);
  assert.equal(stepsCaption([0], 0), null);
});

test("body of a tracked session with exercise, feeling and note", () => {
  const body = buildSessionBody({
    activity: "hall",
    minutes: 47,
    startedAt: "2026-10-24T16:00:00.000Z",
    gaitShares: { walk: 0.4, trot: 0.6 },
    reinChanges: [{ rein: "left", minutes: 20 }],
    exercise: { id: "e1", practised: true },
    feeling: "good",
    note: "  Schenkelweichen klappt  ",
    visibleToRider: true,
  });
  assert.deepEqual(body, {
    activity: "hall",
    minutes: 47,
    started_at: "2026-10-24T16:00:00.000Z",
    gait_shares: { walk: 0.4, trot: 0.6 },
    rein_changes: [{ rein: "left", minutes: 20 }],
    exercise_id: "e1",
    feeling: "good",
    focus_rating: 2,
    note: "Schenkelweichen klappt",
    visible_to_rider: true,
  });
});

test("an exercise that was not practised is not recorded, and a bare body stays bare", () => {
  const none = buildSessionBody({
    activity: "arena",
    minutes: 30,
    exercise: { id: "e1", practised: false },
    feeling: "easy",
    note: "",
  });
  assert.deepEqual(none, { activity: "arena", minutes: 30, feeling: "easy" });

  const bare = buildSessionBody({ activity: "hack", minutes: 60, feeling: null, note: "   " });
  assert.deepEqual(bare, { activity: "hack", minutes: 60 });

  const practised = buildSessionBody({
    activity: "lunge",
    minutes: 20,
    exercise: { id: "e2", practised: true },
    feeling: "easy",
    note: "",
  });
  assert.equal(practised.exercise_id, "e2");
  assert.equal(practised.focus_rating, 3);
});

test("saved title", () => {
  assert.equal(savedTitle("hall", 47), "Halle, 47 Min.");
});
