import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";

import { render, versionOf } from "../../scripts/gen-legal.mjs";
import { CONSENT_COPY, CONSENT_KINDS } from "./consent-core.ts";
import { IMPRINT_MD, LEGAL_TEXT_VERSION, PRIVACY_MD } from "./consent-legal-texts.ts";
import { parseMarkdown } from "./consent-markdown.ts";

const DRAFT = "Entwurf – vor Veröffentlichung rechtlich prüfen lassen";

test("consent-legal-texts.ts is generated from docs/legal (run node scripts/gen-legal.mjs)", () => {
  const current = readFileSync(new URL("./consent-legal-texts.ts", import.meta.url), "utf8");
  assert.equal(current, render());
});

test("both texts carry the draft marker and the same version", () => {
  assert.ok(PRIVACY_MD.includes(DRAFT));
  assert.ok(IMPRINT_MD.includes(DRAFT));
  assert.equal(versionOf(PRIVACY_MD), LEGAL_TEXT_VERSION);
  assert.equal(versionOf(IMPRINT_MD), LEGAL_TEXT_VERSION);
});

test("the privacy text covers every data category and consent", () => {
  for (const topic of [
    "[Name, Anschrift, E-Mail]",
    "Anwesenheit",
    "Geofence",
    "GPS",
    "Fotos",
    "Notfallkarte",
    "Netcup",
    "Backups",
    "Expo",
    "Google",
    "Apple",
    "Beschwerde",
  ]) {
    assert.ok(PRIVACY_MD.includes(topic), topic);
  }
});

test("the texts parse into blocks and contain no unsupported Markdown", () => {
  for (const md of [PRIVACY_MD, IMPRINT_MD]) {
    const blocks = parseMarkdown(md);
    assert.ok(blocks.length > 5);
    assert.equal(blocks[0].type, "heading");
    // tables, code, links and images would show up as literal characters in the app
    assert.doesNotMatch(md, /^\s*\|/m);
    assert.doesNotMatch(md, /```/);
    assert.doesNotMatch(md, /\]\(/);
  }
});

test("every consent kind is mentioned by its copy", () => {
  for (const kind of CONSENT_KINDS) assert.ok(CONSENT_COPY[kind].label.length > 0);
});
