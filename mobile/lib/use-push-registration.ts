import { useEffect, useRef } from "react";

import { authed } from "./api";
import { useAuth } from "./auth";
import { syncGeofence } from "./geofence";
import { registerForPush } from "./push";

/**
 * Once per app start and signed-in user: asks for notification permission and sends the
 * Expo push token to `POST /api/v1/me/push-tokens` (the backend upserts by token, so
 * repeating it is harmless). Failures are silent; without permission or a real device
 * nothing happens. Also re-arms the geofence if this device opted in.
 */
export function useDeviceSetup(): void {
  const { status, hasStable, user, me } = useAuth();
  const doneFor = useRef<string | null>(null);
  const userId = user?.id ?? null;
  const stable = me?.stable ?? null;

  useEffect(() => {
    if (status !== "signedIn" || !hasStable || !userId || doneFor.current === userId) return;
    doneFor.current = userId;
    void (async () => {
      try {
        const result = await registerForPush();
        if (result.ok) await authed.post("/api/v1/me/push-tokens", result.registration);
      } catch {
        // silent: a missing token only means no pushes on this device
      }
      await syncGeofence(stable);
    })();
  }, [status, hasStable, userId, stable]);
}
