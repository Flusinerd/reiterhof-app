import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import {
  actionLabel,
  blanketErrorMessage,
  coveredNights,
  fillLabel,
  firstName,
  greeting,
  historyDayLabel,
  myBlanketLines,
  nightLine,
  nightTempShort,
  nightUnit,
  progressFraction,
  progressLabel,
  progressText,
  recommendationDetail,
  recommendationTitle,
  reminderHint,
  ruleBlanketName,
  ruleCondition,
  hasWeatherDetails,
  rainAmount,
  rainIntensity,
  rainTiming,
  startWeatherText,
  temperatureRange,
  weatherBarFraction,
  weatherFacts,
  windowLabel,
  stateActions,
  stateByline,
  stateLabel,
  type Blanket,
  type BlanketState,
  type Recommendation,
  type TodayHorse,
  type Weather,
} from "./blankets.ts";

const weather = (night_min_c: number, will_rain: boolean): Weather => ({
  night_min_c,
  will_rain,
  rain_probability: 60,
  rain_mm: 1,
  wind_kmh: 10,
  fetched_at: "2026-09-30T16:00:00Z",
  window_start: "2026-09-30T16:00:00Z",
  window_end: "2026-10-01T10:30:00Z",
  temp_max_c: night_min_c,
  rain_peak_mm: 0,
  rain_hours: 0,
  rain_from: null,
  rain_until: null,
  timeline: [],
});

const blanket: Blanket = {
  id: "b1",
  horse_id: "h1",
  name: "Decke 100 g",
  fill_g: 100,
  color: "blau",
  location: "Haken 4",
  photo_path: null,
  photo_url: null,
};

const rec = (status: Recommendation["status"], b: Blanket | null = null): Recommendation => ({
  status,
  rule_index: status === "blanket" || status === "none" ? 0 : null,
  blanket: b,
  note: "",
});

const state = (patch: Partial<BlanketState> = {}): BlanketState => ({
  id: "s1",
  horse_id: "h1",
  day: "2026-09-30",
  action: "covered",
  covered_with: "b1",
  covered_with_name: "Decke 100 g",
  changed_at: "2026-09-30T17:04:00Z",
  changed_by: { id: "u1", name: "Mia" },
  automatic: false,
  ...patch,
});

test("night line", () => {
  assert.equal(nightLine(weather(3, true)), "Heute Nacht 3 °C, 1 mm Regen");
  assert.equal(nightLine(weather(8.4, false)), "Heute Nacht 8 °C, 1 mm Regen");
  assert.equal(nightLine({ ...weather(-1.6, false), rain_mm: 0 }), "Heute Nacht -2 °C, trocken");
  assert.equal(nightLine({ ...weather(3, true), temp_max_c: 9.4, rain_mm: 4.8 }), "Heute Nacht 3 bis 9 °C, 4,8 mm Regen");
  assert.equal(nightUnit({ ...weather(3, true), rain_mm: 4.8 }), "heute Nacht, 4,8 mm Regen");
  assert.equal(nightUnit(null), "heute Nacht");
  assert.match(nightLine(null), /Noch keine Vorhersage/);
  assert.equal(nightTempShort(weather(2.6, false)), "3°");
  assert.equal(nightTempShort(null), "–");
});

test("progress", () => {
  assert.equal(progressLabel({ done: 4, total: 7 }), "4/7");
  assert.equal(progressText({ done: 4, total: 7 }), "Noch 3 Pferde offen");
  assert.equal(progressText({ done: 6, total: 7 }), "Noch 1 Pferd offen");
  assert.equal(progressText({ done: 7, total: 7 }), "Alle versorgt");
  assert.equal(progressText({ done: 0, total: 0 }), "Noch keine Pferde");
  assert.equal(progressFraction({ done: 1, total: 4 }), 0.25);
  assert.equal(progressFraction({ done: 0, total: 0 }), 1);
  assert.equal(progressFraction({ done: 9, total: 4 }), 1);
  assert.equal(reminderHint("20:30"), "Erinnerung um 20:30");
});

test("recommendation texts", () => {
  assert.equal(recommendationTitle(rec("blanket", blanket)), "Decke 100 g");
  assert.equal(recommendationTitle(rec("none")), "Keine Decke");
  assert.equal(recommendationTitle(rec("no_rule")), "Keine passende Regel");
  assert.equal(recommendationTitle(rec("no_weather")), "Noch keine Empfehlung");
  assert.equal(recommendationDetail(rec("blanket", blanket)), "Ort: Haken 4");
  assert.equal(recommendationDetail(rec("blanket", { ...blanket, location: null })), "Ort nicht eingetragen");
  assert.match(recommendationDetail(rec("none")), /Deckenplan/);
});

