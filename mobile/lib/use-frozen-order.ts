import { useFocusEffect } from "expo-router";
import { useCallback, useRef } from "react";

import { freezeOrder, inOrder } from "./health-done.ts";

/**
 * The items in the order they had when the user opened the screen, however the server sorts them
 * afterwards: a row that was acted on (a due date that moved) stays under the finger instead of jumping
 * away. New items appear after their predecessor, deleted ones drop out. Leaving the screen starts the
 * next visit with a fresh sort.
 */
export function useFrozenOrder<T extends { id: string }>(items: readonly T[]): T[] {
  const order = useRef<string[]>([]);
  useFocusEffect(
    useCallback(() => {
      return () => {
        order.current = [];
      };
    }, []),
  );
  order.current = freezeOrder(
    order.current,
    items.map((i) => i.id),
  );
  return inOrder(items, order.current);
}
