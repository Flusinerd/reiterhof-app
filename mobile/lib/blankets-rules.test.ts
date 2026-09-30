import assert from "node:assert/strict";
import { test } from "node:test";

import {
  emptyDraft,
  moveItem,
  parseDraft,
  parseDrafts,
  parseTemp,
  toDraft,
  unreachableRules,
  type RuleValue,
} from "./blankets-rules.ts";

test("parseTemp accepts numbers with comma or dot and empty as open bound", () => {
  assert.equal(parseTemp("5"), 5);
  assert.equal(parseTemp(" -2,5 "), -2.5);
  assert.equal(parseTemp("0.5"), 0.5);
  assert.equal(parseTemp(""), null);
  assert.equal(parseTemp("  "), null);
  assert.equal(parseTemp("abc"), undefined);
  assert.equal(parseTemp("5 °C"), undefined);
  assert.equal(parseTemp("-"), undefined);
});

test("drafts round-trip", () => {
  const draft = toDraft({ temp_min: -2.5, temp_max: 5, rain: true, blanket_id: "b1", note: "Wunsch" });
  assert.equal(draft.temp_min, "-2,5");
  assert.equal(draft.temp_max, "5");
  assert.equal(draft.rain, "rain");
  const parsed = parseDraft(draft, 1);
  assert.deepEqual(parsed, {
    ok: true,
    value: { temp_min: -2.5, temp_max: 5, rain: true, blanket_id: "b1", note: "Wunsch" },
  });
  const open = parseDraft(toDraft({ temp_min: null, temp_max: null, rain: null, blanket_id: null, note: "" }), 1);
  assert.deepEqual(open, { ok: true, value: { temp_min: null, temp_max: null, rain: null, blanket_id: null, note: "" } });
  assert.equal(toDraft({ temp_min: null, temp_max: null, rain: false, blanket_id: null, note: "" }).rain, "dry");
  assert.notEqual(emptyDraft().key, emptyDraft().key);
});

test("draft validation mirrors the server and names the rule", () => {
  const bad = (patch: Partial<ReturnType<typeof emptyDraft>>, position = 2) =>
    parseDraft({ ...emptyDraft(), ...patch }, position);
  const notNumber = bad({ temp_min: "kalt" });
  assert.equal(notNumber.ok, false);
  assert.match(!notNumber.ok ? notNumber.error : "", /^Regel 2: .*Zahl/);
  const order = bad({ temp_min: "5", temp_max: "5" });
  assert.match(!order.ok ? order.error : "", /kleiner/);
  const range = bad({ temp_max: "99" });
  assert.match(!range.ok ? range.error : "", /zwischen -60 und 60/);
  const note = bad({ note: "x".repeat(201) });
  assert.match(!note.ok ? note.error : "", /200 Zeichen/);
  assert.equal(bad({ temp_min: "0", temp_max: "5" }).ok, true);
});

test("parseDrafts reports the first problem and the count limit", () => {
  const ok = parseDrafts([emptyDraft(), { ...emptyDraft(), temp_min: "3" }]);
  assert.equal(ok.ok && ok.value.length, 2);
  const broken = parseDrafts([emptyDraft(), { ...emptyDraft(), temp_max: "x" }, { ...emptyDraft(), temp_max: "y" }]);
  assert.match(!broken.ok ? broken.error : "", /^Regel 2:/);
  const many = parseDrafts(Array.from({ length: 31 }, () => emptyDraft()));
  assert.equal(many.ok, false);
});

test("moveItem", () => {
  assert.deepEqual(moveItem(["a", "b", "c"], 1, -1), ["b", "a", "c"]);
  assert.deepEqual(moveItem(["a", "b", "c"], 1, 1), ["a", "c", "b"]);
  assert.deepEqual(moveItem(["a", "b", "c"], 0, -1), ["a", "b", "c"]);
  assert.deepEqual(moveItem(["a", "b", "c"], 2, 1), ["a", "b", "c"]);
  const original = ["a", "b"];
  moveItem(original, 0, 1);
  assert.deepEqual(original, ["a", "b"]);
});

test("unreachable rules are found", () => {
  const rule = (temp_min: number | null, temp_max: number | null, rain: boolean | null = null): RuleValue => ({
    temp_min,
    temp_max,
    rain,
    blanket_id: null,
    note: "",
  });
  // A catch-all first hides everything after it.
  assert.deepEqual(unreachableRules([rule(null, null), rule(0, 5), rule(null, 0)]), [2, 3]);
  // Narrower rules first are fine.
  assert.deepEqual(unreachableRules([rule(null, 0), rule(0, 5), rule(5, null), rule(null, null)]), []);
  // Same range but rain-only first does not hide the general one.
  assert.deepEqual(unreachableRules([rule(5, 12, true), rule(5, 12)]), []);
  // A general rule hides the rain-only variant.
  assert.deepEqual(unreachableRules([rule(5, 12), rule(5, 12, true)]), [2]);
  assert.deepEqual(unreachableRules([]), []);
});
