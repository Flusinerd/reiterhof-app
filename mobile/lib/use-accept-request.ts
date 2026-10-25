import { useState } from "react";

import { useEffectMutation } from "./api/mutation";
import { requestsApi } from "./api/requests";
import { mutationKeys } from "./query-keys.ts";
import { requestErrorMessage } from "./requests-errors";

/** "Mach ich" from a list: accepts a request, tracks which one is running and the German error text. */
export function useAcceptRequest() {
  const [acceptingId, setAcceptingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const mutation = useEffectMutation({
    mutationFn: (id: string) => requestsApi.accept(id),
    mutationKey: mutationKeys.requestAccept,
    onMutate: (id) => {
      setError(null);
      setAcceptingId(id);
    },
    onError: (e) => setError(requestErrorMessage(e)),
    // The spinner lasts until the lists show the change (the effect is awaited before this runs).
    onSettled: () => setAcceptingId(null),
    // The helper duty also shows up in the reminder center; a failure (the request got full) refreshes too.
    effects: () => [{ type: "request" }],
    errorEffects: () => [{ type: "request" }],
  });
  return { accept: (id: string) => mutation.mutate(id), acceptingId, error };
}
