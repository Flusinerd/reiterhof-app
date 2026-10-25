import { useQueryClient } from "@tanstack/react-query";
import type { Href } from "expo-router";
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
import { pushOnce } from "@/lib/navigation";
import { useColors } from "@/lib/theme";
import { mutationKeys } from "@/lib/query-keys";
import { useKeyedErrors } from "@/lib/use-keyed-errors";
import { usePendingVariables, useSubmitOnce } from "@/lib/use-pending";
import { useRefreshOnFocus } from "@/lib/use-refresh-on-focus";

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
  useRefreshOnFocus([blanketKeys.all]);
  // The row moves to "Erledigt" at once and comes back with the error when the call fails.
  const setState = useSetState({ optimistic: true });
  // Every horse whose state is being sent shows its own spinner, however many are in flight.
  const sending = usePendingVariables<{ horseId: string; action: StateAction }>(mutationKeys.blanketState);
  const pendingOf = (horseId: string) => sending.find((v) => v.horseId === horseId)?.action ?? null;
  const once = useSubmitOnce();
  // The error of a horse's last action, shown under that horse's buttons (the list can be long).
  const [errors, setError] = useKeyedErrors();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const openPlan = (horseId: string) => pushOnce(horseRoutes.blanketPlan(horseId) as Href);

  /** The promise keeps the tapped button busy; a repeat for the same horse right after a success is ignored. */
  function act(horseId: string, action: StateAction) {
    setError(horseId, null);
    return once.run(horseId, () => setState.mutateAsync({ horseId, action })).catch((e) => setError(horseId, errorMessage(e)));
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
        <Button label="Runde starten" fullWidth onPress={() => pushOnce("/blankets/round" as Href)} />
      ) : null}

      <WeatherWarnings timeZone={timeZone} />

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
                    pending={pendingOf(item.horse.id)}
                    error={errors[item.horse.id]}
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
                    pending={pendingOf(item.horse.id)}
                    error={errors[item.horse.id]}
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
