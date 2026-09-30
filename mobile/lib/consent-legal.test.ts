import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { render, versionOf } from "../../scripts/gen-legal.mjs";
import { CONSENT_COPY, CONSENT_KINDS } from "./consent-core.ts";
import { LEGAL_TEXT_VERSION, PRIVACY_MD } from "./consent-legal-texts.ts";
import { parseMarkdown } from "./consent-markdown.ts";

const DRAFT = "Entwurf – vor Veröffentlichung rechtlich prüfen lassen";

test("consent-legal-texts.ts is generated from docs/legal (run node scripts/gen-legal.mjs)", () => {
  const current = readFileSync(new URL("./consent-legal-texts.ts", import.meta.url), "utf8");
  assert.equal(current, render());
});

test("the privacy text carries the draft marker and the version", () => {
  assert.ok(PRIVACY_MD.includes(DRAFT));
  assert.equal(versionOf(PRIVACY_MD), LEGAL_TEXT_VERSION);
});

test("the privacy text covers every data category and consent", () => {
  for (const topic of [
    "Wer ist verantwortlich",
    "Anwesenheit",
    "Geofence",
    "GPS",
    "Fotos",
    "Notfallkarte",
    "Netcup",
    "Backups",
    "Firebase Cloud Messaging",
    "Apple Push Notification service",
    "Google",
    "Apple",
    "OpenFreeMap",
    "Metadaten",
    "Art. 8",
    "Elternteil",
    "localStorage",
    "Beschwerde",
  ]) {
    assert.ok(PRIVACY_MD.includes(topic), topic);
  }
});

test("the privacy text parses into blocks and contains no unsupported Markdown", () => {
  const blocks = parseMarkdown(PRIVACY_MD);
  assert.ok(blocks.length > 5);
  assert.equal(blocks[0].type, "heading");
  // tables, code, links and images would show up as literal characters in the app
  assert.doesNotMatch(PRIVACY_MD, /^\s*\|/m);
  assert.doesNotMatch(PRIVACY_MD, /```/);
  assert.doesNotMatch(PRIVACY_MD, /\]\(/);
});

test("every consent kind is mentioned by its copy", () => {
  for (const kind of CONSENT_KINDS) assert.ok(CONSENT_COPY[kind].label.length > 0);
});
