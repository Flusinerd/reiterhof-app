import { Card, Hero, Screen, SectionLabel, Text } from "@/components/ui";

export default function Requests() {
  return (
    <Screen>
      <Hero
        eyebrow="Reiterhof"
        title="Anfragen"
        description="Anfragen an die Stallgasse folgen in Meilenstein 4."
      />
      <SectionLabel>Offen</SectionLabel>
      <Card>
        <Text variant="secondary">Noch keine Anfragen.</Text>
      </Card>
    </Screen>
  );
}
