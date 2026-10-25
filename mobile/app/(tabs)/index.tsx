import { router, type Href } from "expo-router";
import { RefreshControl, View } from "react-native";

import { NextTask, Timeline } from "@/components/today-parts";
import { WeatherWarnings } from "@/components/weather-warnings";
import { Button, PageHeader, Screen, SectionTitle, Text } from "@/components/ui";
import { useAuth } from "@/lib/auth";
import { greeting } from "@/lib/blankets";
import { colors } from "@/lib/theme";
import { todaySummary } from "@/lib/today";
import { useTodayData } from "@/lib/use-today";

const dateLine = (timeZone: string) =>
  new Intl.DateTimeFormat("de-DE", { timeZone, weekday: "long", day: "numeric", month: "long" }).format(new Date());

/**
 * "Heute": what is up for the user's own horses (docs/design-system.md). One next task with its
 * action, then a timeline of today and tomorrow. Empty things are left out.
 */
export default function Heute() {
  const { user } = useAuth();
  const data = useTodayData();
  const view = data.view;
  const colorOf = (id?: string) => data.horses.find((h) => h.id === id)?.color_key;

  return (
    <Screen
      menu={false}
      refreshControl={<RefreshControl refreshing={data.refreshing} onRefresh={data.refresh} tintColor={colors.primary.DEFAULT} />}
    >
      <PageHeader
        eyebrow={dateLine(data.timeZone)}
        title={greeting(new Date().getHours(), user?.name ?? "")}
        description={view ? todaySummary(view) : "Wird geladen …"}
      />

      {view?.next ? <NextTask entry={view.next} horseColor={colorOf(view.next.horseId)} /> : null}

      <WeatherWarnings timeZone={data.timeZone} compact />

      {view && view.today.length > 0 ? <Timeline entries={view.today} colorOf={colorOf} /> : null}

      {view && view.tomorrow.length > 0 ? (
        <View className="gap-1">
          <SectionTitle>Morgen</SectionTitle>
          <Timeline entries={view.tomorrow} colorOf={colorOf} />
        </View>
      ) : null}

      {view && data.horses.length === 0 ? (
        <View className="gap-3">
          <Text variant="body" tone="muted">
            Sobald du ein Pferd besitzt oder reitest, steht hier, was heute für es ansteht.
          </Text>
          <Button label="Pferd anlegen" variant="outline" onPress={() => router.push("/horses/new" as Href)} />
        </View>
      ) : null}
    </Screen>
  );
}
