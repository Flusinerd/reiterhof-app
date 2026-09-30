import assert from "node:assert/strict";
import { test } from "node:test";

import {
  CONSENT_COPY,
  CONSENT_KINDS,
  consentErrorMessage,
  deleteBlockedMessage,
  exportFileName,
  findConsent,
  formatExport,
  isGranted,
  needsPrompt,
  orderedConsents,
  type ConsentItem,
  type ConsentKind,
} from "./consent-core.ts";

function item(kind: ConsentKind, over: Partial<ConsentItem> = {}): ConsentItem {
  return {
    kind,
    granted: false,
    version: null,
    granted_at: null,
    revoked_at: null,
    current_version: "2026-09-30",
    up_to_date: false,
    ...over,
  };
}

test("every kind has German copy with a button and an explanation", () => {
  assert.equal(CONSENT_KINDS.length, 6);
  assert.ok(CONSENT_KINDS.includes("maps"));
  assert.match(CONSENT_COPY.maps.points.join(" "), /OpenFreeMap/);
  for (const kind of CONSENT_KINDS) {
    const c = CONSENT_COPY[kind];
    assert.ok(c.label && c.title && c.summary && c.declined && c.accept, kind);
    assert.ok(c.points.length >= 2, kind);
  }
});

test("isGranted and needsPrompt", () => {
  const items = [
    item("push", { granted: true, up_to_date: true, version: "2026-09-30" }),
    item("photos", { granted: true, up_to_date: false, version: "2025-01-01" }),
    item("location_geofence", { granted: false, revoked_at: "2026-09-01T10:00:00Z" }),
  ];
  assert.equal(isGranted(items, "push"), true);
  assert.equal(needsPrompt(items, "push"), false);

  // Granted for an older text: still granted, but the sheet is shown again.
  assert.equal(isGranted(items, "photos"), true);
  assert.equal(needsPrompt(items, "photos"), true);

  // Revoked and never seen kinds need a prompt.
  assert.equal(isGranted(items, "location_geofence"), false);
  assert.equal(needsPrompt(items, "location_geofence"), true);
  assert.equal(isGranted(items, "presence_sharing"), false);
  assert.equal(needsPrompt(items, "presence_sharing"), true);

  // Not loaded yet: ask (the server enforces anyway).
  assert.equal(isGranted(undefined, "push"), false);
  assert.equal(needsPrompt(undefined, "push"), true);
});

test("findConsent finds by kind", () => {
  const items = [item("push"), item("photos")];
  assert.equal(findConsent(items, "photos")?.kind, "photos");
  assert.equal(findConsent(items, "location_tracking"), undefined);
});

test("orderedConsents sorts by kind order and drops unknown kinds", () => {
  const items = [item("push"), { ...item("photos"), kind: "future_kind" as ConsentKind }, item("location_geofence"), item("presence_sharing")];
  assert.deepEqual(
    orderedConsents(items).map((i) => i.kind),
    ["location_geofence", "presence_sharing", "push"],
  );
});

test("deleteBlockedMessage explains the two blocking reasons", () => {
  assert.match(deleteBlockedMessage("owns_horses") ?? "", /Pferd/);
  assert.match(deleteBlockedMessage("last_admin") ?? "", /Admin/);
  assert.equal(deleteBlockedMessage("network"), null);
});

test("exportFileName uses the UTC date", () => {
  assert.equal(exportFileName(new Date("2026-09-30T23:30:00Z")), "reiterhof-export-2026-09-30.json");
});

test("formatExport is readable JSON that round-trips", () => {
  const data = { profile: { name: "Mia" }, list: [1, 2] };
  const text = formatExport(data);
  assert.ok(text.includes("\n"));
  assert.deepEqual(JSON.parse(text), data);
});

test("consentErrorMessage explains a text version conflict", () => {
  assert.match(consentErrorMessage("version_mismatch") ?? "", /Aktualisiere/);
  assert.equal(consentErrorMessage("internal"), null);
});
