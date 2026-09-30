import { router, type Href } from "expo-router";
import { Shirt } from "lucide-react-native";

import { Button, Hero } from "@/components/ui";
import { blanketHero, type BlanketCheck } from "@/lib/reminders";

/** The "Deckencheck" hero of the reminder center: tonight's time, progress, link to the blanket tab. */
export function ReminderHero({ check }: { check: BlanketCheck | undefined }) {
  if (!check) {
    return <Hero eyebrow="Deckencheck" title="Erinnerungen" description="Wird geladen ..." />;
  }
  const hero = blanketHero(check);
  return (
    <Hero
      eyebrow={`Deckencheck · ${check.time} Uhr`}
      title={hero.value === "–" ? "Erinnerungen" : undefined}
      value={hero.value === "–" ? undefined : hero.value}
      valueSize="sm"
      unit={hero.unit || undefined}
      description={hero.description}
    >
      <Button
        label="Zu den Decken"
        icon={Shirt}
        variant="secondary"
        fullWidth
        onPress={() => router.push("/blankets" as Href)}
      />
    </Hero>
  );
}
