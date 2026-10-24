import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { requestKeys, requestsApi } from "./api/requests";
import { requestErrorMessage } from "./requests-errors";

/** "Mach ich" from a list: accepts a request, tracks which one is running and the German error text. */
export function useAcceptRequest() {
  const queryClient = useQueryClient();
  const [acceptingId, setAcceptingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const mutation = useMutation({
    mutationFn: (id: string) => requestsApi.accept(id),
    onMutate: (id) => {
      setError(null);
      setAcceptingId(id);
    },
    onError: (e) => setError(requestErrorMessage(e)),
    onSettled: () => {
      setAcceptingId(null);
      void queryClient.invalidateQueries({ queryKey: requestKeys.all });
    },
  });
  return { accept: (id: string) => mutation.mutate(id), acceptingId, error };
}
