import { useQueryClient } from "@tanstack/react-query";
import * as Notifications from "expo-notifications";
import { router, type Href } from "expo-router";
import { useEffect, useRef } from "react";

import { useAuth } from "./auth";
import { notificationRoute } from "./notifications-core.ts";
import { pushKeys } from "./query-effects.ts";

// Foreground presentation: show the notification as a banner and in the list. The sound is on
// because Android does not drop the banner down without it. Without this handler notifications
// that arrive while the app is open are not shown at all.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

/** Two responses for the same notification within this time count as one tap. */
const DUPLICATE_WINDOW_MS = 3000;

/**
 * Opens the screen of a tapped push notification (JAN-18). Covers the tap while the app is
 * running or in the background (listener) and the tap that started the app
 * (`getLastNotificationResponse`). The target comes from `data.screen`/`data.route` and is opened
 * only when it is a known internal route (`notificationRoute`); anything else is ignored. A tap
 * before the user is signed in (with a stable) waits for it, and is dropped when nobody signs in.
 * Render it once, in the root layout.
 */
export function useNotificationNavigation(): void {
  const { status, hasStable } = useAuth();
  const ready = status === "signedIn" && hasStable;
  const pending = useRef<string | null>(null);
  const last = useRef<{ key: string; at: number } | null>(null);
  const readyRef = useRef(ready);
  readyRef.current = ready;
  const signedOut = status === "signedOut";

  useEffect(() => {
    const open = (route: string) => {
      // Let the navigator finish mounting (cold start) before pushing.
      setTimeout(() => router.push(route as Href), 0);
    };

    const handle = (response: Notifications.NotificationResponse | null) => {
      if (!response || response.actionIdentifier !== Notifications.DEFAULT_ACTION_IDENTIFIER) return;
      const key = response.notification.request.identifier;
      const now = Date.now();
      if (last.current && last.current.key === key && now - last.current.at < DUPLICATE_WINDOW_MS) return;
      last.current = { key, at: now };
      Notifications.clearLastNotificationResponse();
      const route = notificationRoute(response.notification.request.content.data);
      if (!route) return;
      if (readyRef.current) open(route);
      else pending.current = route;
    };

    handle(Notifications.getLastNotificationResponse()); // cold start
    const subscription = Notifications.addNotificationResponseReceivedListener(handle);
    return () => subscription.remove();
  }, []);

  useEffect(() => {
    if (ready && pending.current) {
      const route = pending.current;
      pending.current = null;
      setTimeout(() => router.push(route as Href), 0);
    } else if (signedOut) {
      pending.current = null;
    }
  }, [ready, signedOut]);
}

/**
 * A push that arrives while the app is open (shown as a banner) means something changed: the reminder
 * center lists the pushes that were sent, and a helper, observation or weather push changes the data
 * behind it too. Marks those queries stale (`pushKeys`). Render it once, while signed in.
 */
export function useRefreshOnPush(): void {
  const queryClient = useQueryClient();
  useEffect(() => {
    const subscription = Notifications.addNotificationReceivedListener((notification) => {
      const kind = (notification.request.content.data as { kind?: unknown } | null | undefined)?.kind;
      for (const queryKey of pushKeys(kind)) void queryClient.invalidateQueries({ queryKey });
    });
    return () => subscription.remove();
  }, [queryClient]);
}
