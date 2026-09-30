import assert from "node:assert/strict";
import { test } from "node:test";

import { ApiError, createClient, errorMessage, parseErrorBody } from "./api-core.ts";

test("parseErrorBody reads the backend envelope", () => {
  const err = parseErrorBody(404, '{"error":{"code":"invalid_code","message":"nope"}}');
  assert.ok(err instanceof ApiError);
  assert.equal(err.status, 404);
  assert.equal(err.code, "invalid_code");
  assert.equal(err.message, "nope");
});

test("parseErrorBody falls back for non-envelope bodies", () => {
  assert.equal(parseErrorBody(502, "<html>Bad gateway</html>").code, "internal");
  assert.equal(parseErrorBody(400, "").code, "unknown");
  assert.equal(parseErrorBody(400, '{"error":"text"}').code, "unknown");
  assert.equal(parseErrorBody(400, "null").code, "unknown");
});

test("errorMessage is German and never leaks server text", () => {
  assert.match(errorMessage(new ApiError(404, "invalid_code", "english")), /Code/);
  assert.match(errorMessage(new ApiError(0, "network", "x")), /Verbindung/);
  assert.match(errorMessage(new Error("boom")), /schiefgelaufen/);
});

type Seen = { url?: string; init?: RequestInit };

function fakeFetch(status: number, body: string, seen: Seen = {}): typeof fetch {
  return (async (url: string, init?: RequestInit) => {
    seen.url = url;
    seen.init = init;
    return new Response(status === 204 ? null : body, { status });
  }) as unknown as typeof fetch;
}

test("client sends bearer token and JSON body", async () => {
  const seen: Seen = {};
  const client = createClient({
    baseUrl: "http://api.test/",
    getToken: () => "tok",
    fetchImpl: fakeFetch(200, '{"ok":true}', seen),
  });
  const res = await client.post<{ ok: boolean }>("/api/v1/x", { a: 1 });
  assert.deepEqual(res, { ok: true });
  assert.equal(seen.url, "http://api.test/api/v1/x");
  const headers = seen.init?.headers as Record<string, string>;
  assert.equal(headers.Authorization, "Bearer tok");
  assert.equal(seen.init?.body, '{"a":1}');
});

test("client handles 204 and omits Authorization when signed out", async () => {
  const seen: Seen = {};
  const client = createClient({ baseUrl: "http://api.test", getToken: () => null, fetchImpl: fakeFetch(204, "", seen) });
  assert.equal(await client.post("/api/v1/auth/magic-link", { email: "a@b.de" }), undefined);
  assert.equal((seen.init?.headers as Record<string, string>).Authorization, undefined);
});

test("client throws ApiError and reports 401 only for authenticated requests", async () => {
  let unauthorized = 0;
  const body = '{"error":{"code":"unauthorized","message":"sign in required"}}';
  const withToken = createClient({
    baseUrl: "http://api.test",
    getToken: () => "tok",
    onUnauthorized: () => unauthorized++,
    fetchImpl: fakeFetch(401, body),
  });
  await assert.rejects(withToken.get("/api/v1/me"), (e: unknown) => e instanceof ApiError && e.code === "unauthorized");
  assert.equal(unauthorized, 1);

  const anonymous = createClient({
    baseUrl: "http://api.test",
    getToken: () => null,
    onUnauthorized: () => unauthorized++,
    fetchImpl: fakeFetch(401, body),
  });
  await assert.rejects(anonymous.get("/api/v1/me"));
  assert.equal(unauthorized, 1);
});

test("client maps network failures to code network", async () => {
  const client = createClient({
    baseUrl: "http://api.test",
    getToken: () => null,
    fetchImpl: (async () => {
      throw new TypeError("Network request failed");
    }) as unknown as typeof fetch,
  });
  await assert.rejects(client.get("/healthz"), (e: unknown) => e instanceof ApiError && e.code === "network");
});
