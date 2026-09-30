import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { Bell, Settings } from "lucide-react-native";
import { useMemo } from "react";
import { RefreshControl, View } from "react-native";

import { PresenceTile } from "@/components/presence-tile";
import { StartBlanketsTile } from "@/components/start-blankets-tile";
import { StartRequests } from "@/components/start-requests";
import { Button, PageHeader, Screen, Section } from "@/components/ui";
import { BLANKET_PLAN_EVENT, BLANKET_STATE_EVENT, blanketKeys, useToday } from "@/lib/api/blankets";
import { PRESENCE_KEY } from "@/lib/api/presence";
import { requestKeys } from "@/lib/api/requests";
import { useAuth } from "@/lib/auth";
import { greeting, myBlanketLines, nightTempShort, nightUnit, startWeatherText } from "@/lib/blankets";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { colors } from "@/lib/theme";

/** Start screen (JAN-35): greeting, tonight's weather and blankets for my horses, presence, progress, requests. */
export default function Home() {
  const queryClient = useQueryClient();
  const { user, me } = useAuth();
  const today = useToday();
  useInvalidateOnEvents({
    [BLANKET_STATE_EVENT]: [blanketKeys.today],
    [BLANKET_PLAN_EVENT]: [blanketKeys.today],
  });

  const mineIds = useMemo(
    () => new Set([...(me?.roles.owned_horse_ids ?? []), ...(me?.roles.rider_horse_ids ?? [])]),
    [me],
  );
  const data = today.data;
  const lines = data ? myBlanketLines(data.horses, mineIds) : [];
  const weather = data?.weather ?? null;

  return (
    <Screen
      refreshControl={
        <RefreshControl
          refreshing={today.isRefetching}
          onRefresh={() => {
            void queryClient.invalidateQueries({ queryKey: blanketKeys.all });
            void queryClient.invalidateQueries({ queryKey: PRESENCE_KEY });
            void queryClient.invalidateQueries({ queryKey: requestKeys.all });
          }}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <PageHeader
        title={greeting(new Date().getHours(), user?.name ?? "")}
        action={
          <View className="-mt-1 flex-row gap-1">
            <Button
              size="icon"
              variant="ghost"
              icon={Bell}
              accessibilityLabel="Erinnerungen"
              onPress={() => router.push("/reminders" as Href)}
            />
            <Button
              size="icon"
              variant="ghost"
              icon={Settings}
              accessibilityLabel="Einstellungen"
              onPress={() => router.push("/settings" as Href)}
            />
          </View>
        }
        value={data ? nightTempShort(weather) : "–"}
        valueSize="lg"
        unit={nightUnit(weather)}
        description={data ? startWeatherText(lines, weather !== null, mineIds.size > 0) : "Wird geladen ..."}
      />

      <Section title="Anwesenheit">
        <PresenceTile />
      </Section>

      <Section title="Decken">
        <StartBlanketsTile today={data} mineIds={mineIds} />
      </Section>

      <Section
        title="Offene Anfragen"
        action={<Button label="Alle" variant="ghost" size="sm" onPress={() => router.push("/requests" as Href)} />}
      >
        <StartRequests />
      </Section>
    </Screen>
  );
}
