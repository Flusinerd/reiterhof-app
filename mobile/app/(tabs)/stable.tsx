import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { ChevronRight, Plus } from "lucide-react-native";
import { Pressable, RefreshControl, View } from "react-native";

import { PresenceButton } from "@/components/app-menu";
import { BlanketBoard, PresenceRow } from "@/components/stable-parts";
import { StartRequests } from "@/components/start-requests";
import { WeatherWarnings } from "@/components/weather-warnings";
import { Button, Icon, PageHeader, Screen, SectionTitle, Text } from "@/components/ui";
import { BLANKET_STATE_EVENT, blanketKeys, useToday } from "@/lib/api/blankets";
import { PRESENCE_KEY } from "@/lib/api/presence";
import { requestKeys } from "@/lib/api/requests";
import { weatherKeys } from "@/lib/api/weather";
import { useAuth } from "@/lib/auth";
import { progressLabel } from "@/lib/blankets";
import { horseRoutes } from "@/lib/horse-format";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { boardLine, blanketBoard, boardMineLine } from "@/lib/stable-tab";
import { useColors } from "@/lib/theme";

/**
 * "Stall": what concerns everybody. Warnings on top, who is here, tonight's blankets as a board,
 * requests for help as messages. The one primary button is "Runde starten" while horses are open.
 */
export default function Stable() {
  const c = useColors();
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const today = useToday();
  useInvalidateOnEvents({ [BLANKET_STATE_EVENT]: [blanketKeys.today] });
  const mineIds = new Set([...(me?.roles.owned_horse_ids ?? []), ...(me?.roles.rider_horse_ids ?? [])]);
  const data = today.data;
  const openHorses = data?.horses.filter((h) => !h.done).length ?? 0;
  const mineLine = data ? boardMineLine(blanketBoard(data.horses, mineIds)) : null;

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
          tintColor={c.primary.DEFAULT}
        />
      }
    >
      <PageHeader eyebrow={me?.stable?.name} title="Stall" action={<PresenceButton />} />

      <WeatherWarnings timeZone={me?.stable?.timezone ?? "Europe/Berlin"} />

      <PresenceRow />

      {data && data.horses.length === 0 ? null : (
        <View className="gap-3">
          <SectionTitle
            action={
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={data ? `Decken: ${progressLabel(data.progress)} versorgt. Übersicht öffnen` : "Decken, Übersicht öffnen"}
                onPress={() => router.push("/blankets" as Href)}
                className="min-h-touch flex-row items-center gap-1 active:opacity-70"
              >
                <Text variant="bodyStrong" tone="brass">
                  {data ? `${progressLabel(data.progress)} versorgt` : "…"}
                </Text>
                <Icon as={ChevronRight} size={18} className="text-muted" />
              </Pressable>
            }
          >
            Decken heute Nacht
          </SectionTitle>
          {data ? (
            <>
              <BlanketBoard horses={data.horses} mineIds={mineIds} onOpen={(id) => router.push(horseRoutes.blanketPlan(id) as Href)} />
              <View className="gap-0.5">
                <Text variant="secondary">{boardLine(data)}</Text>
                {mineLine ? (
                  <Text variant="secondary" tone="accent">
                    {mineLine}
                  </Text>
                ) : null}
              </View>
              {openHorses > 0 ? <Button label="Runde starten" fullWidth onPress={() => router.push("/blankets/round" as Href)} /> : null}
            </>
          ) : (
            <Text variant="secondary">Wird geladen …</Text>
          )}
        </View>
      )}

      <View className="gap-1">
        <SectionTitle
          action={<Button label="Neu" icon={Plus} variant="ghost" size="sm" className="-mr-4" accessibilityLabel="Neue Bitte um Hilfe" onPress={() => router.push("/requests/new" as Href)} />}
        >
          Bitte um Hilfe
        </SectionTitle>
        <StartRequests />
        <Button label="Alle Anfragen" variant="ghost" size="sm" className="-ml-4" onPress={() => router.push("/requests" as Href)} />
      </View>
    </Screen>
  );
}
