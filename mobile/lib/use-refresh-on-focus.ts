import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { useFocusEffect } from "expo-router";
import { useCallback, useRef } from "react";

/**
 * Refetches the stale active queries under `queryKeys` every time the screen comes into focus again: back
 * from a pushed screen, a switch to its tab. Screens stay mounted below other screens and in tabs, so
 * without this they show what they showed when the user left (the mount that opened them has fetched
 * already, the first focus is skipped).
 *
 * "Stale" follows the default `staleTime` of the app (10 s): rapid tab switches do not hit the API, and
 * whatever a mutation or a realtime event invalidated is refetched at once anyway. Coming back from the
 * background is covered by the focus manager (app/_layout.tsx), which does the same for every active query.
 *
 * @example
 * useRefreshOnFocus([reminderKeys.all, blanketKeys.today]);
 */
export function useRefreshOnFocus(queryKeys: readonly QueryKey[]): void {
  const queryClient = useQueryClient();
  const first = useRef(true);
  // Callers pass a new array on every render; the content decides.
  const keys = JSON.stringify(queryKeys);
  useFocusEffect(
    useCallback(() => {
      if (first.current) {
        first.current = false;
        return;
      }
      for (const queryKey of JSON.parse(keys) as QueryKey[]) {
        void queryClient.refetchQueries({ queryKey, type: "active", stale: true });
      }
    }, [queryClient, keys]),
  );
}
