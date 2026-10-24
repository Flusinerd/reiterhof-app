import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { Plus } from "lucide-react-native";
import { RefreshControl } from "react-native";

import { PresenceButton } from "@/components/app-menu";
import { PresenceTile } from "@/components/presence-tile";
import { StartBlanketsTile } from "@/components/start-blankets-tile";
import { StartRequests } from "@/components/start-requests";
import { WeatherWarnings } from "@/components/weather-warnings";
import { Button, PageHeader, Screen, Section } from "@/components/ui";
import { BLANKET_STATE_EVENT, blanketKeys, useToday } from "@/lib/api/blankets";
import { PRESENCE_KEY } from "@/lib/api/presence";
import { requestKeys } from "@/lib/api/requests";
import { weatherKeys } from "@/lib/api/weather";
import { useAuth } from "@/lib/auth";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { colors } from "@/lib/theme";

/** "Stall": what concerns everybody. Who is here, tonight's blankets, requests for help, warnings. */
export default function Stable() {
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const today = useToday();
  useInvalidateOnEvents({ [BLANKET_STATE_EVENT]: [blanketKeys.today] });
  const mineIds = new Set([...(me?.roles.owned_horse_ids ?? []), ...(me?.roles.rider_horse_ids ?? [])]);
  const openHorses = today.data?.horses.filter((h) => !h.done).length ?? 0;

  return (
    <Screen
      menu={false}
      refreshControl={
        <RefreshControl
          refreshing={today.isRefetching}
          onRefresh={() => {
            void queryClient.invalidateQueries({ queryKey: blanketKeys.all });
            void queryClient.invalidateQueries({ queryKey: PRESENCE_KEY });
            void queryClient.invalidateQueries({ queryKey: requestKeys.all });
            void queryClient.invalidateQueries({ queryKey: weatherKeys.warnings });
          }}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <PageHeader eyebrow={me?.stable?.name} title="Stall" action={<PresenceButton />} />

      <WeatherWarnings timeZone={me?.stable?.timezone ?? "Europe/Berlin"} />

      <Section title="Wer ist da">
        <PresenceTile />
      </Section>

      <Section title="Decken heute Nacht">
        <StartBlanketsTile today={today.data} mineIds={mineIds} />
        {openHorses > 0 ? (
          <Button label="Runde starten" fullWidth onPress={() => router.push("/blankets/round" as Href)} />
        ) : null}
      </Section>

      <Section
        title="Bitte um Hilfe"
        action={<Button label="Neu" icon={Plus} variant="ghost" size="sm" onPress={() => router.push("/requests/new" as Href)} />}
      >
        <StartRequests />
        <Button label="Alle Anfragen" variant="outline" size="sm" onPress={() => router.push("/requests" as Href)} />
      </Section>
    </Screen>
  );
}
