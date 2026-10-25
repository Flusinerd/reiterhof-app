import { useState } from "react";

import { useEffectMutation } from "./api/mutation";
import { requestsApi } from "./api/requests";
import { mutationKeys } from "./query-keys.ts";
import { requestErrorMessage } from "./requests-errors";
import { usePendingIds } from "./use-pending";

/**
 * "Mach ich" from a list: accepts a request, tracks which ones are running (any number at once, each row
 * keeps its own spinner until the lists show the change) and the German error text.
 */
export function useAcceptRequest() {
  const [error, setError] = useState<string | null>(null);
  const accepting = usePendingIds(mutationKeys.requestAccept, (id: string) => id);
  const mutation = useEffectMutation({
    mutationFn: (id: string) => requestsApi.accept(id),
    mutationKey: mutationKeys.requestAccept,
    onMutate: () => setError(null),
    onError: (e) => setError(requestErrorMessage(e)),
    // The helper duty also shows up in the reminder center; a failure (the request got full) refreshes too.
    effects: () => [{ type: "request" }],
    errorEffects: () => [{ type: "request" }],
  });
  // The promise keeps the button busy; the error is shown by the screen, not thrown into the press handler.
  return { accept: (id: string) => mutation.mutateAsync(id).catch(() => undefined), accepting, error };
}
