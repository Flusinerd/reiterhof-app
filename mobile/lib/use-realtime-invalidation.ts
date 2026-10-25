import { useQueryClient } from "@tanstack/react-query";

import { eventKeys, REALTIME_EVENT_TYPES } from "./query-effects.ts";
import { useStableEvents } from "./realtime";

/**
 * The one subscriber to the realtime stream of the app: every event marks the queries it changed as
 * stale (lib/query-effects.ts, `eventKeys`, scoped by the horse in the event), a `resync` after a
 * reconnect marks everything stale because events may have been missed. Mounted once, in
 * `RealtimeBridge`; screens do not subscribe themselves, so a screen that was never opened (or is not
 * on top) is fresh when it shows up.
 */
export function useRealtimeInvalidation(): void {
  const queryClient = useQueryClient();
  useStableEvents(REALTIME_EVENT_TYPES, (event) => {
    const keys = eventKeys(event.type, event.data);
    if (keys === "all") {
      void queryClient.invalidateQueries();
      return;
    }
    for (const queryKey of keys) void queryClient.invalidateQueries({ queryKey });
  });
}
