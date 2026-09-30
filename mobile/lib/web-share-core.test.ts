import assert from "node:assert/strict";
import { test } from "node:test";

import { chooseShareStrategy, icsFileName } from "./web-share-core.ts";

const file = {} as File;

test("shares the file when the browser can", () => {
  assert.equal(chooseShareStrategy({ share: () => {}, canShare: () => true }, file), "share-file");
});

test("downloads when sharing is missing or refused", () => {
  assert.equal(chooseShareStrategy({}, file), "download");
  assert.equal(chooseShareStrategy({ share: () => {} }, file), "download");
  assert.equal(chooseShareStrategy({ share: () => {}, canShare: () => false }, file), "download");
  assert.equal(
    chooseShareStrategy(
      {
        share: () => {},
        canShare: () => {
          throw new Error("nope");
        },
      },
      file,
    ),
    "download",
  );
});

test("ics file names are safe", () => {
  assert.equal(icsFileName("3f2a-91"), "stallfunk-termin-3f2a-91.ics");
  assert.equal(icsFileName("../x/y"), "stallfunk-termin-xy.ics");
  assert.equal(icsFileName(), "stallfunk-termine.ics");
});
