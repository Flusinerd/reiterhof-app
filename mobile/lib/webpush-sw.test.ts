import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import vm from "node:vm";

import { NOTIFICATION_ROUTE_PREFIXES, notificationRoute } from "./notifications-core.ts";
import { NAVIGATE_MESSAGE } from "./webpush-core.ts";

// Loads public/sw.js into a sandbox with a fake service worker scope, so the whitelist copy in the
// worker is checked by behavior against lib/notifications-core.ts.

type Handler = (event: unknown) => void;
type Shown = { title: string; options: { body: string; data: unknown; icon: string; tag: string } };

function loadWorker(windows: { url: string; focused: boolean; messages: unknown[] }[] = []) {
  const source = readFileSync(new URL("../public/sw.js", import.meta.url), "utf8");
  const handlers = new Map<string, Handler>();
  const shown: Shown[] = [];
  const opened: string[] = [];
  const self = {
    location: { origin: "https://app.example.org" },
    addEventListener: (type: string, fn: Handler) => handlers.set(type, fn),
    skipWaiting: () => {},
    registration: {
      showNotification: async (title: string, options: Shown["options"]) => void shown.push({ title, options }),
    },
    clients: {
      claim: async () => {},
      matchAll: async () =>
        windows.map((w) => ({
          url: w.url,
          focus: async () => void (w.focused = true),
          postMessage: (m: unknown) => void w.messages.push(m),
        })),
      openWindow: async (url: string) => void opened.push(url),
    },
  };
  vm.runInNewContext(source, { self, URL, Promise, JSON, Error });
  const run = async (type: string, event: Record<string, unknown>) => {
    let pending: Promise<unknown> = Promise.resolve();
    handlers.get(type)?.({ ...event, waitUntil: (p: Promise<unknown>) => void (pending = p) });
    await pending;
  };
  return { handlers, shown, opened, run };
}

const ID = "00000000-0000-4000-8000-000000000301";

const samples: unknown[] = [
  { screen: "/blankets" },
  { screen: `/requests/${ID}`, kind: "new_request" },
  { route: `/horses/${ID}/health` },
  { screen: "/settings", route: "/reminders" },
  { request_id: ID },
  { observation_id: ID, horse_id: ID },
  { screen: "https://evil.example" },
  { screen: "//evil.example/blankets" },
  { screen: "/blankets/../settings" },
  { screen: "/settings" },
  { screen: "javascript:alert(1)" },
  { request_id: "../x" },
  { kind: "helper" },
  {},
  null,
  "/blankets",
];

test("tapping a notification opens the same routes as lib/notifications-core.ts", async () => {
  for (const data of samples) {
    const want = notificationRoute(data);
    const w = loadWorker();
    await w.run("notificationclick", { notification: { data, close() {} } });
    assert.deepEqual(w.opened, [want ?? "/"], JSON.stringify(data));
  }
});

test("every whitelisted prefix is accepted by the worker", async () => {
  for (const prefix of NOTIFICATION_ROUTE_PREFIXES) {
    const path = prefix.endsWith("/") ? `${prefix}${ID}` : prefix;
    const w = loadWorker();
    await w.run("notificationclick", { notification: { data: { screen: path }, close() {} } });
    assert.deepEqual(w.opened, [path], prefix);
  }
});

test("a tap focuses the running app and messages the route instead of opening a window", async () => {
  const app = { url: "https://app.example.org/requests/x", focused: false, messages: [] as unknown[] };
  const other = { url: "https://evil.example/", focused: false, messages: [] as unknown[] };
  const w = loadWorker([other, app]);
  let closed = false;
  await w.run("notificationclick", { notification: { data: { screen: `/requests/${ID}` }, close: () => (closed = true) } });
  assert.equal(closed, true);
  assert.equal(app.focused, true);
  assert.equal(other.focused, false);
  // objects from the sandbox have another prototype, so compare as JSON
  assert.equal(JSON.stringify(app.messages), JSON.stringify([{ type: NAVIGATE_MESSAGE, route: `/requests/${ID}` }]));
  assert.deepEqual(w.opened, []);
  // an unsafe target only focuses
  const app2 = { url: "https://app.example.org/", focused: false, messages: [] as unknown[] };
  await loadWorker([app2]).run("notificationclick", { notification: { data: { screen: "https://evil.example" }, close() {} } });
  assert.equal(app2.focused, true);
  assert.deepEqual(app2.messages, []);
});

test("a push always shows a notification from the backend payload", async () => {
  const w = loadWorker();
  const payload = { title: "Neue Anfrage", body: "Jemand sucht Hilfe", data: { screen: `/requests/${ID}`, kind: "new_request", request_id: ID } };
  await w.run("push", { data: { json: () => payload } });
  assert.equal(w.shown.length, 1);
  assert.equal(w.shown[0].title, "Neue Anfrage");
  assert.equal(w.shown[0].options.body, "Jemand sucht Hilfe");
  assert.equal(w.shown[0].options.icon, "/icon-192.png");
  assert.equal(w.shown[0].options.tag, `new_request:${ID}`);
  assert.equal(JSON.stringify(w.shown[0].options.data), JSON.stringify(payload.data));

  // Safari revokes subscriptions that push without showing anything, so bad payloads still show.
  for (const event of [{ data: null }, { data: { json: () => { throw new Error("bad json"); } } }, { data: { json: () => ({}) } }]) {
    const x = loadWorker();
    await x.run("push", event);
    assert.equal(x.shown.length, 1);
    assert.equal(x.shown[0].title, "Stallfunk");
  }
});
