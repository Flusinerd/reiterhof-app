import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";

export default function Horses() {
  return (
    <Screen>
      <Hero
        eyebrow="Reiterhof"
        title="Pferde"
        description="Pferdeakte und Notfallkarte folgen in Meilenstein 5."
      />
      <SectionLabel>Stall</SectionLabel>
      <Card>
        <Text variant="secondary">Noch keine Pferde.</Text>
      </Card>
    </Screen>
  );
}
