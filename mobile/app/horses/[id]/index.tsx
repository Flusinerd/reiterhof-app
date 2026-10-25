import { useQueryClient } from "@tanstack/react-query";
import { router, useLocalSearchParams, type Href } from "expo-router";
import { ClipboardPlus } from "lucide-react-native";
import { useMemo, useState } from "react";
import { Alert, RefreshControl, View } from "react-native";

import { BlanketChoiceSheet } from "@/components/blanket-choice-sheet";
import { HorseDiary } from "@/components/horse-diary";
import { HorseFileTab } from "@/components/horse-file-tab";
import { HorseHealthList } from "@/components/horse-health-list";
import { HorseHero } from "@/components/horse-hero";
import { useHorsePhotoSheet } from "@/components/horse-photo-sheet";
import { HorseObservations } from "@/components/horse-observations";
import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseTodayRows } from "@/components/horse-today-rows";
import { HorseWeekStrip } from "@/components/horse-week-strip";
import { Button, Screen, Section, Segmented, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { blanketKeys, useToday as useBlanketsToday } from "@/lib/api/blankets";
import { horseKeys, horsesApi, useHealth, useHorse, useHorseMutation } from "@/lib/api/horses";
import { trainingKeys, useWeek } from "@/lib/api/training";
import { useAuth } from "@/lib/auth";
import { horseRoutes } from "@/lib/horse-format";
import { buildHorseRows, horseWeekStrip, type RowAction } from "@/lib/horse-page";
import { rehaFromObservationRoute } from "@/lib/observations";
import { localDate } from "@/lib/presence-format";
import { colors } from "@/lib/theme";

const SEGMENTS = [
  { value: "diary", label: "Tagebuch" },
  { value: "health", label: "Gesundheit" },
  { value: "file", label: "Akte" },
] as const;

type Segment = (typeof SEGMENTS)[number]["value"];

/** The horse page as a sub page. */
export default function HorseRecordRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  return <HorseRecord id={id} back />;
}

/**
 * The horse page ("Abendlicht"): hero with the photo (or the horse's color and initial), what is up
 * today, the week as a strip, then Tagebuch, Gesundheit and Akte. The horse tab shows it without the
 * back button for the user's only horse.
 */
export function HorseRecord({ id, back }: { id: string; back: boolean }) {
  const queryClient = useQueryClient();
  const { me } = useAuth();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const horse = useHorse(id);
  const h = horse.data;
  const canSeeTraining = !!h && (h.can_manage || h.i_ride);
  const week = useWeek(canSeeTraining ? id : undefined);
  const health = useHealth(id);
  const blankets = useBlanketsToday();
  const markDone = useHorseMutation((itemId: string) => horsesApi.markDone(itemId), () => [{ type: "health" }]);
  const photo = useHorsePhotoSheet({ id, name: h?.name ?? "", photo_url: h?.photo_url ?? null });
  const [segment, setSegment] = useState<Segment>("diary");
  const [blanketOpen, setBlanketOpen] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const today = localDate(new Date(), timeZone);
  const days = week.data?.days ?? null;
  const blanketToday = blankets.data?.horses.find((b) => b.horse.id === id);
  const rows = useMemo(
    () =>
      h
        ? buildHorseRows({
            horseId: id,
            today,
            days: canSeeTraining ? days : null,
            blanket:
              blanketToday && blankets.data
                ? {
                    recommendation: blanketToday.recommendation,
                    rule_recommendation: blanketToday.rule_recommendation,
                    override: blanketToday.override,
                    done: blanketToday.done,
                    state: blanketToday.state,
                    day: blankets.data.day,
                  }
                : null,
            health: health.data?.items ?? null,
            canManage: h.can_manage,
          })
        : [],
    [h, id, today, canSeeTraining, days, blanketToday, blankets.data, health.data],
  );
  const strip = useMemo(() => (days ? horseWeekStrip(days, today) : []), [days, today]);

  async function refresh() {
    setRefreshing(true);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: horseKeys.all }),
      queryClient.invalidateQueries({ queryKey: trainingKeys.all }),
      queryClient.invalidateQueries({ queryKey: blanketKeys.all }),
    ]);
    setRefreshing(false);
  }

  function onAction(action: RowAction) {
    if (action.kind === "start") router.push(`/training?horse=${id}` as Href);
    else if (action.kind === "blanket") setBlanketOpen(true);
    else if (action.kind === "done" && action.itemId) {
      markDone.mutate(action.itemId, { onError: (err) => Alert.alert("Das hat nicht geklappt", errorMessage(err)) });
    }
  }

  if (!h) {
    return (
      <Screen back={back} menu={back}>
        {horse.isError ? <HorseError error={horse.error} onRetry={() => horse.refetch()} /> : <HorseLoading />}
      </Screen>
    );
  }

  const go = (path: string) => router.push(path as Href);
  const items = health.data?.items ?? [];
  const openRows = rows.length > 0;

  return (
    <Screen
      back={back}
      menu={back}
      edges={back ? undefined : []}
      refreshControl={<RefreshControl refreshing={refreshing} onRefresh={refresh} tintColor={colors.primary.DEFAULT} />}
    >
      <HorseHero horse={h} canEditPhoto={h.can_manage} onEditPhoto={photo.open} behindStatusBar={!back} />

      {openRows ? (
        <Section title={`Heute für ${h.name}`}>
          <HorseTodayRows rows={rows} busy={markDone.isPending ? "health" : null} onAction={onAction} />
        </Section>
      ) : null}

      {strip.length > 0 ? (
        <Section title="Diese Woche">
          <HorseWeekStrip items={strip} onPress={() => go(`/training/week?horse=${id}`)} />
        </Section>
      ) : null}

      <View className="gap-5">
        <Segmented items={SEGMENTS} value={segment} onValueChange={(v) => setSegment(v as Segment)} />
        {segment === "diary" ? <HorseDiary horse={h} timeZone={timeZone} /> : null}
        {segment === "health" ? (
          <View className="gap-6">
            {health.isError ? (
              <HorseError error={health.error} onRetry={() => health.refetch()} />
            ) : !health.data ? (
              <HorseLoading />
            ) : items.length === 0 ? (
              <Text variant="body" tone="muted">
                Noch keine Gesundheitstermine.
              </Text>
            ) : (
              <HorseHealthList items={items} onPress={() => go(horseRoutes.health(h.id))} />
            )}
            <Button label="Alle Termine" variant="outline" size="sm" onPress={() => go(horseRoutes.health(h.id))} />
            <Section title="Auffälligkeiten">
              <HorseObservations
                horseId={h.id}
                renderActions={(o) =>
                  h.can_manage && !o.reha_plan_id ? (
                    <Button
                      label="Reha-Plan erstellen"
                      icon={ClipboardPlus}
                      variant="outline"
                      size="sm"
                      fullWidth
                      onPress={() => go(rehaFromObservationRoute(o.id, h.id))}
                    />
                  ) : null
                }
              />
            </Section>
          </View>
        ) : null}
        {segment === "file" ? <HorseFileTab horse={h} /> : null}
      </View>

      {h.can_manage ? <BlanketChoiceSheet open={blanketOpen} onOpenChange={setBlanketOpen} horseId={id} mode="override" /> : null}
      {h.can_manage ? photo.sheet : null}
    </Screen>
  );
}
