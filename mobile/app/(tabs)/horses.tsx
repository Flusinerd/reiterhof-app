import { useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { Plus } from "lucide-react-native";
import { useMemo, useState } from "react";
import { RefreshControl, View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseListItem } from "@/components/horse-list-item";
import { Button, Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { horseKeys, useHorses, type Horse } from "@/lib/api/horses";
import { horseRoutes } from "@/lib/horse-format";
import { colors } from "@/lib/theme";

/** Splits the stable's horses into own horses, horses the user rides, and the rest. */
function groupHorses(horses: Horse[]) {
  return {
    mine: horses.filter((h) => h.is_mine),
    riding: horses.filter((h) => !h.is_mine && h.i_ride),
    others: horses.filter((h) => !h.is_mine && !h.i_ride),
  };
}

export default function Horses() {
  const query = useHorses();
  const queryClient = useQueryClient();
  const [refreshing, setRefreshing] = useState(false);
  const groups = useMemo(() => groupHorses(query.data ?? []), [query.data]);
  const open = (id: string) => router.push(horseRoutes.detail(id) as Href);

  async function refresh() {
    setRefreshing(true);
    await queryClient.invalidateQueries({ queryKey: horseKeys.all });
    setRefreshing(false);
  }

  const count = query.data?.length ?? 0;
  const sections: { title: string; horses: Horse[] }[] = [
    { title: "Meine Pferde", horses: groups.mine },
    { title: "Ich reite", horses: groups.riding },
    { title: groups.mine.length + groups.riding.length > 0 ? "Weitere Pferde im Stall" : "Pferde im Stall", horses: groups.others },
  ];

  return (
    <Screen
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={refresh} tintColor={colors.primary.DEFAULT} />
      }
    >
      <Hero
        eyebrow="Stall"
        value={query.data ? String(count) : "–"}
        unit={count === 1 ? "Pferd" : "Pferde"}
        description={
          query.data
            ? groups.mine.length > 0
              ? `Davon ${groups.mine.length === 1 ? "eins" : groups.mine.length} von dir.`
              : "Tippe auf ein Pferd für die Pferdeakte."
            : "Pferde werden geladen."
        }
      >
        <Button
          label="Pferd anlegen"
          icon={Plus}
          variant="secondary"
          onPress={() => router.push(horseRoutes.create as Href)}
        />
      </Hero>

      {query.isPending ? <HorseLoading /> : null}
      {query.isError ? <HorseError error={query.error} onRetry={() => query.refetch()} /> : null}
      {query.data && count === 0 ? (
        <Card>
          <Text variant="secondary">Noch keine Pferde im Stall. Lege das erste Pferd an.</Text>
        </Card>
      ) : null}

      {sections
        .filter((s) => s.horses.length > 0)
        .map((s) => (
          <View key={s.title} className="gap-3">
            <SectionLabel>{s.title}</SectionLabel>
            {s.horses.map((h) => (
              <HorseListItem key={h.id} horse={h} onPress={() => open(h.id)} />
            ))}
          </View>
        ))}
    </Screen>
  );
}
