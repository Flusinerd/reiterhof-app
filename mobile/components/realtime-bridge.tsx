import { useRefreshOnPush } from "@/lib/notifications";
import { useRealtimeInvalidation } from "@/lib/use-realtime-invalidation";

/**
 * Keeps the cached data current while somebody is signed in to a stable: realtime events and pushes
 * that arrive while the app is open mark what they changed as stale. Renders nothing; the root
 * layout mounts it once (hooks cannot be conditional, so it is a component).
 */
export function RealtimeBridge(): null {
  useRealtimeInvalidation();
  useRefreshOnPush();
  return null;
}
