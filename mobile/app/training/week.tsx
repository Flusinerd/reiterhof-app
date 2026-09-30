import { useLocalSearchParams } from "expo-router";
import { ChevronLeft, ChevronRight } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import { LoadBar } from "@/components/training-dots";
import { HorseSwitcher } from "@/components/training-horse-switcher";
import { Badge, Button, Card, Divider, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { trainingError, useTakeDay, useTrainingHorses, useWeek, type WeekDay } from "@/lib/api/training";
import { addDays, dayStatusLabel, formatDayLong, weekRangeLabel, weekdayShort } from "@/lib/training";

/** Week view (JAN-59): who trains on which day, show days, load bar and an assessment. */
export default function TrainingWeek() {
  const params = useLocalSearchParams<{ horse?: string }>();
  const horses = useTrainingHorses();
  const [horseId, setHorseId] = useState<string | undefined>(params.horse);
  const [start, setStart] = useState<string>();

  const list = horses.data ?? [];
  const activeId = horseId ?? list[0]?.id;
  const week = useWeek(activeId, start);
  const take = useTakeDay(activeId ?? "");
  const data = week.data;

  if (horses.isPending || (week.isPending && !!activeId)) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }
  if (!activeId || week.isError || !data) {
    return (
      <Screen back>
        <Card className="gap-3">
          <Text variant="secondary">{week.isError ? trainingError(week.error) : "Kein Pferd ausgewählt."}</Text>
          {week.isError ? <Button label="Erneut versuchen" variant="outline" onPress={() => void week.refetch()} /> : null}
        </Card>
      </Screen>
    );
  }

  const horseName = list.find((h) => h.id === activeId)?.name ?? "";
  const shift = (days: number) => setStart(addDays(data.start, days));

  return (
    <Screen
      back
      refreshControl={<RefreshControl refreshing={week.isRefetching} onRefresh={() => void week.refetch()} />}
    >
      <HorseSwitcher
        horses={list}
        selected={activeId}
        onSelect={(id) => {
          setHorseId(id);
          setStart(undefined);
        }}
      />

      <Hero
        tone="soft"
        eyebrow={`${horseName} · ${weekRangeLabel(data.start, data.end)}`}
        title={data.assessment}
      >
        <LoadBar segments={data.segments} />
      </Hero>

      <View className="flex-row items-center justify-between">
        <Button variant="outline" size="icon" icon={ChevronLeft} accessibilityLabel="Vorherige Woche" onPress={() => shift(-7)} />
        <Button label="Diese Woche" variant="ghost" size="sm" onPress={() => setStart(undefined)} />
        <Button variant="outline" size="icon" icon={ChevronRight} accessibilityLabel="Nächste Woche" onPress={() => shift(7)} />
      </View>

      <SectionLabel>Tage</SectionLabel>
      <Card padded={false}>
        {data.days.map((d, i) => (
          <View key={d.date}>
            {i > 0 ? <Divider /> : null}
            <DayRow
              day={d}
              busy={take.isPending && take.variables?.day === d.date}
              onTake={() => take.mutate({ day: d.date, status: "planned" })}
            />
          </View>
        ))}
      </Card>
      {take.isError ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(take.error)}
        </Text>
      ) : null}
    </Screen>
  );
}

function DayRow({ day, busy, onTake }: { day: WeekDay; busy: boolean; onTake: () => void }) {
  const label = dayStatusLabel(day.status, day.user, day.is_me, day.rest_reason);
  const open = day.status === "open" || (day.status === "today" && !day.user);
  return (
    <View className="gap-2 p-4" accessible={false}>
      <View className="min-h-touch flex-row items-center gap-3">
        <View className="w-12 items-center">
          <Text variant="bodyStrong" className={day.is_today ? "text-accent-text" : undefined}>
            {weekdayShort(day.date)}
          </Text>
          <Text variant="caption">{Number(day.date.slice(8, 10))}.</Text>
        </View>
        <View className="flex-1 gap-0.5" accessible accessibilityLabel={`${formatDayLong(day.date)}: ${label}`}>
          <Text variant="body" tone={open ? "muted" : "default"}>
            {label}
          </Text>
          {day.activity ? (
            <View className="flex-row items-center gap-1.5">
              <ActivityIcon activity={day.activity} size={14} className="text-muted" />
              <Text variant="caption">
                {day.label}
                {day.minutes > 0 ? ` · ${day.minutes} Min.` : ""}
              </Text>
            </View>
          ) : null}
        </View>
        {day.can_take && !day.is_me ? (
          <Button label="Ich" size="sm" variant="secondary" loading={busy} onPress={onTake} />
        ) : null}
      </View>
      {day.show ? (
        <View className="ml-[60px] gap-1 rounded-tile bg-accent-soft p-3" accessibilityLabel={`Turnier ${day.show.name}`}>
          <View className="flex-row items-center gap-2">
            <Badge variant="accent" label="Turnier" />
            <Text variant="bodyStrong" className="flex-1">
              {day.show.name}
            </Text>
          </View>
          {day.show.classes ? <Text variant="secondary">Klassen: {day.show.classes}</Text> : null}
          {day.show.helper ? <Text variant="secondary">Helfer: {day.show.helper}</Text> : null}
        </View>
      ) : null}
    </View>
  );
}
