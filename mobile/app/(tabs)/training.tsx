import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";

export default function Training() {
  return (
    <Screen>
      <Hero
        eyebrow="Reiterhof"
        title="Training"
        description="Die Empfehlung „Was heute?“ folgt in Meilenstein 6."
      />
      <SectionLabel>Einheiten</SectionLabel>
      <Card>
        <Text variant="secondary">Noch keine Einheiten.</Text>
      </Card>
    </Screen>
  );
}
