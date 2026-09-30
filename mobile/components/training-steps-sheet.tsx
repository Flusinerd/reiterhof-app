import { View } from "react-native";

import { Badge, Sheet, Text } from "@/components/ui";
import type { ExerciseSummary } from "@/lib/api/training";

const LEVEL_LABELS: Record<string, string> = {
  beginner: "Einsteiger",
  intermediate: "Fortgeschritten",
  advanced: "Erfahren",
};

/** "Ablauf": the steps of a library exercise in a bottom sheet. */
export function StepsSheet({
  exercise,
  open,
  onOpenChange,
}: {
  exercise: ExerciseSummary | null | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Sheet open={open && !!exercise} onOpenChange={onOpenChange} title={exercise?.title} description="Ablauf der Übung">
      {exercise?.level ? (
        <Badge variant="primary" label={LEVEL_LABELS[exercise.level] ?? exercise.level} />
      ) : null}
      {(exercise?.steps ?? []).map((step, i) => (
        <View key={i} className="flex-row gap-3">
          <View className="h-7 w-7 items-center justify-center rounded-pill bg-primary-soft">
            <Text variant="caption" className="font-sans-semibold text-primary-deep">
              {i + 1}
            </Text>
          </View>
          <Text variant="body" className="flex-1">
            {step}
          </Text>
        </View>
      ))}
    </Sheet>
  );
}
