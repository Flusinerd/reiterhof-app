// Pure helpers for tapping a push notification (no React Native imports, unit-tested).
// The backend puts the target in the push `data`: `screen` (blankets, requests, training plan),
// `route` (health and reha reminders) or only ids (`request_id`, `observation_id`).

/** Internal routes a notification may open. Everything else is ignored. */
export const NOTIFICATION_ROUTE_PREFIXES = [
  "/blankets",
  "/horses/",
  "/requests/",
  "/observations/",
  "/reminders",
  "/training",
] as const;

// A path without scheme, host, backslashes, control characters or ".." segments.
const SAFE_PATH = /^\/[A-Za-z0-9._~\-/%=&+:@,]*(\?[A-Za-z0-9._~\-/%=&+:@,]*)?$/;

/** True when `path` is an internal route of the whitelist (prefix match on whole segments). */
export function isInternalRoute(path: unknown): path is string {
  if (typeof path !== "string" || path.length > 300) return false;
  if (!SAFE_PATH.test(path) || path.startsWith("//") || path.includes("://")) return false;
  const pathname = path.split("?")[0];
  if (pathname.split("/").some((segment) => segment === "..")) return false;
  return NOTIFICATION_ROUTE_PREFIXES.some((prefix) => {
    if (prefix.endsWith("/")) return pathname.startsWith(prefix) && pathname.length > prefix.length;
    return pathname === prefix || pathname.startsWith(`${prefix}/`);
  });
}

const ID = /^[A-Za-z0-9-]{1,64}$/;

/**
 * The route a notification with this `data` should open, or null. `screen` wins over `route`;
 * without either, a `request_id` or `observation_id` opens that item. Unknown or unsafe
 * targets give null (the app just opens as usual).
 */
export function notificationRoute(data: unknown): string | null {
  if (typeof data !== "object" || data === null) return null;
  const d = data as Record<string, unknown>;
  for (const key of ["screen", "route"]) {
    if (isInternalRoute(d[key])) return d[key] as string;
  }
  if (typeof d.request_id === "string" && ID.test(d.request_id)) return `/requests/${d.request_id}`;
  if (typeof d.observation_id === "string" && ID.test(d.observation_id)) return `/observations/${d.observation_id}`;
  return null;
}
