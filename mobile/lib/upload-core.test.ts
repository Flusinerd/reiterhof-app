import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import { absoluteUrl, uploadErrorMessage, withAccessToken } from "./upload-core.ts";

test("upload errors are German and specific", () => {
  assert.match(uploadErrorMessage(new ApiError(413, "file_too_large", "x")), /zu groß/);
  assert.match(uploadErrorMessage(new ApiError(415, "unsupported_type", "x")), /PDF/);
  assert.match(uploadErrorMessage(new ApiError(0, "network", "x")), /Verbindung/);
  assert.match(uploadErrorMessage(new ApiError(500, "internal", "x")), /Upload/);
  assert.match(uploadErrorMessage(new Error("boom")), /Upload/);
});

test("absoluteUrl", () => {
  assert.equal(absoluteUrl("http://10.0.2.2:8080/", "/api/v1/files/a.png"), "http://10.0.2.2:8080/api/v1/files/a.png");
  assert.equal(absoluteUrl("http://x", "api/v1/files/a.png"), "http://x/api/v1/files/a.png");
  assert.equal(absoluteUrl("http://x", "https://cdn/y.png"), "https://cdn/y.png");
});

test("withAccessToken", () => {
  assert.equal(withAccessToken("http://x/f", "a b"), "http://x/f?access_token=a%20b");
  assert.equal(withAccessToken("http://x/f?v=1", "t"), "http://x/f?v=1&access_token=t");
  assert.equal(withAccessToken("http://x/f", null), "http://x/f");
});
