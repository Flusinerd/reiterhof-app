import { useRouter, type Href } from "expo-router";
import { Sparkles } from "lucide-react-native";

import { Button, Card, Text } from "@/components/ui";
import { horseRoutes } from "@/lib/horse-format";
import type { NextStep } from "@/lib/training-plan";

export type NextStepCardProps = {
  step: NextStep;
  horseId: string;
  horseName: string;
};

/** "Nächster Schritt" on the training tab: set up the profile or plan the open days. Renders nothing for `null`. */
export function NextStepCard({ step, horseId, horseName }: NextStepCardProps) {
  const router = useRouter();
  if (!step) return null;

  if (step.kind === "profile") {
    return (
      <Card className="gap-3">
        <Text variant="label">Nächster Schritt</Text>
        <Text variant="bodyStrong">Profil einrichten</Text>
        <Text variant="secondary">{`Fünf kurze Schritte, dann gibt es Empfehlungen für ${horseName}.`}</Text>
        <Button
          label="Einrichten"
          className="self-start"
          onPress={() => router.push(horseRoutes.trainingSetup(horseId) as Href)}
        />
      </Card>
    );
  }

  return (
    <Card className="gap-3">
      <Text variant="label">Nächster Schritt</Text>
      <Text variant="bodyStrong">Woche planen</Text>
      <Text variant="secondary">
        {step.openDays === 1 ? "1 Tag noch offen." : `${step.openDays} Tage noch offen.`}
      </Text>
      <Button
        label="Planen"
        variant="secondary"
        icon={Sparkles}
        className="self-start"
        onPress={() => router.push(`/training/week?horse=${horseId}&plan=1` as Href)}
      />
    </Card>
  );
}
