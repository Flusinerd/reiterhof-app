import { useEffect, useRef } from "react";
import { Platform } from "react-native";

import { authed } from "./api";
import { useAuth } from "./auth";
import { useHasConsent } from "./consent";
import { syncGeofence } from "./geofence";
import { registerForPush } from "./push";
import { useWebPushNavigation } from "./use-web-push-navigation";
import { syncWebPush } from "./webpush";

/**
 * Once per app start and signed-in user: asks for notification permission and sends the
 * native device token to `POST /api/v1/me/push-tokens` (the backend upserts by token, so
 * repeating it is harmless). Only with the push consent (JAN-19; asked by ConsentOnboarding).
 * On the web (PWA) it refreshes the Web Push subscription instead, and only when the browser
 * permission is already granted. Failures are silent; without permission or a real device nothing happens. Also re-arms the geofence if this device opted in.
 */
export function useDeviceSetup(): void {
  useWebPushNavigation(); // web: tapping a web push opens its screen (no-op on native)
  const { status, hasStable, user, me } = useAuth();
  const doneFor = useRef<string | null>(null);
  const pushConsent = useHasConsent("push");
  const userId = user?.id ?? null;
  const stable = me?.stable ?? null;

  useEffect(() => {
    if (status !== "signedIn" || !hasStable || !userId || pushConsent === undefined) return;
    // Runs once per user and consent state, so granting the consent registers the token right away.
    const key = `${userId}:${pushConsent}`;
    if (doneFor.current === key) return;
    doneFor.current = key;
    void (async () => {
      if (pushConsent) {
        try {
          if (Platform.OS === "web") {
            // Web Push (JAN-74): never asks here (iOS needs a tap); `enableWebPush` does that from
            // the consent sheet and the settings card. This only refreshes an existing subscription.
            await syncWebPush();
          } else {
            const result = await registerForPush();
            if (result.ok) await authed.post("/api/v1/me/push-tokens", result.registration);
          }
        } catch {
          // silent: a missing token only means no pushes on this device
        }
      }
      await syncGeofence(stable);
    })();
  }, [status, hasStable, userId, stable, pushConsent]);
}
