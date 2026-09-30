import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import { absoluteUrl, downloadLinkPath, uploadErrorMessage } from "./upload-core.ts";

test("upload errors are German and specific", () => {
  assert.match(uploadErrorMessage(new ApiError(413, "file_too_large", "x")), /zu groß/);
  assert.match(uploadErrorMessage(new ApiError(415, "unsupported_type", "x")), /PDF/);
  assert.doesNotMatch(uploadErrorMessage(new ApiError(415, "unsupported_type", "x")), /HEIC/);
  assert.match(uploadErrorMessage(new ApiError(400, "invalid_upload", "x")), /nicht gelesen/);
  assert.match(uploadErrorMessage(new ApiError(0, "network", "x")), /Verbindung/);
  assert.match(uploadErrorMessage(new ApiError(500, "internal", "x")), /Upload/);
  assert.match(uploadErrorMessage(new Error("boom")), /Upload/);
});

test("absoluteUrl", () => {
  assert.equal(absoluteUrl("http://10.0.2.2:8080/", "/api/v1/files/a.png"), "http://10.0.2.2:8080/api/v1/files/a.png");
  assert.equal(absoluteUrl("http://x", "api/v1/files/a.png"), "http://x/api/v1/files/a.png");
  assert.equal(absoluteUrl("http://x", "https://cdn/y.png"), "https://cdn/y.png");
});

test("downloadLinkPath accepts file routes only, without origin and query", () => {
  const file = "/api/v1/files/00000000-0000-4000-8000-000000000101/0123456789abcdef0123456789abcdef.png";
  const doc = "/api/v1/horses/00000000-0000-4000-8000-000000000301/documents/00000000-0000-4000-8000-000000000901/file";
  assert.equal(downloadLinkPath("http://x", file), file);
  assert.equal(downloadLinkPath("http://x/", "http://x" + file), file);
  assert.equal(downloadLinkPath("http://x", doc + "?v=1"), doc);
  assert.equal(downloadLinkPath("http://x", "http://other" + file), null);
  assert.equal(downloadLinkPath("http://x", "/api/v1/horses"), null);
  assert.equal(downloadLinkPath("http://x", "/api/v1/horses/1/documents/2"), null);
  assert.equal(downloadLinkPath("http://x", ""), null);
});
