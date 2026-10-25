import assert from "node:assert/strict";
import { test } from "node:test";

import {
  classesAbove,
  goalLetter,
  goalOptions,
  goalSummary,
  isGoalLevel,
  levelUp,
  levelUpConfirm,
  levelUpLabel,
  nextLections,
  progressText,
  selectedGoal,
  type LevelProgress,
} from "./training-goal.ts";

const progress = (over: Partial<LevelProgress> = {}): LevelProgress => ({
  level: "beginner",
  goal_level: "intermediate",
  goal_until: null,
  mastered: 7,
  total: 12,
  next: [{ id: "e1", title: "Schulterherein" }],
  ready: false,
  text: "Auf dem Weg zu L: 7 von 12 Lektionen gefestigt.",
  ...over,
});

test("a goal names a class above the current one", () => {
  assert.deepEqual(classesAbove("beginner"), ["intermediate", "advanced"]);
  assert.deepEqual(classesAbove("intermediate"), ["advanced"]);
  assert.deepEqual(classesAbove("advanced"), []);
  assert.deepEqual(classesAbove(""), ["intermediate", "advanced"], "an unknown class counts as beginner");
  assert.deepEqual(classesAbove(undefined), ["intermediate", "advanced"]);
  assert.equal(isGoalLevel("intermediate"), true);
  assert.equal(isGoalLevel("advanced"), true);
  assert.equal(isGoalLevel("beginner"), false);
  assert.equal(isGoalLevel(null), false);
  assert.equal(goalLetter("intermediate"), "L");
  assert.equal(goalLetter("advanced"), "M");
  assert.equal(goalLetter("beginner"), null);
  assert.equal(goalLetter(null), null);
});

test("the picker offers the next class and every higher one, with the existing level labels", () => {
  assert.deepEqual(goalOptions("beginner"), [
    { value: null, label: "Nächste Klasse" },
    { value: "advanced", label: "Erfahren" },
  ]);
  assert.deepEqual(goalOptions("intermediate"), [{ value: null, label: "Nächste Klasse" }]);
  assert.deepEqual(goalOptions("advanced"), [], "nothing above advanced");
  assert.equal(goalOptions(undefined).length, 2);
});

test("the chosen option: an explicit goal above the next class, else the next class", () => {
  assert.equal(selectedGoal(progress({ goal_level: "intermediate" })), null, "the next class is the default");
  assert.equal(selectedGoal(progress({ goal_level: "advanced" })), "advanced");
  assert.equal(selectedGoal(progress({ goal_level: null })), null);
  assert.equal(selectedGoal(progress({ level: "intermediate", goal_level: "advanced" })), null);
  assert.equal(selectedGoal(undefined), null);
  assert.equal(goalSummary(progress()), "Nächste Klasse");
  assert.equal(goalSummary(progress({ goal_level: "advanced" })), "Erfahren");
  assert.equal(goalSummary(progress({ goal_until: "2027-03-15" })), "Nächste Klasse bis 15.03.");
  assert.equal(goalSummary(progress({ goal_level: "advanced", goal_until: "2027-03-15" })), "Erfahren bis 15.03.");
  assert.equal(goalSummary(progress({ goal_until: "kaputt" })), "Nächste Klasse");
  assert.equal(goalSummary(null), "Nächste Klasse");
});

test("the progress text is the API's, with a fallback in the same forms", () => {
  assert.equal(progressText(progress()), "Auf dem Weg zu L: 7 von 12 Lektionen gefestigt.");
  assert.equal(progressText(progress({ text: "  Eigener Text.  " })), "Eigener Text.");
  assert.equal(progressText(progress({ text: "" })), "Auf dem Weg zu L: 7 von 12 Lektionen gefestigt.");
  assert.equal(progressText(progress({ text: undefined, goal_level: "advanced" })), "Auf dem Weg zu M: 7 von 12 Lektionen gefestigt.");
  assert.equal(progressText(progress({ text: undefined, goal_until: "2027-03-15", mastered: 7 })), "Ziel L bis 15.03.: noch 5 Lektionen.");
  assert.equal(progressText(progress({ text: undefined, goal_until: "2027-03-15", mastered: 11 })), "Ziel L bis 15.03.: noch 1 Lektion.");
  assert.equal(
    progressText(progress({ text: undefined, ready: true, mastered: 10 }), "Luna"),
    "Luna ist bereit für L: 10 von 12 Lektionen gefestigt.",
  );
  assert.equal(progressText(progress({ text: undefined, ready: true }), " "), "Das Pferd ist bereit für L: 7 von 12 Lektionen gefestigt.");
  assert.equal(progressText(progress({ text: undefined, total: 0 })), null, "no lections, nothing to say");
  assert.equal(progressText(progress({ text: undefined, goal_level: null })), null);
  assert.equal(progressText(undefined), null);
  assert.equal(progressText(null), null);
});

test("at most three next lections are shown", () => {
  const next = [1, 2, 3, 4].map((n) => ({ id: `e${n}`, title: `Lektion ${n}` }));
  assert.deepEqual(nextLections({ next }).map((n) => n.id), ["e1", "e2", "e3"]);
  assert.deepEqual(nextLections({}), []);
  assert.deepEqual(nextLections(undefined), []);
});

test("the level_up warning offers the class it names, else the goal's", () => {
  const text = "Luna hat 10 von 12 L-Lektionen gefestigt – Einstufung auf L anpassen?";
  assert.deepEqual(levelUp([{ key: "level_up", text }], progress()), { text, letter: "L" });
  assert.deepEqual(levelUp([{ key: "level_up", text: "Luna hat 10 von 12 M-Lektionen gefestigt – Einstufung auf M anpassen?" }], progress()), {
    text: "Luna hat 10 von 12 M-Lektionen gefestigt – Einstufung auf M anpassen?",
    letter: "M",
  });
  assert.deepEqual(levelUp([{ key: "level_up", text: "Bereit für die nächste Klasse." }], progress({ goal_level: "advanced" })), {
    text: "Bereit für die nächste Klasse.",
    letter: "M",
  });
  assert.deepEqual(levelUp([{ key: "level_up", text: "Bereit." }], undefined), { text: "Bereit.", letter: null });
  assert.equal(levelUp([{ key: "versammlung", text: "…" }], progress()), null);
  assert.equal(levelUp([], progress()), null);
  assert.equal(levelUp(undefined, progress()), null);
  assert.equal(levelUpLabel("L"), "Auf L einstufen");
  assert.deepEqual(levelUpConfirm("Luna", "L"), {
    title: "Auf L einstufen?",
    description: "Luna bekommt im Profil das Niveau L. Als Ziel gilt danach wieder die nächste Klasse.",
  });
  assert.match(levelUpConfirm(" ", "L").description, /^Das Pferd bekommt/);
});
