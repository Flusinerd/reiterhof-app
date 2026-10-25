import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { Share2 } from "lucide-react-native";
import { useMemo } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { RequestRow } from "@/components/request-row";
import { Button, Card, PageHeader, Screen, SectionTitle, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import { toIsoDate } from "@/lib/requests";
import { shareCalendarFeed } from "@/lib/requests-calendar";
import { requestErrorMessage } from "@/lib/requests-errors";
import { groupRequestsByDay } from "@/lib/stable-tab";
import { useColors } from "@/lib/theme";
import { useRefreshOnFocus } from "@/lib/use-refresh-on-focus";

/** "Mein Kalender": the requests I help with, by day. */
export default function HelperCalendar() {
  const c = useColors();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const today = useMemo(() => toIsoDate(new Date()), []);
  const query = useQuery({ queryKey: requestKeys.calendar, queryFn: requestsApi.calendar });
  useRefreshOnFocus([requestKeys.calendar]);
  const requests = query.data?.requests ?? [];

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl
          refreshing={query.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: requestKeys.calendar })}
          tintColor={c.primary.DEFAULT}
        />
      }
    >
      <PageHeader
        eyebrow="Anfragen"
        title="Mein Kalender"
        value={query.data ? String(requests.length) : "–"}
        valueSize="sm"
        unit={requests.length === 1 ? "Termin" : "Termine"}
      />

      {query.isPending ? (
        <ActivityIndicator color={c.primary.DEFAULT} />
      ) : query.isError ? (
        <Card className="gap-3">
          <Text variant="body">{requestErrorMessage(query.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void query.refetch()} />
        </Card>
      ) : requests.length === 0 ? (
        <View className="gap-3">
          <Text variant="body" tone="muted">
            Hier stehen Anfragen, bei denen du hilfst.
          </Text>
          <Button label="Zu den Anfragen" variant="outline" onPress={() => router.replace("/requests")} />
        </View>
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
                  onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
                />
              ))}
            </View>
          </View>
        ))
      )}

      <Button label="Kalenderdatei teilen" icon={Share2} variant="outline" fullWidth onPress={() => void shareCalendarFeed()} />
    </Screen>
  );
}
