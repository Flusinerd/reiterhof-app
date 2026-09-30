import { Platform } from "react-native";

import { ApiError, authed } from "./api";
import { registerServiceWorker } from "./service-worker";
import {
  base64UrlToBytes,
  isIOSDevice,
  permissionReason,
  sameBytes,
  subscriptionBody,
  webPushSupport,
  type WebPushEnv,
  type WebPushReason,
  type WebPushSupport,
} from "./webpush-core.ts";

// Web Push for the PWA (JAN-74). Only does anything on Platform.OS === "web"; native keeps
// using native push (lib/push.ts). Never throws; problems come back as a `reason`.

export type WebPushResult = { ok: true } | { ok: false; reason: WebPushReason };

/** What this browser offers, read from `window` (all false outside the browser). */
export function webPushEnv(): WebPushEnv {
  if (Platform.OS !== "web" || typeof window === "undefined" || typeof navigator === "undefined") {
    return {
      isSecureContext: false,
      hasServiceWorker: false,
      hasPushManager: false,
      hasNotification: false,
      isIOS: false,
      isStandalone: false,
    };
  }
  const nav = navigator as Navigator & { standalone?: boolean };
  return {
    isSecureContext: window.isSecureContext === true,
    hasServiceWorker: "serviceWorker" in navigator,
    hasPushManager: "PushManager" in window,
    hasNotification: "Notification" in window,
    isIOS: isIOSDevice(navigator.userAgent, navigator.maxTouchPoints ?? 0),
    isStandalone: nav.standalone === true || window.matchMedia?.("(display-mode: standalone)").matches === true,
  };
}

/** Whether this page can receive pushes at all (on iOS: only when added to the home screen). */
export function webPushSupportNow(): WebPushSupport {
  return webPushSupport(webPushEnv());
}

/** `Notification.permission`, or "unsupported" where there is no Notification API. */
export function webPushPermission(): NotificationPermission | "unsupported" {
  if (Platform.OS !== "web" || typeof Notification === "undefined") return "unsupported";
  return Notification.permission;
}

async function fetchPublicKey(): Promise<Uint8Array<ArrayBuffer> | WebPushReason> {
  try {
    const { public_key } = await authed.get<{ public_key: string }>("/api/v1/push/web/public-key");
    return base64UrlToBytes(public_key);
  } catch (e) {
    return e instanceof ApiError && e.status === 503 ? "not_configured" : "error";
  }
}

/** Subscribes (or re-subscribes after a key change) and sends the subscription to the backend. */
async function subscribeAndRegister(): Promise<WebPushResult> {
  const registration = await registerServiceWorker();
  if (!registration) return { ok: false, reason: "unsupported" };
  await navigator.serviceWorker.ready;
  const key = await fetchPublicKey();
  if (typeof key === "string") return { ok: false, reason: key };

  let subscription = await registration.pushManager.getSubscription();
  if (subscription && !sameBytes(subscription.options.applicationServerKey ? new Uint8Array(subscription.options.applicationServerKey) : null, key)) {
    // The server's VAPID key changed; the old subscription can never receive again.
    await subscription.unsubscribe();
    subscription = null;
  }
  subscription ??= await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key });
  const body = subscriptionBody(subscription.toJSON());
  if (!body) return { ok: false, reason: "error" };
  await authed.post<void>("/api/v1/me/web-push-subscriptions", body);
  return { ok: true };
}

/**
 * Asks for notification permission and subscribes this browser. Call it from a tap handler and
 * before the first `await` there: iOS only shows the permission prompt for a user gesture, and
 * the request is the first thing this function does. Without support (iOS tab, old browser) it
 * returns `needs_install` / `unsupported` so the UI can show `webPushReasonMessage(reason)`.
 */
export async function enableWebPush(): Promise<WebPushResult> {
  if (Platform.OS !== "web") return { ok: false, reason: "unsupported" };
  const support = webPushSupportNow();
  if (!support.ok) return { ok: false, reason: support.reason };
  try {
    const permission = Notification.permission === "default" ? await Notification.requestPermission() : Notification.permission;
    const reason = permissionReason(permission);
    if (reason) return { ok: false, reason };
    return await subscribeAndRegister();
  } catch {
    return { ok: false, reason: "error" };
  }
}

/**
 * Refreshes the subscription without asking: does nothing unless the permission is already
 * granted. Run at app start (`useDeviceSetup`) so a changed endpoint or key reaches the backend.
 */
export async function syncWebPush(): Promise<WebPushResult> {
  if (Platform.OS !== "web") return { ok: false, reason: "unsupported" };
  const support = webPushSupportNow();
  if (!support.ok) return { ok: false, reason: support.reason };
  if (Notification.permission !== "granted") return { ok: false, reason: "permission_denied" };
  try {
    return await subscribeAndRegister();
  } catch {
    return { ok: false, reason: "error" };
  }
}

/** Removes this browser's subscription from the push service and the backend (push consent withdrawn). */
export async function disableWebPush(): Promise<void> {
  if (Platform.OS !== "web" || !webPushSupportNow().ok) return;
  try {
    const registration = await navigator.serviceWorker.getRegistration("/");
    const subscription = await registration?.pushManager.getSubscription();
    if (!subscription) return;
    await subscription.unsubscribe();
    await authed.delete<void>("/api/v1/me/web-push-subscriptions", { endpoint: subscription.endpoint });
  } catch {
    // the server drops the subscription with the consent anyway
  }
}
