import assert from "node:assert/strict";
import { test } from "node:test";

import { isInternalRoute, notificationRoute } from "./notifications-core.ts";

const ID = "00000000-0000-4000-8000-000000000301";

test("routes of the whitelist are accepted", () => {
  for (const path of [
    "/blankets",
    "/blankets/history/" + ID,
    `/horses/${ID}/health`,
    `/horses/${ID}/reha`,
    `/horses/${ID}/blanket-plan`,
    `/requests/${ID}`,
    "/requests/new?type=appointment_companion&horse=" + ID,
    `/observations/${ID}`,
    "/reminders",
    "/training",
    "/training/week",
  ]) {
    assert.equal(isInternalRoute(path), true, path);
  }
});

test("everything else is ignored", () => {
  for (const path of [
    "",
    "/",
    "/settings",
    "/settings/privacy",
    "/horses/",
    "/requests/",
    "/observations/",
    "/blanketsX",
    "/training-evil",
    "/remindersX",
    "blankets",
    "//evil.example/blankets",
    "https://evil.example/blankets",
    "stallfunk://blankets",
    "javascript:alert(1)",
    "/blankets/../settings",
    "/horses/../../etc",
    "/blankets\\..\\x",
    "/blankets?next=https://evil.example",
    "/blankets#frag",
    "/blankets\n",
    "/blankets " + "a".repeat(400),
  ]) {
    assert.equal(isInternalRoute(path), false, JSON.stringify(path));
  }
  for (const value of [undefined, null, 42, {}, ["/blankets"]]) {
    assert.equal(isInternalRoute(value), false);
  }
});

test("notificationRoute reads screen, then route, then ids", () => {
  assert.equal(notificationRoute({ screen: "/blankets" }), "/blankets");
  assert.equal(notificationRoute({ route: `/horses/${ID}/health`, horse_id: ID }), `/horses/${ID}/health`);
  assert.equal(notificationRoute({ screen: "/blankets", route: "/reminders" }), "/blankets");
  // an unknown screen falls through to the next source
  assert.equal(notificationRoute({ screen: "/settings", route: "/reminders" }), "/reminders");
  assert.equal(notificationRoute({ request_id: ID }), `/requests/${ID}`);
  assert.equal(notificationRoute({ observation_id: ID, horse_id: ID, urgency: "urgent" }), `/observations/${ID}`);
});

test("notificationRoute ignores unknown or unsafe data", () => {
  assert.equal(notificationRoute(undefined), null);
  assert.equal(notificationRoute(null), null);
  assert.equal(notificationRoute("/blankets"), null);
  assert.equal(notificationRoute({}), null);
  assert.equal(notificationRoute({ kind: "helper" }), null);
  assert.equal(notificationRoute({ screen: "https://evil.example" }), null);
  assert.equal(notificationRoute({ screen: 42 }), null);
  assert.equal(notificationRoute({ request_id: "../settings" }), null);
  assert.equal(notificationRoute({ request_id: 7 }), null);
  assert.equal(notificationRoute({ observation_id: "x/y" }), null);
});
