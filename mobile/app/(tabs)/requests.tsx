import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { Bell, CalendarDays, Plus } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, ScrollView, View } from "react-native";

import { RequestCard } from "@/components/request-card";
import { Button, Card, Hero, Pill, Screen, SectionLabel, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { colors } from "@/lib/theme";
import { REQUEST_FILTERS, filterParams, toIsoDate, type RequestFilter } from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";

const EMPTY_TEXT: Record<RequestFilter, string> = {
  open: "Aktuell sucht niemand Hilfe. Schön!",
  mine: "Du hast noch keine Anfrage gestellt.",
  helping: "Du hilfst gerade bei keiner Anfrage.",
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
      <Hero
        eyebrow="Stallgasse"
        value={String(open)}
        unit={open === 1 ? "offene Anfrage" : "offene Anfragen"}
        description={open === 0 ? "Gerade braucht niemand Hilfe." : "Vielleicht kannst du heute jemandem helfen."}
      >
        <View className="flex-row flex-wrap gap-3">
          <Button
            label="Neu"
            icon={Plus}
            variant="secondary"
            onPress={() => router.push("/requests/new")}
            accessibilityLabel="Neue Anfrage stellen"
          />
        </View>
      </Hero>

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

      <View className="gap-3">
        <SectionLabel
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
          {REQUEST_FILTERS.find((f) => f.value === filter)?.label ?? ""}
        </SectionLabel>
        {list.isPending ? (
          <ActivityIndicator />
        ) : list.isError ? (
          <Card className="gap-3">
            <Text variant="body">{requestErrorMessage(list.error)}</Text>
            <Button label="Erneut versuchen" variant="outline" onPress={() => void list.refetch()} />
          </Card>
        ) : requests.length === 0 ? (
          <Card>
            <Text variant="secondary">{EMPTY_TEXT[filter]}</Text>
          </Card>
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
      </View>

      <SectionLabel>Benachrichtigungen</SectionLabel>
      <Card className="gap-2">
        <Text variant="body">Ob dich neue Anfragen und Erinnerungen benachrichtigen, stellst du in den Einstellungen ein.</Text>
        <Button
          label="Benachrichtigungen einstellen"
          icon={Bell}
          variant="outline"
          fullWidth
          onPress={() => router.push("/settings")}
        />
        {thanks.data && thanks.data.count > 0 ? (
          <Text variant="secondary">
            {thanks.data.count === 1 ? "Dir wurde 1-mal Danke gesagt." : `Dir wurde ${thanks.data.count}-mal Danke gesagt.`}
          </Text>
        ) : null}
      </Card>
    </Screen>
  );
}
