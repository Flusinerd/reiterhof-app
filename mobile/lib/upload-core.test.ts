import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import {
  absoluteUrl,
  downloadLinkPath,
  uploadContentType,
  uploadErrorMessage,
  uploadFileName,
  uploadSource,
} from "./upload-core.ts";

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

test("uploadContentType trusts the file that is sent, not the picker's label", () => {
  // Android re-encodes a HEIF photo to .jpeg and still reports image/heif.
  assert.equal(uploadContentType("file:///data/cache/ImagePicker/1a2b.jpeg", "image/heif"), "image/jpeg");
  assert.equal(uploadContentType("file:///data/cache/ImagePicker/1a2b.jpeg", null), "image/jpeg");
  assert.equal(uploadContentType("file:///tmp/x.JPG", "image/heic"), "image/jpeg");
  assert.equal(uploadContentType("file:///tmp/x.png", "image/png"), "image/png");
  assert.equal(uploadContentType("file:///tmp/scan.pdf?x=1", undefined), "application/pdf");
  // No usable extension: an accepted declared type counts, anything else is left to the server.
  assert.equal(uploadContentType("content://media/external/images/42", "image/webp"), "image/webp");
  assert.equal(uploadContentType("content://media/external/images/42", "image/jpg"), "image/jpeg");
  assert.equal(uploadContentType("file:///tmp/x.heic", "image/heic"), "application/octet-stream");
  assert.equal(uploadContentType("file:///tmp/x", ""), "application/octet-stream");
  // Web: blob: URLs carry no extension, the browser's type is right.
  assert.equal(uploadContentType("blob:http://localhost:8081/4f6c.png", "image/jpeg"), "image/jpeg");
  assert.equal(uploadContentType("data:image/png;base64,AAAA", "image/png; charset=binary"), "image/png");
});

test("uploadFileName matches the extension to the content", () => {
  assert.equal(uploadFileName("IMG_0001.HEIC", "image/jpeg"), "IMG_0001.jpg");
  assert.equal(uploadFileName("IMG_0001.jpeg", "image/jpeg"), "IMG_0001.jpeg");
  assert.equal(uploadFileName("foto", "image/png"), "foto.png");
  assert.equal(uploadFileName("Pass.pdf", "application/pdf"), "Pass.pdf");
  assert.equal(uploadFileName("x.heic", "application/octet-stream"), "x.heic");
  assert.equal(uploadFileName("  ", "image/jpeg"), "datei.jpg");
  assert.equal(uploadFileName("a/b.jpg", "image/jpeg"), "a_b.jpg");
});

test("uploadSource: web files as blobs, native URIs as React Native file objects", () => {
  assert.equal(uploadSource({ uri: "blob:http://localhost/1", file: new Blob(["x"]) }), "blob");
  assert.equal(uploadSource({ uri: "blob:http://localhost/1" }), "fetch");
  assert.equal(uploadSource({ uri: "data:image/jpeg;base64,AAAA", file: null }), "fetch");
  assert.equal(uploadSource({ uri: "file:///var/mobile/x.jpg" }), "native");
  assert.equal(uploadSource({ uri: "content://com.android.providers/x" }), "native");
});
