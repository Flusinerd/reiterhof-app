import assert from "node:assert/strict";
import { test } from "node:test";

import {
  BODY_PARTS,
  bodyPartLabel,
  bodyPartShort,
  bundleText,
  categoryLabel,
  CATEGORIES,
  horseNamesText,
  MAX_DESCRIPTION,
  MAX_PHOTOS,
  newObservationRoute,
  nextStatus,
  observationRoute,
  observationTitle,
  openCount,
  reportedText,
  reportProblem,
  statusActionLabel,
  statusBadgeVariant,
  statusLabel,
  urgencyBadgeVariant,
  urgencyHint,
  urgencyLabel,
  URGENCIES,
} from "./observations.ts";

test("every category and body part has a German label", () => {
  assert.equal(categoryLabel("cough"), "Husten");
  assert.equal(categoryLabel("lameness"), "Lahmheit");
  assert.equal(categoryLabel("injury"), "Verletzung");
  assert.equal(categoryLabel("not_eating"), "Frisst nicht");
  assert.equal(categoryLabel("colic"), "Kolik");
  assert.equal(categoryLabel("behavior"), "Verhalten");
  assert.equal(categoryLabel("blanket_equipment"), "Decke/Ausrüstung");
  assert.equal(categoryLabel("other"), "Sonstiges");
  for (const c of CATEGORIES) assert.notEqual(categoryLabel(c), c);
  for (const b of BODY_PARTS) assert.notEqual(bodyPartLabel(b), b);
  assert.equal(categoryLabel("mystery"), "mystery");
  assert.equal(categoryLabel(null), "Auffälligkeit");
  assert.equal(bodyPartLabel(null), "");
  assert.equal(bodyPartLabel("front_left"), "Vorne links");
  assert.equal(bodyPartLabel("belly"), "Bauch");
});

test("leg captions", () => {
  assert.deepEqual(
    ["front_left", "front_right", "hind_left", "hind_right", "head"].map(bodyPartShort),
    ["VL", "VR", "HL", "HR", ""],
  );
});

test("urgency labels, variants and hints", () => {
  assert.deepEqual(URGENCIES.map(urgencyLabel), ["Info", "Bitte ansehen", "Dringend"]);
  assert.deepEqual(URGENCIES.map(urgencyBadgeVariant), ["neutral", "accent", "danger"]);
  for (const u of URGENCIES) assert.ok(urgencyHint(u).length > 10);
  assert.match(urgencyHint("urgent"), /Notfallkarte/);
});

test("status labels and toggling", () => {
  assert.equal(statusLabel("watch"), "Beobachten");
  assert.equal(statusLabel("done"), "Erledigt");
  assert.equal(statusBadgeVariant("done"), "primary");
  assert.equal(statusBadgeVariant("watch"), "info");
  assert.equal(nextStatus("watch"), "done");
  assert.equal(nextStatus("done"), "watch");
  assert.equal(statusActionLabel("watch"), "Als erledigt markieren");
  assert.equal(statusActionLabel("done"), "Wieder beobachten");
});

test("observationTitle joins category and body part", () => {
  assert.equal(observationTitle({ category: "cough", body_part: "head" }), "Husten · Kopf");
  assert.equal(observationTitle({ category: "colic", body_part: null }), "Kolik");
  assert.equal(observationTitle({ category: null, body_part: null }), "Auffälligkeit");
});

test("reportedText", () => {
  const now = new Date(2026, 9, 1, 12, 0);
  assert.equal(reportedText(new Date(2026, 9, 1, 8, 0).toISOString(), now), "heute");
  assert.equal(reportedText(new Date(2026, 8, 30, 23, 0).toISOString(), now), "gestern");
  assert.equal(reportedText(new Date(2026, 8, 26, 9, 0).toISOString(), now), "vor 5 Tagen");
  assert.equal(reportedText(new Date(2026, 8, 10, 9, 0).toISOString(), now), "10.09.2026");
  assert.equal(reportedText("garbage", now), "");
});

test("openCount counts observations to watch", () => {
  assert.equal(openCount([{ status: "watch" }, { status: "done" }, { status: "watch" }]), 2);
  assert.equal(openCount([]), 0);
});

test("bundleText", () => {
  assert.equal(bundleText(3), "3 weitere Pferde sind ebenfalls fällig – Termin bündeln?");
  assert.equal(bundleText(2), "2 weitere Pferde sind ebenfalls fällig – Termin bündeln?");
  assert.equal(bundleText(1), "1 weiteres Pferd ist ebenfalls fällig – Termin bündeln?");
  assert.equal(bundleText(0), "");
});

test("horseNamesText", () => {
  assert.equal(horseNamesText([]), "");
  assert.equal(horseNamesText(["Fanta"]), "Fanta");
  assert.equal(horseNamesText(["Fanta", "Nala"]), "Fanta und Nala");
  assert.equal(horseNamesText(["Cookie", "Fanta", "Nala"]), "Cookie, Fanta und Nala");
});

test("reportProblem validates the draft", () => {
  const ok = { horseId: "h1", category: "cough", description: "", photos: 0 };
  assert.equal(reportProblem(ok), null);
  assert.match(reportProblem({ ...ok, horseId: null }) ?? "", /Pferd/);
  assert.match(reportProblem({ ...ok, category: null }) ?? "", /aufgefallen/);
  assert.match(reportProblem({ ...ok, description: "x".repeat(MAX_DESCRIPTION + 1) }) ?? "", /zu lang/);
  assert.equal(reportProblem({ ...ok, description: "x".repeat(MAX_DESCRIPTION) }), null);
  assert.match(reportProblem({ ...ok, photos: MAX_PHOTOS + 1 }) ?? "", /Fotos/);
  assert.equal(reportProblem({ ...ok, photos: MAX_PHOTOS }), null);
});

test("routes", () => {
  assert.equal(observationRoute("abc"), "/observations/abc");
  assert.equal(newObservationRoute("h 1"), "/observations/new?horse=h%201");
  assert.equal(newObservationRoute(), "/observations/new");
  assert.equal(newObservationRoute(null), "/observations/new");
});
