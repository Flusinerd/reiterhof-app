import { useLocalSearchParams } from "expo-router";

import { BlanketHistory } from "@/components/blanket-history";
import { PageHeader, Screen } from "@/components/ui";
import { useHistory } from "@/lib/api/blankets";
import { useHorse } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";
import { usePullToRefresh } from "@/lib/use-pull-to-refresh";

/** Verlauf: the day states of the last 60 days for one horse (JAN-31). */
export default function BlanketHistoryScreen() {
  const { horseId } = useLocalSearchParams<{ horseId: string }>();
  const horse = useHorse(horseId);
  const history = useHistory(horseId, 60); // the same query as in the list below
  const refresh = usePullToRefresh([history, horse]);
  const { me } = useAuth();
  return (
    <Screen back refreshControl={refresh}>
      <PageHeader eyebrow={horse.data?.name ?? "Pferd"} title="Deckenverlauf" description="Letzte 60 Tage." />
      <BlanketHistory horseId={horseId} days={60} timeZone={me?.stable?.timezone ?? "Europe/Berlin"} />
    </Screen>
  );
}
