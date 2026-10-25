import type { Href } from "expo-router";
import { Shirt } from "lucide-react-native";

import { Button, PageHeader } from "@/components/ui";
import { pushOnce } from "@/lib/navigation";
import { blanketHero, type BlanketCheck } from "@/lib/reminders";

/**
 * The head of the reminder center: tonight's blanket check with time and progress. The link to the
 * blanket list is the one primary button while horses are still open, otherwise a quiet outline.
 */
export function ReminderHead({ check }: { check: BlanketCheck | undefined }) {
  if (!check) {
    return <PageHeader eyebrow="Deckencheck" title="Erinnerungen" description="Wird geladen ..." />;
  }
  const head = blanketHero(check);
  return (
    <PageHeader
      eyebrow={`Deckencheck · ${check.time} Uhr`}
      title="Erinnerungen"
      value={head.value === "–" ? undefined : head.value}
      valueSize="sm"
      unit={head.unit || undefined}
      description={head.description}
    >
      <Button
        label="Zu den Decken"
        icon={Shirt}
        variant={check.state === "due" ? "primary" : "outline"}
        size="sm"
        onPress={() => pushOnce("/blankets" as Href)}
      />
    </PageHeader>
  );
}
