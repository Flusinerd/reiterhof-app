import assert from "node:assert/strict";
import { test } from "node:test";

import type { BlanketState, Recommendation, TodayHorse, Weather } from "./blankets.ts";
import {
  boxLine,
  firstOpenIndex,
  hasRecommendation,
  nextIndex,
  peekText,
  previousIndex,
  roundActions,
  roundDetails,
  roundEyebrow,
  roundOrder,
  roundSegments,
  roundSummary,
  roundTitle,
  weatherLine,
} from "./blanket-round.ts";

const blanket = (name: string, location: string | null = null) =>
  ({ id: `b-${name}`, horse_id: "h", name, fill_g: 100, color: null, location, photo_path: null, photo_url: null }) as NonNullable<Recommendation["blanket"]>;

const rec = (over: Partial<Recommendation>): Recommendation => ({
  status: "blanket",
  rule_index: 0,
  blanket: blanket("Regendecke 100 g"),
  note: "",
  ...over,
});

function horse(name: string, box: string | null, over: Partial<TodayHorse> = {}): TodayHorse {
  return {
    horse: { id: name.toLowerCase(), name, box, color_key: null },
    recommendation: rec({}),
    state: null,
    done: false,
    is_mine: false,
    ...over,
  };
}

const state = (action: BlanketState["action"], over: Partial<BlanketState> = {}): BlanketState => ({
  id: "s1",
  horse_id: "h",
  day: "2026-10-24",
  action,
  covered_with: null,
  covered_with_name: null,
  changed_at: "2026-10-24T17:04:00Z",
  changed_by: { id: "u1", name: "Jan" },
  automatic: false,
  ...over,
});

const weather = (over: Partial<Weather> = {}): Weather => ({
  night_min_c: 4,
  will_rain: true,
  rain_probability: 80,
  rain_mm: 4.8,
  wind_kmh: 20,
  fetched_at: "",
  window_start: "",
  window_end: "",
  temp_max_c: 9,
  rain_peak_mm: 1,
  rain_hours: 4,
  rain_from: null,
  rain_until: null,
  timeline: [],
  ...over,
});

const horses = [
  horse("Merlin", "11"),
  horse("Luna", "12"),
  horse("Pepe", "10"),
  horse("Nala", "9"),
  horse("Zoe", null),
  horse("Fanta", "14"),
];

test("own horses come first, then the rest, each by box", () => {
  const order = roundOrder(horses, new Set(["luna", "fanta"]));
  assert.deepEqual(
    order.map((h) => h.horse.name),
    ["Luna", "Fanta", "Nala", "Pepe", "Merlin", "Zoe"],
  );
  // numbers compare as numbers ("9" before "10"), horses without a box come last
  assert.deepEqual(
    roundOrder(horses, new Set()).map((h) => h.horse.name),
    ["Nala", "Pepe", "Merlin", "Luna", "Fanta", "Zoe"],
  );
});

test("the order does not change its input", () => {
  const copy = [...horses];
  roundOrder(horses, new Set(["luna"]));
  assert.deepEqual(horses, copy);
});

test("next horse skips the done ones and comes back to the skipped ones", () => {
  const order = [horse("A", "1"), horse("B", "2", { done: true }), horse("C", "3"), horse("D", "4", { done: true })];
  assert.equal(firstOpenIndex(order), 0);
  assert.equal(nextIndex(order, 0), 2); // B is done
  assert.equal(nextIndex(order, 2), 0); // end of the list: A was skipped with "Später" and comes around
  assert.equal(nextIndex(order, 3), 0);
  assert.equal(firstOpenIndex([horse("A", "1", { done: true })]), null);
  assert.equal(nextIndex([horse("A", "1")], 0), null); // the only open horse: nothing else to show
  assert.equal(nextIndex([horse("A", "1", { done: true }), horse("B", "2", { done: true })], 0), null);
  assert.equal(nextIndex([], 0), null);
});

test("going back is allowed from the second horse on", () => {
  assert.equal(previousIndex(0), null);
  assert.equal(previousIndex(3), 2);
});

test("progress segments: done, current, open", () => {
  const order = [horse("A", "1", { done: true }), horse("B", "2", { done: true }), horse("C", "3"), horse("D", "4")];
  assert.deepEqual(roundSegments(order, 2), ["done", "done", "current", "open"]);
  // going back to a done horse shows it as the current one
  assert.deepEqual(roundSegments(order, 0), ["current", "done", "open", "open"]);
  assert.deepEqual(roundSegments(order, null), ["done", "done", "open", "open"]);
});

