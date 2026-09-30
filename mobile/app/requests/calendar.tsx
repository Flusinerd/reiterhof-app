import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "expo-router";
import { Share2 } from "lucide-react-native";
import { useMemo } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { RequestCard } from "@/components/request-card";
import { Button, Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { requestKeys, requestsApi } from "@/lib/api/requests";
import { colors } from "@/lib/theme";
import { relativeDay, toIsoDate, type HelpRequest } from "@/lib/requests";
import { shareCalendarFeed } from "@/lib/requests-calendar";
import { requestErrorMessage } from "@/lib/requests-errors";

/** Groups requests (already sorted by date) by their first day. */
function groupByDay(requests: HelpRequest[]): { date: string; items: HelpRequest[] }[] {
  const groups: { date: string; items: HelpRequest[] }[] = [];
  for (const r of requests) {
    const last = groups[groups.length - 1];
    if (last && last.date === r.date) last.items.push(r);
    else groups.push({ date: r.date, items: [r] });
  }
  return groups;
}

export default function HelperCalendar() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const today = useMemo(() => toIsoDate(new Date()), []);
  const query = useQuery({ queryKey: requestKeys.calendar, queryFn: requestsApi.calendar });
  const requests = query.data?.requests ?? [];

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl
          refreshing={query.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: requestKeys.calendar })}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <Hero
        eyebrow="Mein Kalender"
        value={String(requests.length)}
        unit={requests.length === 1 ? "Termin" : "Termine"}
      />

      {query.isPending ? (
        <ActivityIndicator />
      ) : query.isError ? (
        <Card className="gap-3">
          <Text variant="body">{requestErrorMessage(query.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void query.refetch()} />
        </Card>
      ) : requests.length === 0 ? (
        <Card className="gap-3">
          <Text variant="secondary">Hier stehen Anfragen, bei denen du hilfst.</Text>
          <Button label="Zu den Anfragen" variant="outline" onPress={() => router.replace("/(tabs)/requests")} />
        </Card>
      ) : (
        groupByDay(requests).map((g) => (
          <View key={g.date} className="gap-3">
            <SectionLabel>{relativeDay(g.date, today)}</SectionLabel>
            {g.items.map((r) => (
              <RequestCard
                key={r.id}
                request={r}
                onOpen={() => router.push({ pathname: "/requests/[id]", params: { id: r.id } })}
              />
            ))}
          </View>
        ))
      )}

      <Button
        label="Kalenderdatei teilen"
        icon={Share2}
        variant="outline"
        fullWidth
        onPress={() => void shareCalendarFeed()}
      />
    </Screen>
  );
}
