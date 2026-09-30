import assert from "node:assert/strict";
import { test } from "node:test";

import { createSseParser } from "./sse.ts";

test("parses events and ignores comments", () => {
  const p = createSseParser();
  const out = p.push('retry: 3000\n: connected\n\nevent: presence.changed\ndata: {"a":1}\n\n: keep-alive\n\n');
  assert.deepEqual(out, [{ event: "presence.changed", data: '{"a":1}' }]);
});

test("handles chunks that split lines and events", () => {
  const p = createSseParser();
  assert.deepEqual(p.push("event: pres"), []);
  assert.deepEqual(p.push("ence.changed\nda"), []);
  assert.deepEqual(p.push('ta: {"x":2}\n'), []);
  assert.deepEqual(p.push("\n"), [{ event: "presence.changed", data: '{"x":2}' }]);
});

test("handles CRLF, also split between chunks", () => {
  const p = createSseParser();
  assert.deepEqual(p.push("event: a\r\ndata: 1\r"), []);
  assert.deepEqual(p.push("\n\r\n"), [{ event: "a", data: "1" }]);
});

test("joins multi-line data and defaults the event name", () => {
  const p = createSseParser();
  assert.deepEqual(p.push("data: one\ndata: two\n\n"), [{ event: "message", data: "one\ntwo" }]);
});

test("returns several events from one chunk and resets the event name", () => {
  const p = createSseParser();
  const out = p.push("event: a\ndata: 1\n\ndata: 2\n\n");
  assert.deepEqual(out, [
    { event: "a", data: "1" },
    { event: "message", data: "2" },
  ]);
});
