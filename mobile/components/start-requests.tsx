import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { router } from "expo-router";
import { useState } from "react";
import { ActivityIndicator } from "react-native";

import { RequestCard } from "@/components/request-card";
import { Button, Card, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import type { ListParams } from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";
import { useInvalidateOnEvents } from "@/lib/realtime";

/** The two newest open help requests on the start screen (JAN-35). */
const PARAMS: ListParams = { status: "open", limit: 2 };

export function StartRequests() {
  const queryClient = useQueryClient();
  const list = useQuery({ queryKey: requestKeys.list(PARAMS), queryFn: () => requestsApi.list(PARAMS) });
  useInvalidateOnEvents({ "request.changed": [requestKeys.all] });
  const [acceptingId, setAcceptingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const accept = useMutation({
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

  if (list.isPending) return <ActivityIndicator />;
  if (list.isError) {
    return (
      <Card className="gap-3">
        <Text variant="body">{requestErrorMessage(list.error)}</Text>
        <Button label="Erneut versuchen" variant="outline" onPress={() => void list.refetch()} />
      </Card>
    );
  }
  if (list.data.requests.length === 0) {
    return (
      <Card>
        <Text variant="secondary">Keine offenen Anfragen.</Text>
      </Card>
    );
  }
  return (
    <>
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      {list.data.requests.map((r) => (
        <RequestCard
          key={r.id}
          request={r}
          accepting={acceptingId === r.id}
          onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
          onAccept={() => accept.mutate(r.id)}
        />
      ))}
    </>
  );
}
