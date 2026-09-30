import { useLocalSearchParams } from "expo-router";

import { BlanketHistory } from "@/components/blanket-history";
import { PageHeader, Screen } from "@/components/ui";
import { useHorse } from "@/lib/api/horses";
import { useAuth } from "@/lib/auth";

/** Verlauf: the day states of the last 60 days for one horse (JAN-31). */
export default function BlanketHistoryScreen() {
  const { horseId } = useLocalSearchParams<{ horseId: string }>();
  const horse = useHorse(horseId);
  const { me } = useAuth();
  return (
    <Screen back>
      <PageHeader eyebrow={horse.data?.name ?? "Pferd"} title="Deckenverlauf" description="Letzte 60 Tage." />
      <BlanketHistory horseId={horseId} days={60} timeZone={me?.stable?.timezone ?? "Europe/Berlin"} />
    </Screen>
  );
}