test("buttons depend on the recommendation", () => {
  assert.deepEqual(stateActions(rec("blanket", blanket)), ["covered", "uncovered"]);
  assert.deepEqual(stateActions(rec("none")), ["checked"]);
  assert.deepEqual(stateActions(rec("no_weather")), ["covered", "uncovered", "checked"]);
  assert.deepEqual(stateActions(rec("no_rule")), ["covered", "uncovered", "checked"]);
  assert.equal(actionLabel("covered"), "Eingedeckt");
  assert.equal(actionLabel("uncovered"), "Abgedeckt");
  assert.equal(actionLabel("checked"), "Geprüft");
});

test("state labels", () => {
  assert.equal(stateLabel(state()), "Eingedeckt mit Decke 100 g");
  assert.equal(stateLabel(state({ covered_with_name: null })), "Eingedeckt");
  assert.equal(stateLabel(state({ action: "uncovered", covered_with: null, covered_with_name: null })), "Abgedeckt");
  assert.equal(stateLabel(state({ action: "checked" })), "Geprüft");
  // 17:04 UTC is 19:04 in Berlin during summer time.
  assert.equal(stateByline(state(), "Europe/Berlin"), "Mia, 19:04");
  assert.equal(stateByline(state({ changed_by: null }), "Europe/Berlin"), "19:04");
  // Automatic uncovering by the farm staff: no author, shown as "Hof".
  const auto = state({ action: "uncovered", covered_with: null, covered_with_name: null, changed_by: null, automatic: true });
  assert.equal(stateLabel(auto), "Abgedeckt (Hof, automatisch)");
  assert.equal(stateByline(auto, "Europe/Berlin"), "Hof, 19:04");
  assert.equal(stateLabel({ ...auto, automatic: false }), "Abgedeckt");
});

test("rule texts", () => {
  assert.equal(ruleCondition({ temp_min: 0, temp_max: 5, rain: null }), "0 bis unter 5 °C");
  assert.equal(ruleCondition({ temp_min: 12, temp_max: null, rain: null }), "ab 12 °C");
  assert.equal(ruleCondition({ temp_min: null, temp_max: 0, rain: true }), "unter 0 °C, bei Regen");
  assert.equal(ruleCondition({ temp_min: 5, temp_max: 12, rain: false }), "5 bis unter 12 °C, ohne Regen");
  assert.equal(ruleCondition({ temp_min: null, temp_max: null, rain: null }), "Jede Temperatur");
  assert.equal(ruleBlanketName({ blanket_id: "b1" }, [blanket]), "Decke 100 g");
  assert.equal(ruleBlanketName({ blanket_id: null }, [blanket]), "Keine Decke");
  assert.equal(ruleBlanketName({ blanket_id: "gone" }, [blanket]), "Keine Decke");
  assert.equal(fillLabel(150), "Füllung 150 g");
  assert.equal(fillLabel(0), "ohne Füllung");
});

test("history day labels", () => {
  assert.equal(historyDayLabel("2026-09-30", "2026-09-30"), "Heute Nacht");
  assert.equal(historyDayLabel("2026-09-29", "2026-09-30"), "Gestern Nacht");
  assert.equal(historyDayLabel("2026-09-28", "2026-09-30"), "Nacht vom 28.09.");
  assert.equal(historyDayLabel("2026-09-30", "2026-10-01"), "Gestern Nacht");
  assert.equal(historyDayLabel("2026-02-28", "2026-03-01"), "Gestern Nacht");
  assert.equal(historyDayLabel("garbage", "2026-09-30"), "garbage");
});

test("covered nights count each day once", () => {
  const list = [
    state({ id: "1", day: "2026-09-30" }),
    state({ id: "2", day: "2026-09-30" }),
    state({ id: "3", day: "2026-09-29", action: "uncovered" }),
    state({ id: "4", day: "2026-09-28" }),
  ];
  assert.equal(coveredNights(list), 2);
});

test("error messages", () => {
  assert.match(blanketErrorMessage(new ApiError(403, "forbidden", "x")), /Besitzerin/);
  assert.match(blanketErrorMessage(new ApiError(409, "in_use", "x")), /Regel/);
  assert.match(blanketErrorMessage(new ApiError(413, "file_too_large", "x")), /20 MB/);
  assert.match(blanketErrorMessage(new ApiError(0, "network", "x")), /Keine Verbindung/);
  assert.match(blanketErrorMessage(new Error("boom")), /schiefgelaufen/);
});

