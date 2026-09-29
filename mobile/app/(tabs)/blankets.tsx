import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";

export default function Blankets() {
  return (
    <Screen>
      <Hero
        eyebrow="Reiterhof"
        title="Decken"
        description="Deckenplan und Tagesstatus folgen in Meilenstein 3."
      />
      <SectionLabel>Übersicht</SectionLabel>
      <Card>
        <Text variant="secondary">Noch keine Einträge.</Text>
      </Card>
    </Screen>
  );
}
