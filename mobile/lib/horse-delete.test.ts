import assert from "node:assert/strict";
import { test } from "node:test";

import { DELETE_SCOPE, deletedHorses, deletedMessage, deleteSheetText, nameMatches } from "./horse-delete.ts";

test("the typed name must equal the horse's name, ignoring case and surrounding spaces", () => {
  assert.equal(nameMatches("Luna", "Luna"), true);
  assert.equal(nameMatches("Luna", "luna"), true);
  assert.equal(nameMatches("Luna", "  LUNA  "), true);
  assert.equal(nameMatches(" Luna ", "luna"), true);
  assert.equal(nameMatches("Käthe", "käthe"), true);
  assert.equal(nameMatches("Luna Bella", "luna bella"), true);
});

test("a wrong, partial or empty name does not match", () => {
  assert.equal(nameMatches("Luna", ""), false);
  assert.equal(nameMatches("Luna", "   "), false);
  assert.equal(nameMatches("Luna", "Lun"), false);
  assert.equal(nameMatches("Luna", "Luna B"), false);
  assert.equal(nameMatches("Luna Bella", "Luna  Bella"), false);
  assert.equal(nameMatches("", ""), false);
});

test("the sheet names what is deleted and that it cannot be undone", () => {
  const t = deleteSheetText("Luna");
  assert.equal(t.title, "Luna löschen?");
  for (const what of ["Tagebuch", "Training", "Gesundheit", "Decken", "Reha", "Dokumente", "Fotos", "Aufgaben"]) {
    assert.ok(t.scope.includes(what), what);
    assert.ok((DELETE_SCOPE as readonly string[]).includes(what), what);
  }
  assert.equal(t.warning, "Das lässt sich nicht rückgängig machen.");
  assert.equal(t.fieldLabel, "Zum Bestätigen den Namen eingeben");
});

test("the result line", () => {
  assert.equal(deletedMessage("Luna"), "Luna wurde gelöscht.");
});

test("a horse is marked while its delete is under way and unmarked when it failed", () => {
  assert.equal(deletedHorses.has("h-x"), false);
  deletedHorses.mark("h-x");
  assert.equal(deletedHorses.has("h-x"), true);
  deletedHorses.unmark("h-x");
  assert.equal(deletedHorses.has("h-x"), false);
});