test("greeting", () => {
  assert.equal(greeting(7, "Jan Krüger"), "Guten Morgen, Jan");
  assert.equal(greeting(14, "Jan"), "Guten Tag, Jan");
  assert.equal(greeting(20, "Jan"), "Guten Abend, Jan");
  assert.equal(greeting(20, "  "), "Guten Abend");
  assert.equal(firstName(" Mia  Muster "), "Mia");
});

test("my blanket lines on the start screen", () => {
  const horse = (id: string, name: string, r: Recommendation): TodayHorse => ({
    horse: { id, name, box: null, color_key: null },
    recommendation: r,
    state: null,
    done: false,
    is_mine: false,
  });
  const horses = [
    horse("h1", "Luna", rec("blanket", blanket)),
    horse("h2", "Balu", rec("none")),
    horse("h3", "Fanta", rec("blanket", blanket)),
    horse("h4", "Nala", rec("no_weather")),
  ];
  const lines = myBlanketLines(horses, new Set(["h1", "h2", "h4"]));
  assert.deepEqual(lines, ["Luna: Decke 100 g", "Balu: Keine Decke"]);
  assert.equal(startWeatherText(lines, true, true), "Luna: Decke 100 g\nBalu: Keine Decke");
  assert.match(startWeatherText([], false, true), /Noch keine Vorhersage/);
  assert.match(startWeatherText([], true, false), /Keine eigenen Pferde/);
  assert.match(startWeatherText([], true, true), /Noch keine Empfehlung/);
});

test("rain amount and intensity", () => {
  assert.equal(rainAmount({ rain_mm: 0 }), "trocken");
  assert.equal(rainAmount({ rain_mm: 0.3 }), "kaum Regen");
  assert.equal(rainAmount({ rain_mm: 1.54 }), "1,5 mm Regen");
  assert.equal(rainAmount({ rain_mm: 12 }), "12 mm Regen");
  assert.deepEqual([0, 0.3, 1, 3, 8].map(rainIntensity), [
    "trocken",
    "kaum Regen",
    "leichter Regen",
    "mäßiger Regen",
    "starker Regen",
  ]);
  assert.equal(weatherBarFraction(2), 0.5);
  assert.equal(weatherBarFraction(10), 1);
});

test("temperature range", () => {
  assert.equal(temperatureRange({ night_min_c: 3.2, temp_max_c: 3.4 }), "3 °C");
  assert.equal(temperatureRange({ night_min_c: 3.2, temp_max_c: 9.4 }), "3 bis 9 °C");
});

test("weather details", () => {
  const w: Weather = {
    ...weather(3, true),
    temp_max_c: 9,
    rain_mm: 4.8,
    rain_probability: 70,
    wind_kmh: 24.4,
    rain_peak_mm: 2,
    rain_hours: 3,
    rain_from: "2026-09-30T20:00:00Z",
    rain_until: "2026-10-01T00:00:00Z",
    timeline: [{ time: "2026-09-30T16:00:00Z", temp_c: 8, rain_mm: 0, rain_prob: 0, wind_kmh: 10 }],
  };
  assert.equal(hasWeatherDetails(w), true);
  assert.equal(hasWeatherDetails(weather(3, true)), false);
  assert.equal(windowLabel(w, "Europe/Berlin"), "18:00 bis 12:30 Uhr");
  assert.equal(windowLabel(weather(3, true), "Europe/Berlin"), "über Nacht");
  assert.equal(rainTiming(w, "Europe/Berlin"), "22:00 bis 02:00 Uhr");
  assert.equal(rainTiming(weather(3, false), "Europe/Berlin"), "trocken");
  assert.deepEqual(weatherFacts(w, "Europe/Berlin"), [
    { label: "Temperatur", value: "3 bis 9 °C" },
    { label: "Regen", value: "4,8 mm (mäßiger Regen)" },
    { label: "Regenzeit", value: "22:00 bis 02:00 Uhr" },
    { label: "Stärkster Regen", value: "2 mm pro Stunde" },
    { label: "Regenwahrscheinlichkeit", value: "bis 70 %" },
    { label: "Wind", value: "bis 24 km/h" },
  ]);
  const dry = weatherFacts({ ...w, rain_mm: 0, rain_from: null, rain_until: null }, "Europe/Berlin");
  assert.deepEqual(dry.map((f) => f.label), ["Temperatur", "Regen", "Regenwahrscheinlichkeit", "Wind"]);
});
