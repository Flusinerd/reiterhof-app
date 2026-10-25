import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { useState } from "react";
import { RefreshControl, View } from "react-native";

import { BlanketDoneRow, BlanketHorseRow } from "@/components/blanket-horse-row";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { Button, PageHeader, Screen, SectionTitle, Text } from "@/components/ui";
import { WeatherCard } from "@/components/weather-card";
import { WeatherWarnings } from "@/components/weather-warnings";
import { errorMessage } from "@/lib/api";
import { blanketKeys, useSetState, useToday } from "@/lib/api/blankets";
import { weatherKeys } from "@/lib/api/weather";
import { useAuth } from "@/lib/auth";
import {
  nightLine,
  progressFraction,
  progressLabel,
  reminderHint,
  type StateAction,
} from "@/lib/blankets";
import { horseRoutes } from "@/lib/horse-format";
import { useColors } from "@/lib/theme";

/** "Mittwoch, 30. September" */
function dateLine(date: Date): string {
  return new Intl.DateTimeFormat("de-DE", { weekday: "long", day: "numeric", month: "long" }).format(date);
}

/**
 * Blankets of tonight (JAN-32): the head with "4/7 versorgt" and the weather line, "Runde starten"
 * as the one primary button, the horses that still need a decision as rows with their compact state
 * buttons, the finished ones faded below, the forecast at the end.
 */
export default function Blankets() {
  const c = useColors();
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const today = useToday();
  const setState = useSetState();
  const [pending, setPending] = useState<{ horseId: string; action: StateAction } | null>(null);
  const [error, setError] = useState<string | null>(null);
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
    <Screen back
      refreshControl={
        <RefreshControl
          refreshing={today.isRefetching}
          onRefresh={() => {
            void queryClient.invalidateQueries({ queryKey: blanketKeys.all });
            void queryClient.invalidateQueries({ queryKey: weatherKeys.warnings });
          }}
          tintColor={c.primary.DEFAULT}
        />
      }
    >
      <PageHeader
        eyebrow={dateLine(new Date())}
        title="Decken"
        value={data ? progressLabel(data.progress) : "–"}
        valueSize="sm"
        unit="versorgt"
        description={data ? nightLine(data.weather) : "Wird geladen ..."}
      >
        {data ? (
          <View className="gap-2">
            <View
              className="h-2 overflow-hidden rounded-pill bg-border"
              accessibilityRole="progressbar"
              accessibilityValue={{ min: 0, max: data.progress.total, now: data.progress.done }}
            >
              <View className="h-2 rounded-pill bg-primary" style={{ width: `${progressFraction(data.progress) * 100}%` }} />
            </View>
            <Text variant="secondary">{reminderHint(data.reminder_time)}</Text>
          </View>
        ) : null}
      </PageHeader>

      {open.length > 0 ? (
        <Button label="Runde starten" fullWidth onPress={() => router.push("/blankets/round" as Href)} />
      ) : null}

      <WeatherWarnings timeZone={timeZone} />

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
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
            <View className="gap-1">
              <SectionTitle>Noch offen</SectionTitle>
              <View>
                {open.map((item, i) => (
                  <BlanketHorseRow
                    key={item.horse.id}
                    item={item}
                    timeZone={timeZone}
                    last={i === open.length - 1}
                    pending={pending?.horseId === item.horse.id ? pending.action : null}
                    onAction={(a) => act(item.horse.id, a)}
                    onOpenPlan={() => openPlan(item.horse.id)}
                  />
                ))}
              </View>
            </View>
          ) : (
            <Text variant="body" tone="muted">
              {data.horses.length === 0 ? "Noch keine Pferde" : "Alles erledigt"}
            </Text>
          )}

          {done.length > 0 ? (
            <View className="gap-1">
              <SectionTitle>Erledigt</SectionTitle>
              <View>
                {done.map((item, i) => (
                  <BlanketDoneRow
                    key={item.horse.id}
                    item={item}
                    timeZone={timeZone}
                    last={i === done.length - 1}
                    pending={pending?.horseId === item.horse.id ? pending.action : null}
                    onAction={(a) => act(item.horse.id, a)}
                    onOpenPlan={() => openPlan(item.horse.id)}
                  />
                ))}
              </View>
            </View>
          ) : null}

          {data.weather ? <WeatherCard weather={data.weather} timeZone={timeZone} /> : null}
        </>
      )}
    </Screen>
  );
}
