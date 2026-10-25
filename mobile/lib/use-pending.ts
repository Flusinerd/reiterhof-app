import { useMutationState, type MutationKey } from "@tanstack/react-query";
import { useMemo, useRef } from "react";

import { createSubmitOnce } from "./submit-once.ts";

/**
 * The variables of every running mutation with this key, from any component: the horse page and the
 * health screen see the same pending "Erledigt". Give every hook that can trigger the same write the same
 * `mutationKey` (lib/query-keys.ts, `mutationKeys`).
 */
export function usePendingVariables<TVars>(mutationKey: MutationKey): TVars[] {
  return useMutationState({
    filters: { mutationKey, status: "pending" },
    select: (mutation) => mutation.state.variables as TVars,
  });
}

/**
 * The ids whose write is running right now. A list shows a spinner on exactly those rows: tapping a second
 * row while the first one is in flight neither takes the first one's spinner away nor re-enables its buttons.
 *
 * @example
 * const dismissing = usePendingIds(mutationKeys.reminderDismiss, (item: ReminderItem) => item.id);
 * <ReminderRow dismissing={dismissing.has(item.id)} />
 */
export function usePendingIds<TVars>(mutationKey: MutationKey, idOf: (vars: TVars) => string | undefined): ReadonlySet<string> {
  const vars = usePendingVariables<TVars>(mutationKey);
  const ids = vars.map(idOf).filter((id): id is string => !!id);
  const key = ids.join("|");
  // The joined ids are the identity of the set.
  return useMemo(() => new Set(ids), [key]);
}

/**
 * `run(id, call)` starts the write unless that id is running or finished less than `cooldownMs` ago (see
 * createSubmitOnce); an ignored call resolves with `undefined`.
 */
export function useSubmitOnce(cooldownMs = 1500) {
  const once = useRef<ReturnType<typeof createSubmitOnce>>(null);
  once.current ??= createSubmitOnce(cooldownMs);
  return once.current;
}
