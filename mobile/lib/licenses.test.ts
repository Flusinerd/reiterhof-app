import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { test } from "node:test";

import { isAllowed, licenseIds, OUTPUT, render } from "../scripts/gen-licenses.mjs";

test("license expressions", () => {
  assert.deepEqual(licenseIds("MIT"), ["MIT"]);
  assert.deepEqual(licenseIds("(MIT OR Apache-2.0)"), ["MIT", "Apache-2.0"]);
  assert.deepEqual(licenseIds("MIT AND OFL-1.1"), ["MIT", "OFL-1.1"]);
  assert.equal(isAllowed("MIT"), true);
  assert.equal(isAllowed("(BSD-3-Clause OR GPL-2.0)"), true, "a dual license with a permissive option");
  assert.equal(isAllowed("MIT AND OFL-1.1"), true);
  assert.equal(isAllowed("GPL-3.0"), false);
  assert.equal(isAllowed("(GPL-2.0 OR AGPL-3.0)"), false);
  assert.equal(isAllowed("MIT AND GPL-2.0"), false, "a combined license needs every part allowed");
  assert.equal(isAllowed(""), false);
  assert.equal(isAllowed(undefined), false);
});

test("lib/licenses.generated.ts exists and matches the installed packages (npm ci regenerates it)", () => {
  assert.ok(existsSync(OUTPUT), "run node scripts/gen-licenses.mjs");
  const current = readFileSync(OUTPUT, "utf8");
  assert.equal(current, render());
  assert.match(current, /"name":"react-native"/);
  assert.match(current, /"name":"expo"/);
  assert.match(current, /OFL-1\.1/, "the bundled fonts carry their license");
});
