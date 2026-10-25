// The React side of the effect table (lib/query-effects.ts): a mutation that names what it changes.
// After a successful call the stale queries are refetched and the mutation stays pending until they
// have, so a button's spinner lasts until the screen shows the new data (and not one tap longer or
// shorter): the row the user tapped does not look unchanged for a moment, inviting a second tap.

import { useMutation, useQueryClient, type DefaultError, type QueryClient, type UseMutationOptions } from "@tanstack/react-query";

import { effectPrunedKeys, effectRemovedKeys, keysOf, withoutHorse, type Effect } from "../query-effects.ts";

/**
 * What a deleted horse leaves behind: it drops out of the cached lists first (the screens that show them
 * re-render without it, e.g. the horse tab named after it), then, a tick later, its own queries are removed.
 * Removing them while a screen still observes them would let that screen refetch into a 404.
 */
async function forgetHorses(client: QueryClient, effects: readonly Effect[]): Promise<void> {
  const deleted = effects.filter((e) => e.type === "horseDeleted");
  if (deleted.length === 0) return;
  for (const e of deleted) {
    for (const queryKey of effectPrunedKeys(e)) {
      client.setQueryData(queryKey, (old: unknown) => withoutHorse(old, (e as { horseId: string }).horseId));
    }
  }
  await new Promise((resolve) => setTimeout(resolve, 0));
  for (const e of deleted) {
    for (const queryKey of effectRemovedKeys(e)) {
      await client.cancelQueries({ queryKey });
      client.removeQueries({ queryKey });
    }
  }
}

/** Marks the keys of the effects stale; resolves when the active queries among them have refetched. */
export async function invalidateEffects(client: QueryClient, effects: readonly Effect[]): Promise<void> {
  await forgetHorses(client, effects);
  await Promise.all(keysOf(effects).map((queryKey) => client.invalidateQueries({ queryKey })));
}

/**
 * Refreshes after a write. A failure to refresh must not turn the write into an error (the data
 * is saved; the screens catch up on focus or with the next realtime event).
 */
async function refresh(client: QueryClient, effects: readonly Effect[]): Promise<void> {
  try {
    await invalidateEffects(client, effects);
  } catch {
    // see above
  }
}

export type EffectMutationOptions<TVars, TData, TContext> = UseMutationOptions<TData, DefaultError, TVars, TContext> & {
  /** What a successful call changed. */
  effects: (vars: TVars, data: TData) => Effect[];
  /** What a failed call may still have changed (a plan applied day by day, a conflict with a change of someone else). */
  errorEffects?: (vars: TVars, error: DefaultError) => Effect[];
};

/**
 * `useMutation` with the effects of the call. `onSuccess` of the options runs first (write the response into
 * the cache there), then the effects are invalidated, and the mutation is pending until that is done.
 *
 * @example
 * const remove = useEffectMutation({
 *   mutationFn: (id: string) => horsesApi.removeDocument(horseId, id),
 *   effects: () => [{ type: "documents", horseId }],
 * });
 */
export function useEffectMutation<TVars = void, TData = unknown, TContext = unknown>(
  options: EffectMutationOptions<TVars, TData, TContext>,
) {
  const client = useQueryClient();
  const { effects, errorEffects, onSuccess, onError, ...rest } = options;
  return useMutation<TData, DefaultError, TVars, TContext>({
    ...rest,
    onSuccess: async (data, vars, onMutateResult, context) => {
      await onSuccess?.(data, vars, onMutateResult, context);
      await refresh(client, effects(vars, data));
    },
    onError: async (error, vars, onMutateResult, context) => {
      await onError?.(error, vars, onMutateResult, context);
      if (errorEffects) await refresh(client, errorEffects(vars, error));
    },
  });
}
