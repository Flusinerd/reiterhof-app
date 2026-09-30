// Pure helpers for Web Push in the PWA (JAN-74). No React Native or DOM imports; unit-tested.

/** base64url (with or without padding) to bytes, e.g. the VAPID public key. Throws on bad input. */
export function base64UrlToBytes(input: string): Uint8Array<ArrayBuffer> {
  if (!/^[A-Za-z0-9_-]*={0,2}$/.test(input)) throw new Error("not base64url");
  const base64 = input.replace(/=+$/, "").replace(/-/g, "+").replace(/_/g, "/");
  if (base64.length % 4 === 1) throw new Error("not base64url");
  const binary = atob(base64 + "=".repeat((4 - (base64.length % 4)) % 4));
  const bytes = new Uint8Array(new ArrayBuffer(binary.length));
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes;
}

/** Bytes to unpadded base64url. */
export function bytesToBase64Url(bytes: Uint8Array): string {
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

/** True when both byte arrays hold the same bytes. */
export function sameBytes(a: ArrayLike<number> | null | undefined, b: ArrayLike<number> | null | undefined): boolean {
  if (!a || !b || a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

// --- support detection -----------------------------------------------------------------------

/** What the browser offers; `webPushEnv()` in webpush.ts reads it from `window`. */
export type WebPushEnv = {
  isSecureContext: boolean;
  hasServiceWorker: boolean;
  hasPushManager: boolean;
  hasNotification: boolean;
  /** iPhone or iPad (including iPadOS, which reports itself as a Mac). */
  isIOS: boolean;
  /** Running as an installed app (home screen), not in a browser tab. */
  isStandalone: boolean;
};

export type WebPushReason =
  /** iOS Safari tab: push only exists for apps added to the home screen. */
  | "needs_install"
  /** The browser has no Push API, or the page is not served over https. */
  | "unsupported"
  | "permission_denied"
  /** The server has no VAPID keys (503). */
  | "not_configured"
  | "error";

export type WebPushSupport = { ok: true } | { ok: false; reason: "needs_install" | "unsupported" };

/** Whether this page can subscribe to push at all. On iOS a tab always needs the install first. */
export function webPushSupport(env: WebPushEnv): WebPushSupport {
  if (env.isIOS && !env.isStandalone) return { ok: false, reason: "needs_install" };
  if (!env.isSecureContext || !env.hasServiceWorker || !env.hasPushManager || !env.hasNotification) {
    return { ok: false, reason: "unsupported" };
  }
  return { ok: true };
}

/** iPhone, iPad and iPod, also iPadOS 13+ which sends a Macintosh user agent but has a touch screen. */
export function isIOSDevice(userAgent: string, maxTouchPoints: number): boolean {
  if (/iPhone|iPad|iPod/.test(userAgent)) return true;
  return /Macintosh/.test(userAgent) && maxTouchPoints > 1;
}

/** German, user-facing text for a reason. */
export function webPushReasonMessage(reason: WebPushReason): string {
  switch (reason) {
    case "needs_install":
      return "Zum Home-Bildschirm hinzufügen, dann Mitteilungen erlauben";
    case "unsupported":
      return "Dieser Browser unterstützt keine Mitteilungen. Nutze aktuelles Safari, Chrome oder Firefox.";
    case "permission_denied":
      return "Mitteilungen sind blockiert. Erlaube sie in den Geräte- oder Browser-Einstellungen.";
    case "not_configured":
      return "Mitteilungen sind auf dem Server noch nicht eingerichtet.";
    case "error":
      return "Mitteilungen konnten nicht eingerichtet werden. Versuch es später noch mal.";
  }
}

/** Reasons where "Erlauben" cannot work on this device until something changes (the install). */
export function isBlockingReason(reason: WebPushReason): boolean {
  return reason === "needs_install" || reason === "unsupported";
}

/** Maps `Notification.permission` (after asking) to a result reason, or null when granted. */
export function permissionReason(permission: string): WebPushReason | null {
  if (permission === "granted") return null;
  // "default" after asking means the prompt was dismissed; treat it like a refusal for this attempt.
  return "permission_denied";
}

// --- the settings card -------------------------------------------------------------------------

export type WebPushCardState =
  | { kind: "hidden" }
  | { kind: "hint"; reason: WebPushReason }
  | { kind: "enable" };

/**
 * What the settings card shows on the web: nothing without the push consent (the consent card
 * handles that) or when everything is on; a hint when this device cannot receive pushes; a button
 * when the permission was not asked yet.
 */
export function webPushCardState(input: {
  isWeb: boolean;
  consent: boolean | undefined;
  support: WebPushSupport;
  permission: string;
}): WebPushCardState {
  if (!input.isWeb || input.consent !== true) return { kind: "hidden" };
  if (!input.support.ok) return { kind: "hint", reason: input.support.reason };
  if (input.permission === "denied") return { kind: "hint", reason: "permission_denied" };
  if (input.permission === "granted") return { kind: "hidden" };
  return { kind: "enable" };
}

// --- subscription body -----------------------------------------------------------------------

export type SubscriptionBody = { endpoint: string; keys: { p256dh: string; auth: string } };

/** Picks the fields the backend needs from `PushSubscription.toJSON()`; null when incomplete. */
export function subscriptionBody(json: unknown): SubscriptionBody | null {
  if (typeof json !== "object" || json === null) return null;
  const j = json as { endpoint?: unknown; keys?: { p256dh?: unknown; auth?: unknown } };
  const { endpoint, keys } = j;
  if (typeof endpoint !== "string" || !endpoint.startsWith("https://")) return null;
  if (!keys || typeof keys.p256dh !== "string" || typeof keys.auth !== "string") return null;
  if (!keys.p256dh || !keys.auth) return null;
  return { endpoint, keys: { p256dh: keys.p256dh, auth: keys.auth } };
}

/** The message the service worker posts to the page when a notification was tapped. */
export const NAVIGATE_MESSAGE = "stallfunk:navigate";
