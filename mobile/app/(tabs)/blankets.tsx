import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { BlanketDoneRow, BlanketHorseCard } from "@/components/blanket-horse-card";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { BLANKET_PLAN_EVENT, BLANKET_STATE_EVENT, blanketKeys, useSetState, useToday } from "@/lib/api/blankets";
import { useAuth } from "@/lib/auth";
import {
  nightLine,
  progressFraction,
  progressLabel,
  reminderHint,
  type StateAction,
} from "@/lib/blankets";
import { horseRoutes } from "@/lib/horse-format";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { colors } from "@/lib/theme";

/** "Mittwoch, 30. September" */
function dateLine(date: Date): string {
  return new Intl.DateTimeFormat("de-DE", { weekday: "long", day: "numeric", month: "long" }).format(date);
}

/** Tab "Decken" (JAN-32): who still needs a blanket decision tonight, and who is done. */
export default function Blankets() {
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const today = useToday();
  const setState = useSetState();
  const [pending, setPending] = useState<{ horseId: string; action: StateAction } | null>(null);
  const [error, setError] = useState<string | null>(null);
  useInvalidateOnEvents({
    [BLANKET_STATE_EVENT]: [blanketKeys.today],
    [BLANKET_PLAN_EVENT]: [blanketKeys.all],
  });
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const openPlan = (horseId: string) => router.push(horseRoutes.blanketPlan(horseId) as Href);

  function act(horseId: string, action: StateAction) {
    setError(null);
    setPending({ horseId, action });
    setState.mutate(
      { horseId, action },
      { onError: (e) => setError(errorMessage(e)), onSettled: () => setPending(null) },
    );
  }

  const data = today.data;
  const open = data?.horses.filter((h) => !h.done) ?? [];
  const done = data?.horses.filter((h) => h.done) ?? [];

  return (
    <Screen
      refreshControl={
        <RefreshControl
          refreshing={today.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: blanketKeys.all })}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <Hero
        eyebrow={dateLine(new Date())}
        value={data ? progressLabel(data.progress) : "–"}
        valueSize="lg"
        unit="versorgt"
        description={data ? nightLine(data.weather) : "Wird geladen ..."}
      >
        {data ? (
          <View className="gap-2">
            <View
              className="h-2 overflow-hidden rounded-pill bg-white/20"
              accessibilityRole="progressbar"
              accessibilityValue={{ min: 0, max: data.progress.total, now: data.progress.done }}
            >
              <View className="h-2 rounded-pill bg-white" style={{ width: `${progressFraction(data.progress) * 100}%` }} />
            </View>
            <Text variant="secondary" className="text-white/70">
              {reminderHint(data.reminder_time)}
            </Text>
          </View>
        ) : null}
      </Hero>

      {error ? (
        <Card className="border-danger bg-danger-soft">
          <Text variant="bodySm" tone="danger" accessibilityRole="alert">
            {error}
          </Text>
        </Card>
      ) : null}

      {!data ? (
        today.isError ? (
          <HorseError error={today.error} onRetry={() => today.refetch()} />
        ) : (
          <HorseLoading />
        )
      ) : (
        <>
          {open.length > 0 ? (
            <View className="gap-3">
              <SectionLabel>Noch offen</SectionLabel>
              {open.map((item) => (
                <BlanketHorseCard
                  key={item.horse.id}
                  item={item}
                  timeZone={timeZone}
                  pending={pending?.horseId === item.horse.id ? pending.action : null}
                  onAction={(a) => act(item.horse.id, a)}
                  onOpenPlan={() => openPlan(item.horse.id)}
                />
              ))}
            </View>
          ) : (
            <Card>
              <Text variant="bodyStrong">{data.horses.length === 0 ? "Noch keine Pferde" : "Alles erledigt"}</Text>
            </Card>
          )}

          {done.length > 0 ? (
            <View className="gap-3">
              <SectionLabel>Erledigt</SectionLabel>
              {done.map((item) => (
                <BlanketDoneRow
                  key={item.horse.id}
                  item={item}
                  timeZone={timeZone}
                  pending={pending?.horseId === item.horse.id ? pending.action : null}
                  onAction={(a) => act(item.horse.id, a)}
                  onOpenPlan={() => openPlan(item.horse.id)}
                />
              ))}
            </View>
          ) : null}
        </>
      )}
    </Screen>
  );
}
