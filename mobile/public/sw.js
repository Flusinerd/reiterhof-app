// Stallfunk service worker: Web Push (JAN-74). Served at /sw.js, registered by
// lib/service-worker.web.ts. Plain JS, no build step.
//
// It only handles `push` and `notificationclick`; there is no offline caching here. Caching
// can be added in a separate file and pulled in with importScripts("/sw-cache.js") at the top.
//
// Payload from the backend (internal/push/notifier.go): {title, body, data: {screen, kind, ...}}.

// Keep ROUTE_PREFIXES, SAFE_PATH, isInternalRoute and routeOf in sync with
// lib/notifications-core.ts (lib/webpush-sw.test.ts loads this file and compares behavior).
const ROUTE_PREFIXES = ["/blankets", "/horses/", "/requests/", "/observations/", "/reminders", "/training"];
const SAFE_PATH = /^\/[A-Za-z0-9._~\-/%=&+:@,]*(\?[A-Za-z0-9._~\-/%=&+:@,]*)?$/;
const ID = /^[A-Za-z0-9-]{1,64}$/;
// The message the page listens for (lib/use-web-push-navigation.ts).
const NAVIGATE_MESSAGE = "stallfunk:navigate";

function isInternalRoute(path) {
  if (typeof path !== "string" || path.length > 300) return false;
  if (!SAFE_PATH.test(path) || path.startsWith("//") || path.includes("://")) return false;
  const pathname = path.split("?")[0];
  if (pathname.split("/").some((segment) => segment === "..")) return false;
  return ROUTE_PREFIXES.some((prefix) => {
    if (prefix.endsWith("/")) return pathname.startsWith(prefix) && pathname.length > prefix.length;
    return pathname === prefix || pathname.startsWith(prefix + "/");
  });
}

function routeOf(data) {
  if (typeof data !== "object" || data === null) return null;
  for (const key of ["screen", "route"]) {
    if (isInternalRoute(data[key])) return data[key];
  }
  if (typeof data.request_id === "string" && ID.test(data.request_id)) return "/requests/" + data.request_id;
  if (typeof data.observation_id === "string" && ID.test(data.observation_id)) {
    return "/observations/" + data.observation_id;
  }
  return null;
}

self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch (e) {
    payload = {};
  }
  const data = typeof payload.data === "object" && payload.data !== null ? payload.data : {};
  const title = typeof payload.title === "string" && payload.title ? payload.title : "Stallfunk";
  // Same kind and item replace each other instead of piling up.
  const id = data.request_id || data.observation_id || data.horse_id || "";
  // Safari requires every push to show a notification, so this is unconditional.
  event.waitUntil(
    self.registration.showNotification(title, {
      body: typeof payload.body === "string" ? payload.body : "",
      data,
      icon: "/icon-192.png",
      tag: (data.kind || "stallfunk") + (id ? ":" + id : ""),
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const route = routeOf(event.notification.data);
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      for (const client of windows) {
        let sameOrigin = false;
        try {
          sameOrigin = new URL(client.url).origin === self.location.origin;
        } catch (e) {
          sameOrigin = false;
        }
        if (!sameOrigin) continue;
        try {
          await client.focus();
        } catch (e) {
          // focusing can be refused; the message below still reaches the page
        }
        if (route) client.postMessage({ type: NAVIGATE_MESSAGE, route });
        return;
      }
      await self.clients.openWindow(route || "/");
    })(),
  );
});
