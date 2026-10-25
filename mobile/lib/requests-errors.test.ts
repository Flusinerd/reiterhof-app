import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError } from "./api-core.ts";
import { requestErrorMessage, requestLoadFailure } from "./requests-errors.ts";

test("request specific codes have German texts", () => {
  assert.match(requestErrorMessage(new ApiError(409, "request_full", "x")), /vergeben/);
  assert.match(requestErrorMessage(new ApiError(409, "not_open", "x")), /erledigt oder abgesagt/);
  assert.match(requestErrorMessage(new ApiError(403, "own_request", "x")), /eigene Anfrage/);
  assert.match(requestErrorMessage(new ApiError(404, "not_found", "x")), /nicht mehr/);
});

test("other errors use the generic messages", () => {
  assert.match(requestErrorMessage(new ApiError(0, "network", "x")), /Verbindung/);
  assert.match(requestErrorMessage(new Error("boom")), /schiefgelaufen/);
});

test("only a 404 says that the request was not found; a network error can be retried", () => {
  assert.deepEqual(requestLoadFailure(new ApiError(404, "not_found", "x")), { title: "Anfrage nicht gefunden", retry: false });
  assert.deepEqual(requestLoadFailure(new ApiError(0, "network", "x")), { title: "Laden fehlgeschlagen", retry: true });
  assert.deepEqual(requestLoadFailure(new ApiError(500, "internal", "x")), { title: "Laden fehlgeschlagen", retry: true });
  assert.deepEqual(requestLoadFailure(new Error("boom")), { title: "Laden fehlgeschlagen", retry: true });
});
