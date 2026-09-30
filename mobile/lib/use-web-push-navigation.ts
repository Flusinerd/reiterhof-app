import { router, type Href } from "expo-router";
import { useEffect, useRef } from "react";
import { Platform } from "react-native";

import { useAuth } from "./auth";
import { isInternalRoute } from "./notifications-core.ts";
import { NAVIGATE_MESSAGE } from "./webpush-core.ts";

/**
 * Web only (no-op on native, where `useNotificationNavigation` handles taps): opens the screen
 * of a tapped web push. The service worker (public/sw.js) focuses the running app and posts
 * `{type: "stallfunk:navigate", route}`; with no app window open it opens the route as URL
 * instead, which the router handles on its own. The route is checked again against the
 * whitelist here. A message before sign-in waits for it. Render once, next to `useDeviceSetup`.
 */
export function useWebPushNavigation(): void {
  const { status, hasStable } = useAuth();
  const ready = status === "signedIn" && hasStable;
  const readyRef = useRef(ready);
  readyRef.current = ready;
  const pending = useRef<string | null>(null);
  const signedOut = status === "signedOut";

  useEffect(() => {
    if (Platform.OS !== "web" || typeof navigator === "undefined" || !("serviceWorker" in navigator)) return;
    const onMessage = (event: MessageEvent) => {
      const data: unknown = event.data;
      if (typeof data !== "object" || data === null) return;
      const { type, route } = data as { type?: unknown; route?: unknown };
      if (type !== NAVIGATE_MESSAGE || !isInternalRoute(route)) return;
      if (readyRef.current) router.push(route as Href);
      else pending.current = route;
    };
    navigator.serviceWorker.addEventListener("message", onMessage);
    return () => navigator.serviceWorker.removeEventListener("message", onMessage);
  }, []);

  useEffect(() => {
    if (ready && pending.current) {
      const route = pending.current;
      pending.current = null;
      router.push(route as Href);
    } else if (signedOut) {
      pending.current = null;
    }
  }, [ready, signedOut]);
}
