import { useRouter, type Href } from "expo-router";
import { Sparkles } from "lucide-react-native";
import { View } from "react-native";

import { Button, Text } from "@/components/ui";
import { ListRow } from "@/components/ui/list-row";
import { horseRoutes } from "@/lib/horse-format";
import type { NextStep } from "@/lib/training-plan";

export type NextStepCardProps = {
  step: NextStep;
  horseId: string;
  horseName: string;
};

/**
 * The next step on the training tab. Without a profile it is the page's one raised block with the
 * primary "Einrichten"; planning open days is a quiet row below the week. Renders nothing for `null`.
 */
export function NextStepCard({ step, horseId, horseName }: NextStepCardProps) {
  const router = useRouter();
  if (!step) return null;

  if (step.kind === "profile") {
    return (
      <View className="gap-4 rounded-card border border-border bg-raised p-5">
        <View className="gap-1">
          <Text variant="eyebrow" tone="brass">
            Nächster Schritt
          </Text>
          <Text variant="titleLg">Profil einrichten</Text>
          <Text variant="bodySm" tone="muted">{`Fünf kurze Schritte, dann gibt es Empfehlungen für ${horseName}.`}</Text>
        </View>
        <Button label="Einrichten" fullWidth onPress={() => router.push(horseRoutes.trainingSetup(horseId) as Href)} />
      </View>
    );
  }

  return (
    <ListRow
      eyebrow="Nächster Schritt"
      eyebrowTone="brass"
      title="Woche planen"
      detail={step.openDays === 1 ? "1 Tag noch offen." : `${step.openDays} Tage noch offen.`}
      trailing={
        <Button
          label="Planen"
          variant="outline"
          size="sm"
          icon={Sparkles}
          onPress={() => router.push(`/training/week?horse=${horseId}&plan=1` as Href)}
        />
      }
    />
  );
}
