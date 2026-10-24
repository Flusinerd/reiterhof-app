import assert from "node:assert/strict";
import { test } from "node:test";

import type { Weather } from "./blankets.ts";
import { weatherFigures, weatherSentence } from "./weather-summary.ts";

const base: Weather = {
  night_min_c: 3.2,
  will_rain: true,
  rain_probability: 70,
  rain_mm: 4.8,
  wind_kmh: 24.4,
  fetched_at: "2026-09-30T14:00:00Z",
  window_start: "2026-09-30T16:00:00Z",
  window_end: "2026-10-01T10:30:00Z",
  temp_max_c: 9.4,
  rain_peak_mm: 2,
  rain_hours: 3,
  rain_from: "2026-09-30T20:00:00Z",
  rain_until: "2026-10-01T00:00:00Z",
  timeline: [],
};

test("sentence: rain with and without timing, dry", () => {
  assert.equal(weatherSentence(base, "Europe/Berlin"), "Mäßiger Regen von 22:00 bis 02:00 Uhr, bis 70 % wahrscheinlich.");
  assert.equal(
    weatherSentence({ ...base, rain_from: null, rain_until: null, rain_mm: 0.3 }, "Europe/Berlin"),
    "Kaum Regen, bis 70 % wahrscheinlich.",
  );
  assert.equal(weatherSentence({ ...base, rain_mm: 0, will_rain: false }, "Europe/Berlin"), "Trocken im ganzen Zeitraum.");
  assert.equal(weatherSentence({ ...base, rain_mm: 12 }, "Europe/Berlin").startsWith("Starker Regen von"), true);
});

test("figures: temperature, rain and wind", () => {
  assert.deepEqual(weatherFigures(base), [
    { label: "Temperatur", value: "3–9", unit: "°C" },
    { label: "Regen", value: "4,8", unit: "mm" },
    { label: "Wind bis", value: "24", unit: "km/h" },
  ]);
  const flat = weatherFigures({ ...base, night_min_c: 8.2, temp_max_c: 8.4, rain_mm: 0 });
  assert.deepEqual(flat[0], { label: "Temperatur", value: "8", unit: "°C" });
  assert.deepEqual(flat[1], { label: "Regen", value: "0", unit: "mm" });
});
