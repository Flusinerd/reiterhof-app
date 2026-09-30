import assert from "node:assert/strict";
import { test } from "node:test";

import { actionFor, failureMessage, regionFor } from "./geofence-core.ts";

test("regionFor builds a region with the configured radius", () => {
  const r = regionFor({ lat: 51.66, lng: 6.96, geofence_radius_m: 150 });
  assert.deepEqual(r, {
    identifier: "reiterhof-stable",
    latitude: 51.66,
    longitude: 6.96,
    radius: 150,
    notifyOnEnter: true,
    notifyOnExit: true,
  });
});

test("regionFor clamps the radius and falls back to the default", () => {
  assert.equal(regionFor({ lat: 1, lng: 1, geofence_radius_m: 20 })?.radius, 100);
  assert.equal(regionFor({ lat: 1, lng: 1, geofence_radius_m: 999_999 })?.radius, 2000);
  assert.equal(regionFor({ lat: 1, lng: 1, geofence_radius_m: null })?.radius, 150);
  assert.equal(regionFor({ lat: 1, lng: 1, geofence_radius_m: 0 })?.radius, 150);
});

test("regionFor returns null without valid coordinates", () => {
  assert.equal(regionFor(null), null);
  assert.equal(regionFor({ lat: null, lng: 6.96, geofence_radius_m: 150 }), null);
  assert.equal(regionFor({ lat: 51.66, lng: null, geofence_radius_m: 150 }), null);
  assert.equal(regionFor({ lat: 91, lng: 6.96, geofence_radius_m: 150 }), null);
  assert.equal(regionFor({ lat: Number.NaN, lng: 6.96, geofence_radius_m: 150 }), null);
});

test("actionFor maps region events to API calls", () => {
  assert.equal(actionFor("enter"), "check-in");
  assert.equal(actionFor("exit"), "check-out");
  assert.equal(actionFor(null), null);
});

test("failure messages are German", () => {
  assert.match(failureMessage("background_denied"), /Immer/);
  assert.match(failureMessage("no_location"), /Standort/);
});
