import type { Href } from "expo-router";
import { Megaphone } from "lucide-react-native";
import type { ReactNode } from "react";
import { useState } from "react";
import { View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { ObservationCard } from "@/components/observation-card";
import { Button, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { useObservations, type Observation } from "@/lib/api/observations";
import { pushOnce } from "@/lib/navigation";
import { newObservationRoute, openCount } from "@/lib/observations";

/** Observations shown before "Ältere anzeigen". */
const VISIBLE = 5;

/**
 * Observations ("Auffälligkeiten", JAN-52) in the horse record: button "Melden", the newest
 * reports with status badges "Beobachten"/"Erledigt".
 *
 * `renderActions` is the slot for extra buttons of other features per observation, e.g. the reha
 * feature's "In Reha-Plan umwandeln"; return null for observations without an action.
 */
export function HorseObservations({
  horseId,
  renderActions,
}: {
  horseId: string;
  renderActions?: (observation: Observation) => ReactNode;
}) {
  const query = useObservations(horseId);
  const [all, setAll] = useState(false);

  const list = query.data;
  const open = list ? openCount(list) : 0;
  const shown = list ? (all ? list : list.slice(0, VISIBLE)) : [];

  return (
    <View className="gap-3">
      <Button
        label="Melden"
        icon={Megaphone}
        variant="outline"
        size="sm"
        onPress={() => pushOnce(newObservationRoute(horseId) as Href)}
      />
      {query.isError ? (
        <HorseError error={query.error} onRetry={() => query.refetch()} />
      ) : !list ? (
        <HorseLoading />
      ) : list.length === 0 ? (
        <Text variant="body" tone="muted">
          Keine Auffälligkeiten.
        </Text>
      ) : (
        <>
          <Text variant="secondary">
            {open === 0 ? "Nichts zu beobachten." : open === 1 ? "1 Auffälligkeit wird beobachtet." : `${open} Auffälligkeiten werden beobachtet.`}
          </Text>
          <RowList>
            {shown.map((o) => (
              <ObservationCard key={o.id} observation={o} actions={renderActions?.(o)} />
            ))}
          </RowList>
          {list.length > VISIBLE ? (
            <Button
              label={all ? "Weniger anzeigen" : `Ältere anzeigen (${list.length - VISIBLE})`}
              variant="ghost"
              size="sm"
              onPress={() => setAll((v) => !v)}
            />
          ) : null}
        </>
      )}
    </View>
  );
}
