import assert from "node:assert/strict";
import { test } from "node:test";

import {
  base64UrlToBytes,
  bytesToBase64Url,
  isBlockingReason,
  isIOSDevice,
  permissionReason,
  sameBytes,
  subscriptionBody,
  webPushCardState,
  webPushReasonMessage,
  webPushSupport,
  type WebPushEnv,
} from "./webpush-core.ts";

// A VAPID public key as the backend returns it (65 bytes, starts with 0x04).
const KEY = "BMJu7r0vEKyNWGCEDbQed7Pb0X0t0tMUM7aheK5PWwhkWsiK9p5-eM1L70Ut9Zr_NesnJCOVROrM-zw172dGWYw";

test("base64url round trip", () => {
  const bytes = base64UrlToBytes(KEY);
  assert.equal(bytes.length, 65);
  assert.equal(bytes[0], 4);
  assert.equal(bytesToBase64Url(bytes), KEY);
  const all = Uint8Array.from({ length: 256 }, (_, i) => i);
  assert.deepEqual([...base64UrlToBytes(bytesToBase64Url(all))], [...all]);
  assert.equal(bytesToBase64Url(new Uint8Array([251, 255, 254])), "-__-"); // the url-safe alphabet, no padding
  assert.equal(bytesToBase64Url(new Uint8Array([1])), "AQ");
});

test("base64url accepts padding and rejects garbage", () => {
  assert.deepEqual([...base64UrlToBytes("AQ==")], [1]);
  assert.deepEqual([...base64UrlToBytes("")], []);
  for (const bad of ["a+b/", "not base64", "A", "AQ=x", "AAAAA"]) {
    assert.throws(() => base64UrlToBytes(bad), bad);
  }
});

test("sameBytes", () => {
  assert.equal(sameBytes(new Uint8Array([1, 2]), new Uint8Array([1, 2])), true);
  assert.equal(sameBytes(new Uint8Array([1, 2]), new Uint8Array([1, 3])), false);
  assert.equal(sameBytes(new Uint8Array([1]), new Uint8Array([1, 2])), false);
  assert.equal(sameBytes(null, new Uint8Array([1])), false);
});

const supported: WebPushEnv = {
  isSecureContext: true,
  hasServiceWorker: true,
  hasPushManager: true,
  hasNotification: true,
  isIOS: false,
  isStandalone: false,
};

test("support detection", () => {
  assert.deepEqual(webPushSupport(supported), { ok: true });
  // iOS tab: even when the APIs seem to exist, the install comes first
  assert.deepEqual(webPushSupport({ ...supported, isIOS: true }), { ok: false, reason: "needs_install" });
  assert.deepEqual(webPushSupport({ ...supported, isIOS: true, hasPushManager: false }), { ok: false, reason: "needs_install" });
  // iOS home screen app
  assert.deepEqual(webPushSupport({ ...supported, isIOS: true, isStandalone: true }), { ok: true });
  // no Push API elsewhere, or no https
  assert.deepEqual(webPushSupport({ ...supported, hasPushManager: false }), { ok: false, reason: "unsupported" });
  assert.deepEqual(webPushSupport({ ...supported, hasServiceWorker: false }), { ok: false, reason: "unsupported" });
  assert.deepEqual(webPushSupport({ ...supported, hasNotification: false }), { ok: false, reason: "unsupported" });
  assert.deepEqual(webPushSupport({ ...supported, isSecureContext: false }), { ok: false, reason: "unsupported" });
});

test("iOS detection includes iPadOS", () => {
  assert.equal(isIOSDevice("Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) Safari/604.1", 5), true);
  assert.equal(isIOSDevice("Mozilla/5.0 (iPad; CPU OS 16_4 like Mac OS X)", 5), true);
  assert.equal(isIOSDevice("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15", 5), true); // iPadOS
  assert.equal(isIOSDevice("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Safari/605.1.15", 0), false); // a Mac
  assert.equal(isIOSDevice("Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/126", 5), false);
});

test("reasons map to German messages and to blocking", () => {
  assert.equal(webPushReasonMessage("needs_install"), "Zum Home-Bildschirm hinzufügen, dann Mitteilungen erlauben");
  for (const r of ["needs_install", "unsupported", "permission_denied", "not_configured", "error"] as const) {
    assert.ok(webPushReasonMessage(r).length > 10, r);
  }
  assert.equal(isBlockingReason("needs_install"), true);
  assert.equal(isBlockingReason("unsupported"), true);
  assert.equal(isBlockingReason("permission_denied"), false);
  assert.equal(isBlockingReason("error"), false);
  assert.equal(permissionReason("granted"), null);
  assert.equal(permissionReason("denied"), "permission_denied");
  assert.equal(permissionReason("default"), "permission_denied");
});

test("settings card state", () => {
  const base = { isWeb: true, consent: true as boolean | undefined, support: { ok: true } as const, permission: "default" };
  assert.deepEqual(webPushCardState(base), { kind: "enable" });
  assert.deepEqual(webPushCardState({ ...base, permission: "granted" }), { kind: "hidden" });
  assert.deepEqual(webPushCardState({ ...base, permission: "denied" }), { kind: "hint", reason: "permission_denied" });
  assert.deepEqual(webPushCardState({ ...base, support: { ok: false, reason: "needs_install" } }), {
    kind: "hint",
    reason: "needs_install",
  });
  // the consent card owns these cases
  assert.deepEqual(webPushCardState({ ...base, consent: false }), { kind: "hidden" });
  assert.deepEqual(webPushCardState({ ...base, consent: undefined }), { kind: "hidden" });
  assert.deepEqual(webPushCardState({ ...base, isWeb: false }), { kind: "hidden" });
});

test("subscriptionBody keeps only what the backend needs", () => {
  const json = { endpoint: "https://web.push.apple.com/x", expirationTime: null, keys: { p256dh: "k", auth: "a", extra: 1 } };
  assert.deepEqual(subscriptionBody(json), { endpoint: "https://web.push.apple.com/x", keys: { p256dh: "k", auth: "a" } });
  for (const bad of [null, "x", {}, { endpoint: "http://x/y", keys: { p256dh: "k", auth: "a" } }, { endpoint: "https://x/y" }, { endpoint: "https://x/y", keys: { p256dh: "k" } }, { endpoint: "https://x/y", keys: { p256dh: "", auth: "a" } }]) {
    assert.equal(subscriptionBody(bad), null, JSON.stringify(bad));
  }
});
