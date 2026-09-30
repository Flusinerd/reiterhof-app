import assert from "node:assert/strict";
import { test } from "node:test";

import { planAlert } from "./web-alert-core.ts";

test("title only is a notice", () => {
  const plan = planAlert("Fertig");
  assert.equal(plan.kind, "notice");
  assert.equal(plan.text, "Fertig");
});

test("title and message are joined", () => {
  assert.equal(planAlert("Titel", "Text").text, "Titel\n\nText");
});

test("cancel plus one action is a confirm that runs the action on OK", () => {
  const calls: string[] = [];
  const plan = planAlert("Löschen?", "Wirklich?", [
    { text: "Abbrechen", style: "cancel", onPress: () => calls.push("cancel") },
    { text: "Löschen", style: "destructive", onPress: () => calls.push("delete") },
  ]);
  assert.equal(plan.kind, "confirm");
  if (plan.kind !== "confirm") return;
  plan.confirm();
  plan.cancel?.();
  assert.deepEqual(calls, ["delete", "cancel"]);
});

test("a single action button is a notice whose handler runs afterwards", () => {
  let ran = false;
  const plan = planAlert("Info", undefined, [{ text: "OK", onPress: () => (ran = true) }]);
  assert.equal(plan.kind, "notice");
  if (plan.kind !== "notice") return;
  plan.then?.();
  assert.equal(ran, true);
});

test("several action buttons prefer the destructive one", () => {
  const calls: string[] = [];
  const plan = planAlert("Was jetzt?", undefined, [
    { text: "Abbrechen", style: "cancel" },
    { text: "Pause", onPress: () => calls.push("pause") },
    { text: "Beenden", style: "destructive", onPress: () => calls.push("end") },
  ]);
  assert.equal(plan.kind, "confirm");
  if (plan.kind === "confirm") plan.confirm();
  assert.deepEqual(calls, ["end"]);
});
