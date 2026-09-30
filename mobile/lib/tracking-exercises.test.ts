import assert from "node:assert/strict";
import { test } from "node:test";

import {
  NO_FILTER,
  applyFilter,
  checklistLabel,
  disciplineLabel,
  filterOptions,
  hasFilter,
  levelLabel,
  stepCountLabel,
  tagLabel,
  toggleFilter,
} from "./tracking-exercises.ts";

const list = [
  { id: "1", title: "Übergänge", discipline: "dressage", level: "beginner", goal_tags: ["durchlaessigkeit", "losgelassenheit"] },
  { id: "2", title: "Travers", discipline: "dressage", level: "intermediate", goal_tags: ["seitengaenge", "biegung"] },
  { id: "3", title: "Cavaletti", discipline: "jumping", level: "beginner", goal_tags: ["rhythmus"] },
  { id: "4", title: "Neu", discipline: "", level: "", goal_tags: [] },
];

test("filters combine with and", () => {
  assert.equal(applyFilter(list, NO_FILTER).length, 4);
  assert.deepEqual(applyFilter(list, { ...NO_FILTER, discipline: "dressage" }).map((e) => e.id), ["1", "2"]);
  assert.deepEqual(applyFilter(list, { discipline: "dressage", level: "beginner", tag: null }).map((e) => e.id), ["1"]);
  assert.deepEqual(applyFilter(list, { ...NO_FILTER, tag: "rhythmus" }).map((e) => e.id), ["3"]);
  assert.deepEqual(applyFilter(list, { discipline: "jumping", level: "intermediate", tag: null }), []);
});

test("filter options list what exists, levels from easy to hard", () => {
  const o = filterOptions(list);
  assert.deepEqual(o.disciplines, ["dressage", "jumping"]);
  assert.deepEqual(o.levels, ["beginner", "intermediate"]);
  assert.deepEqual(o.tags, ["biegung", "durchlaessigkeit", "losgelassenheit", "rhythmus", "seitengaenge"]);
  assert.deepEqual(filterOptions([]), { disciplines: [], levels: [], tags: [] });
});

test("toggling a filter value selects and clears it", () => {
  const a = toggleFilter(NO_FILTER, "level", "beginner");
  assert.equal(a.level, "beginner");
  assert.equal(hasFilter(a), true);
  const b = toggleFilter(a, "level", "beginner");
  assert.equal(b.level, null);
  assert.equal(hasFilter(b), false);
  assert.equal(toggleFilter(a, "level", "advanced").level, "advanced");
});

test("labels are German with a readable fallback", () => {
  assert.equal(levelLabel("beginner"), "Einsteiger");
  assert.equal(levelLabel("expert"), "Expert");
  assert.equal(levelLabel(""), "");
  assert.equal(disciplineLabel("dressage"), "Dressur");
  assert.equal(disciplineLabel("some_new"), "Some new");
  assert.equal(tagLabel("seitengaenge"), "Seitengänge");
  assert.equal(tagLabel("neue_ziele"), "Neue ziele");
  assert.equal(stepCountLabel(1), "1 Schritt");
  assert.equal(stepCountLabel(4), "4 Schritte");
  assert.equal(checklistLabel(2, 5), "2 von 5 Schritten erledigt");
  assert.equal(checklistLabel(1, 1), "1 von 1 Schritt erledigt");
});