test("eyebrow and box line", () => {
  assert.equal(roundEyebrow(7), "Abendrunde · 7 Pferde");
  assert.equal(roundEyebrow(1), "Abendrunde · 1 Pferd");
  assert.equal(boxLine(horse("Luna", "12"), true), "Box 12 · dein Pferd");
  assert.equal(boxLine(horse("Merlin", "11"), false), "Box 11");
  assert.equal(boxLine(horse("Zoe", null), true), "dein Pferd");
  assert.equal(boxLine(horse("Zoe", " "), false), "");
});

test("title: the blanket, none, or a hint to check the rule", () => {
  assert.equal(roundTitle(rec({})), "Regendecke 100 g");
  assert.equal(roundTitle(rec({ status: "none", blanket: null })), "Keine Decke");
  assert.equal(roundTitle(rec({ status: "no_weather", blanket: null })), "Keine Empfehlung — Regel prüfen");
  assert.equal(roundTitle(rec({ status: "no_rule", blanket: null })), "Keine Empfehlung — Regel prüfen");
  assert.equal(hasRecommendation(rec({ status: "none" })), true);
  assert.equal(hasRecommendation(rec({ status: "no_weather" })), false);
});

test("weather line", () => {
  assert.equal(weatherLine(weather()), "4 bis 9 °C, 4,8 mm Regen");
  assert.equal(weatherLine(weather({ wind_kmh: 62, rain_mm: 0, temp_max_c: 4 })), "4 °C, trocken, Wind bis 62 km/h");
  assert.equal(weatherLine(null), "Noch keine Vorhersage");
});

test("details: manual choice, owner's wish and place", () => {
  assert.deepEqual(roundDetails(rec({}), weather()), ["4 bis 9 °C, 4,8 mm Regen"]);
  assert.deepEqual(roundDetails(rec({ manual: true, note: "frisch geschoren" }), null), [
    "Noch keine Vorhersage",
    "von Hand gewählt: frisch geschoren",
  ]);
  assert.deepEqual(roundDetails(rec({ manual: true }), null)[1], "von Hand gewählt");
  assert.deepEqual(roundDetails(rec({ note: "Nur mit Halsteil" }), null)[1], "Besitzerwunsch: Nur mit Halsteil");
  assert.deepEqual(roundDetails(rec({ blanket: blanket("Fleece", "Sattelkammer") }), null)[1], "Ort: Sattelkammer");
});

test("buttons follow the recommendation", () => {
  assert.deepEqual(roundActions(rec({})), {
    primary: { action: "covered", label: "Eingedeckt" },
    extra: [{ action: "uncovered", label: "Abgedeckt" }],
  });
  assert.deepEqual(roundActions(rec({ status: "none", blanket: null })), {
    primary: { action: "checked", label: "Keine Decke — passt" },
    extra: [],
  });
  assert.deepEqual(roundActions(rec({ status: "no_weather", blanket: null })), {
    primary: { action: "covered", label: "Eingedeckt" },
    extra: [
      { action: "uncovered", label: "Abgedeckt" },
      { action: "checked", label: "Geprüft" },
    ],
  });
});

test("peek row", () => {
  assert.deepEqual(peekText(horse("Merlin", "11", { recommendation: rec({ blanket: blanket("Fleecedecke") }) })), {
    title: "Merlin als Nächstes",
    detail: "Fleecedecke · Box 11",
  });
  assert.equal(peekText(horse("Zoe", null)).detail, "Regendecke 100 g");
  assert.equal(peekText(horse("Pepe", "10", { recommendation: rec({ status: "no_weather", blanket: null }) })).detail, "Keine Empfehlung · Box 10");
});

test("summary lists what was set", () => {
  const order = [
    horse("Luna", "12", { done: true, state: state("covered", { covered_with_name: "Regendecke 100 g" }) }),
    horse("Merlin", "11", { done: true, state: state("uncovered", { changed_by: null, automatic: true }) }),
    horse("Pepe", "10"),
  ];
  const s = roundSummary(order, "UTC");
  assert.equal(s.title, "2 von 3 versorgt");
  assert.equal(s.subtitle, "Ein Pferd ist noch offen.");
  assert.equal(s.openCount, 1);
  assert.deepEqual(s.rows[0], { horseId: "luna", name: "Luna", text: "Eingedeckt mit Regendecke 100 g", meta: "Jan, 17:04", done: true });
  assert.equal(s.rows[1]!.text, "Abgedeckt (Hof, automatisch)");
  assert.deepEqual(s.rows[2], { horseId: "pepe", name: "Pepe", text: "Noch offen", meta: "", done: false });

  const all = roundSummary(order.slice(0, 2), "UTC");
  assert.equal(all.title, "2 von 2 versorgt");
  assert.equal(all.subtitle, "Alle Pferde sind versorgt.");
  assert.equal(roundSummary([horse("A", null), horse("B", null)], "UTC").subtitle, "2 Pferde sind noch offen.");
});
