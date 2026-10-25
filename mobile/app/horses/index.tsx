import { useQueryClient } from "@tanstack/react-query";
import type { Href } from "expo-router";
import { Plus } from "lucide-react-native";
import { useMemo, useState } from "react";
import { RefreshControl } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { HorseListItem } from "@/components/horse-list-item";
import { Button, PageHeader, Screen, Section, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { horseKeys, useHorses, type Horse } from "@/lib/api/horses";
import { horseRoutes } from "@/lib/horse-format";
import { pushOnce } from "@/lib/navigation";
import { useColors } from "@/lib/theme";

/** Splits the stable's horses into own horses, horses the user rides, and the rest. */
function groupHorses(horses: Horse[]) {
  return {
    mine: horses.filter((h) => h.is_mine),
    riding: horses.filter((h) => !h.is_mine && h.i_ride),
    others: horses.filter((h) => !h.is_mine && !h.i_ride),
  };
}

/** All horses of the stable as a sub page. */
export default function HorsesRoute() {
  return <Horses back />;
}

/** Own horses, horses the user rides, the rest; also the horse tab for users with several horses. */
export function Horses({ back }: { back: boolean }) {
  const c = useColors();
  const query = useHorses();
  const queryClient = useQueryClient();
  const [refreshing, setRefreshing] = useState(false);
  const groups = useMemo(() => groupHorses(query.data ?? []), [query.data]);
  const open = (id: string) => pushOnce(horseRoutes.detail(id) as Href);

  async function refresh() {
    setRefreshing(true);
    await queryClient.invalidateQueries({ queryKey: horseKeys.all });
    setRefreshing(false);
  }

  const count = query.data?.length ?? 0;
  const sections: { title: string; horses: Horse[] }[] = [
    { title: "Meine Pferde", horses: groups.mine },
    { title: "Ich reite", horses: groups.riding },
    { title: groups.mine.length + groups.riding.length > 0 ? "Weitere Pferde" : "Pferde", horses: groups.others },
  ];

  return (
    <Screen back={back} menu={back}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={refresh} tintColor={c.primary.DEFAULT} />
      }
    >
      <PageHeader
        eyebrow="Stall"
        title="Pferde"
        action={
          <Button
            label="Pferd anlegen"
            icon={Plus}
            size="sm"
            variant="outline"
            onPress={() => pushOnce(horseRoutes.create as Href)}
          />
        }
        description={
          query.data && count > 0
            ? groups.mine.length > 0
              ? `${count} ${count === 1 ? "Pferd" : "Pferde"}, ${groups.mine.length === 1 ? "eins" : groups.mine.length} davon von dir.`
              : `${count} ${count === 1 ? "Pferd" : "Pferde"}.`
            : undefined
        }
      />

      {query.isPending ? <HorseLoading /> : null}
      {query.isError ? <HorseError error={query.error} onRetry={() => query.refetch()} /> : null}

      {sections
        .filter((s) => s.horses.length > 0)
        .map((s) => (
          <Section key={s.title} title={s.title}>
            <RowList>
              {s.horses.map((h) => (
                <HorseListItem key={h.id} horse={h} onPress={() => open(h.id)} />
              ))}
            </RowList>
          </Section>
        ))}
      {query.data && count === 0 ? (
        <Text variant="body" tone="muted">
          Noch keine Pferde.
        </Text>
      ) : null}
    </Screen>
  );
}
