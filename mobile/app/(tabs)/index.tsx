import { useQuery } from "@tanstack/react-query";
import { ActivityIndicator } from "react-native";

import { PresenceTile } from "@/components/presence-tile";
import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";
import { fetchHealth } from "@/lib/api";

export default function Home() {
  const health = useQuery({ queryKey: ["health"], queryFn: fetchHealth });

  return (
    <Screen>
      <Hero
        eyebrow="Reiterhof"
        title="Start"
        description="Wetter und Deckenempfehlung folgen in Meilenstein 3."
      />
      <SectionLabel>Anwesenheit</SectionLabel>
      <PresenceTile />
      <SectionLabel>Verbindung</SectionLabel>
      <Card className="flex-row items-center justify-between">
        <Text variant="body">Server</Text>
        {health.isPending ? (
          <ActivityIndicator />
        ) : (
          <Text variant="bodyStrong" tone={health.isError ? "danger" : "primary"}>
            {health.isError ? "nicht erreichbar" : health.data.status}
          </Text>
        )}
      </Card>
    </Screen>
  );
}
