import { type UseQueryResult } from "@tanstack/react-query";
import { RefreshControl } from "react-native";

import { useColors } from "./theme";

type Refetchable = Pick<UseQueryResult, "isRefetching" | "refetch">;

/**
 * The pull-to-refresh control of a list screen: pulling refetches the given queries and the spinner
 * runs while any of them refetches. Pass it as `refreshControl` of `Screen`.
 *
 * @example
 * const refresh = usePullToRefresh([health, horse]);
 * <Screen back refreshControl={refresh}>
 */
export function usePullToRefresh(queries: readonly Refetchable[]) {
  const c = useColors();
  return (
    <RefreshControl
      refreshing={queries.some((q) => q.isRefetching)}
      onRefresh={() => void Promise.all(queries.map((q) => q.refetch()))}
      tintColor={c.primary.DEFAULT}
    />
  );
}
