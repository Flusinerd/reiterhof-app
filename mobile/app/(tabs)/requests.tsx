import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { Bell, CalendarDays, Plus } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, ScrollView, View } from "react-native";

import { RequestCard } from "@/components/request-card";
import { Button, Card, PageHeader, Pill, Screen, Section, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { colors } from "@/lib/theme";
import { REQUEST_FILTERS, filterParams, toIsoDate, type RequestFilter } from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";

const EMPTY_TEXT: Record<RequestFilter, string> = {
  open: "Keine offenen Anfragen.",
  mine: "Noch keine eigenen Anfragen.",
  helping: "Du hilfst bei keiner Anfrage.",
  done: "Noch nichts erledigt.",
};

export default function Requests() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<RequestFilter>("open");
  const [error, setError] = useState<string | null>(null);
  const [acceptingId, setAcceptingId] = useState<string | null>(null);

  const params = filterParams(filter, toIsoDate(new Date()));
  const list = useQuery({ queryKey: requestKeys.list(params), queryFn: () => requestsApi.list(params) });
  const thanks = useQuery({ queryKey: requestKeys.thanks, queryFn: requestsApi.myThanks });

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

  const open = list.data?.open_count ?? 0;
  const requests = list.data?.requests ?? [];

  return (
    <Screen
      refreshControl={
        <RefreshControl
          refreshing={list.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: requestKeys.all })}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <PageHeader
        title="Anfragen"
        action={
          <Button
            label="Neu"
            icon={Plus}
            size="sm"
            onPress={() => router.push("/requests/new")}
            accessibilityLabel="Neue Anfrage"
          />
        }
        value={list.data ? String(open) : "–"}
        valueSize="sm"
        unit={open === 1 ? "offene Anfrage" : "offene Anfragen"}
      />

      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2" className="-mx-6" contentContainerStyle={{ paddingHorizontal: 24 }}>
        {REQUEST_FILTERS.map((f) => (
          <Pill key={f.value} label={f.label} selected={filter === f.value} onPress={() => setFilter(f.value)} />
        ))}
      </ScrollView>

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <Section
        title={REQUEST_FILTERS.find((f) => f.value === filter)?.label ?? ""}
        action={
          <Button
            label="Mein Kalender"
            icon={CalendarDays}
            variant="ghost"
            size="sm"
            onPress={() => router.push("/requests/calendar")}
          />
        }
      >
        {list.isPending ? (
          <ActivityIndicator color={colors.primary.DEFAULT} />
        ) : list.isError ? (
          <Card className="gap-3">
            <Text variant="body">{requestErrorMessage(list.error)}</Text>
            <Button label="Erneut versuchen" variant="outline" onPress={() => void list.refetch()} />
          </Card>
        ) : requests.length === 0 ? (
          <Text variant="body" tone="muted">
            {EMPTY_TEXT[filter]}
          </Text>
        ) : (
          requests.map((r) => (
            <RequestCard
              key={r.id}
              request={r}
              accepting={acceptingId === r.id}
              onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
              onAccept={() => accept.mutate(r.id)}
            />
          ))
        )}
      </Section>

      <View className="gap-3">
        <Button
          label="Benachrichtigungen"
          icon={Bell}
          variant="outline"
          fullWidth
          onPress={() => router.push("/settings")}
        />
        {thanks.data && thanks.data.count > 0 ? (
          <Text variant="secondary" className="text-center">
            {`${thanks.data.count}-mal bedankt.`}
          </Text>
        ) : null}
      </View>
    </Screen>
  );
}
