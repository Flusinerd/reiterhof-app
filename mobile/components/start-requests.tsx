import { useQuery } from "@tanstack/react-query";
import { router } from "expo-router";
import { useMemo } from "react";
import { ActivityIndicator, View } from "react-native";

import { RequestRow } from "@/components/request-row";
import { Button, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import type { ListParams } from "@/lib/requests";
import { toIsoDate } from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";
import { useColors } from "@/lib/theme";
import { useAcceptRequest } from "@/lib/use-accept-request";

/** The three newest open help requests on the "Stall" tab. */
const PARAMS: ListParams = { status: "open", limit: 3 };

/**
 * Open help requests as messages. Renders nothing at all when there are none: the section of the
 * tab keeps its heading and "Neu", an empty state belongs on the request list.
 */
export function StartRequests() {
  const c = useColors();
  const { user } = useAuth();
  const today = useMemo(() => toIsoDate(new Date()), []);
  const list = useQuery({ queryKey: requestKeys.list(PARAMS), queryFn: () => requestsApi.list(PARAMS) });
  const { accept, acceptingId, error } = useAcceptRequest();

  if (list.isPending) return <ActivityIndicator color={c.primary.DEFAULT} className="self-start" />;
  if (list.isError) {
    return (
      <View className="gap-3">
        <Text variant="body">{requestErrorMessage(list.error)}</Text>
        <Button label="Erneut versuchen" variant="outline" size="sm" onPress={() => void list.refetch()} />
      </View>
    );
  }
  const requests = list.data.requests;
  if (requests.length === 0) return null;
  return (
    <View>
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert" className="pb-2">
          {error}
        </Text>
      ) : null}
      {requests.map((r, i) => (
        <RequestRow
          key={r.id}
          request={r}
          today={today}
          userId={user?.id}
          last={i === requests.length - 1}
          accepting={acceptingId === r.id}
          onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
          onAccept={() => accept(r.id)}
        />
      ))}
    </View>
  );
}
