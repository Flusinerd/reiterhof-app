import assert from "node:assert/strict";
import { test } from "node:test";

import { capabilitiesFor, DEFAULT_NATIVE_API_URL, resolveApiUrl } from "./platform-core.ts";

test("web has no geofence, no Apple sign-in, no background location, no device calendar", () => {
  const c = capabilitiesFor("web");
  assert.equal(c.geofence, false);
  assert.equal(c.appleSignIn, false);
  assert.equal(c.backgroundLocation, false);
  assert.equal(c.deviceCalendar, false);
  assert.equal(c.secureStorage, false);
  assert.equal(c.motionNeedsGesture, true);
});

test("ios keeps every native feature, android has no Apple sign-in", () => {
  const ios = capabilitiesFor("ios");
  assert.deepEqual(Object.values(ios).filter((v) => v === false).length, 1); // motionNeedsGesture
  assert.equal(ios.appleSignIn, true);
  assert.equal(capabilitiesFor("android").appleSignIn, false);
  assert.equal(capabilitiesFor("android").geofence, true);
});

test("web uses the origin of the page by default", () => {
  assert.equal(
    resolveApiUrl({ os: "web", configUrl: "https://api.stallfunk.de", origin: "https://stallfunk.de" }),
    "https://stallfunk.de",
  );
});

test("a build-time override wins on every platform, trailing slashes are removed", () => {
  assert.equal(
    resolveApiUrl({ os: "web", configUrl: "https://api.stallfunk.de", override: "http://localhost:8080/", origin: "http://localhost:8081" }),
    "http://localhost:8080",
  );
  assert.equal(resolveApiUrl({ os: "android", configUrl: "https://api.stallfunk.de", override: "http://10.0.2.2:8080" }), "http://10.0.2.2:8080");
});

test("native ignores the origin and uses the configured URL", () => {
  assert.equal(resolveApiUrl({ os: "ios", configUrl: "https://api.stallfunk.de/", origin: "https://x.test" }), "https://api.stallfunk.de");
  assert.equal(resolveApiUrl({ os: "ios" }), DEFAULT_NATIVE_API_URL);
});

test("web without a window falls back to the configured URL", () => {
  assert.equal(resolveApiUrl({ os: "web", configUrl: "https://api.stallfunk.de", origin: "" }), "https://api.stallfunk.de");
  assert.equal(resolveApiUrl({ os: "web", override: "  ", origin: null }), DEFAULT_NATIVE_API_URL);
});

test("non-string config values (expo config turns null into {}) are ignored", () => {
  assert.equal(resolveApiUrl({ os: "web", configUrl: "https://api.stallfunk.de", override: {}, origin: "https://stallfunk.de" }), "https://stallfunk.de");
  assert.equal(resolveApiUrl({ os: "ios", configUrl: "https://api.stallfunk.de", override: {} }), "https://api.stallfunk.de");
});
