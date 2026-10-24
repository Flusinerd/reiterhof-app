import assert from "node:assert/strict";
import { test } from "node:test";

import { severityLabel, severityTone, warningSpan, warningSummary, type WeatherWarning } from "./weather-warnings.ts";

const tz = "Europe/Berlin";

test("warningSpan formats the range in the stable's time zone", () => {
  assert.equal(warningSpan("2026-10-24T12:00:00Z", "2026-10-24T20:00:00Z", tz), "Sa 24.10., 14:00 bis 22:00 Uhr");
  assert.equal(warningSpan("2026-10-24T12:00:00Z", "2026-10-25T07:00:00Z", tz), "Sa 24.10., 14:00 bis So 25.10., 08:00 Uhr");
  assert.equal(warningSpan("2026-10-24T12:00:00Z", null, tz), "ab Sa 24.10., 14:00 Uhr");
  assert.equal(warningSpan("kaputt", null, tz), "");
});

test("severity labels and tones", () => {
  assert.equal(severityLabel("minor"), "Wetterwarnung");
  assert.equal(severityLabel("severe"), "Unwetterwarnung");
  assert.equal(severityLabel("unknown"), "Wetterwarnung");
  assert.equal(severityTone("minor"), "info");
  assert.equal(severityTone("moderate"), "accent");
  assert.equal(severityTone("extreme"), "danger");
});

test("warningSummary names the most severe warning", () => {
  const w = (severity: WeatherWarning["severity"], headline: string): WeatherWarning => ({
    id: headline,
    event: "",
    severity,
    headline,
    description: "",
    instruction: "",
    onset: "2026-10-24T12:00:00Z",
    expires: null,
    fetched_at: "2026-10-24T10:00:00Z",
  });
  assert.equal(warningSummary([]), "");
  assert.equal(warningSummary([w("severe", "Amtliche UNWETTERWARNUNG vor ORKANBÖEN")]), "Unwetterwarnung: Amtliche UNWETTERWARNUNG vor ORKANBÖEN");
  assert.equal(
    warningSummary([w("moderate", "Amtliche WARNUNG vor STURMBÖEN"), w("minor", "Amtliche WARNUNG vor FROST")]),
    "Markante Wetterwarnung: Amtliche WARNUNG vor STURMBÖEN (+1 weitere)",
  );
});
