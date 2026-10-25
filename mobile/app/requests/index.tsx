import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { CalendarDays, Plus } from "lucide-react-native";
import { useMemo, useState } from "react";
import { ActivityIndicator, RefreshControl, ScrollView, View } from "react-native";

import { RequestRow } from "@/components/request-row";
import { Button, Card, PageHeader, Pill, Screen, SectionTitle, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import { REQUEST_FILTERS, filterParams, toIsoDate, type RequestFilter } from "@/lib/requests";
import { requestErrorMessage } from "@/lib/requests-errors";
import { groupRequestsByDay } from "@/lib/stable-tab";
import { useColors } from "@/lib/theme";
import { useAcceptRequest } from "@/lib/use-accept-request";
import { useRefreshOnFocus } from "@/lib/use-refresh-on-focus";

const EMPTY_TEXT: Record<RequestFilter, string> = {
  open: "Keine offenen Anfragen.",
  mine: "Noch keine eigenen Anfragen.",
  helping: "Du hilfst bei keiner Anfrage.",
  done: "Noch nichts erledigt.",
};

/** All requests for help as messages grouped by day; the filters are pills, "Neu" is the one primary button. */
export default function Requests() {
  const c = useColors();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [filter, setFilter] = useState<RequestFilter>("open");
  const today = useMemo(() => toIsoDate(new Date()), []);

  const params = filterParams(filter, today);
  const list = useQuery({ queryKey: requestKeys.list(params), queryFn: () => requestsApi.list(params) });
  useRefreshOnFocus([requestKeys.all]);
  const thanks = useQuery({ queryKey: requestKeys.thanks, queryFn: requestsApi.myThanks });
  const { accept, acceptingId, error } = useAcceptRequest();

  const open = list.data?.open_count ?? 0;
  const requests = list.data?.requests ?? [];

  return (
    <Screen back
      refreshControl={
        <RefreshControl
          refreshing={list.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: requestKeys.all })}
          tintColor={c.primary.DEFAULT}
        />
      }
    >
      <PageHeader
        eyebrow="Bitte um Hilfe"
        title="Anfragen"
        action={
          <View className="flex-row items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              icon={CalendarDays}
              accessibilityLabel="Mein Kalender"
              onPress={() => router.push("/requests/calendar")}
            />
            <Button
              label="Neu"
              icon={Plus}
              size="sm"
              onPress={() => router.push("/requests/new")}
              accessibilityLabel="Neue Anfrage"
            />
          </View>
        }
        description={list.data ? `${open} ${open === 1 ? "offene Anfrage" : "offene Anfragen"}` : undefined}
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

      {list.isPending ? (
        <ActivityIndicator color={c.primary.DEFAULT} />
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
        groupRequestsByDay(requests, today).map((group) => (
          <View key={group.date} className="gap-1">
            <SectionTitle>{group.heading}</SectionTitle>
            <View>
              {group.items.map((r, i) => (
                <RequestRow
                  key={r.id}
                  request={r}
                  today={today}
                  userId={user?.id}
                  withDay={false}
                  last={i === group.items.length - 1}
                  accepting={acceptingId === r.id}
                  onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
                  onAccept={() => accept(r.id)}
                />
              ))}
            </View>
          </View>
        ))
      )}

      {thanks.data && thanks.data.count > 0 ? (
        <Text variant="secondary" className="text-center">
          {`${thanks.data.count}-mal bedankt.`}
        </Text>
      ) : null}
    </Screen>
  );
}
