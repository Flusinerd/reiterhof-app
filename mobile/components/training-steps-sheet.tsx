import { NumberedSteps } from "@/components/exercise-steps";
import { Sheet, Text } from "@/components/ui";
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
    <Sheet open={open && !!exercise} onOpenChange={onOpenChange} title={exercise?.title} description="Ablauf">
      {exercise?.level ? (
        <Text variant="eyebrow" tone="brass">
          {LEVEL_LABELS[exercise.level] ?? exercise.level}
        </Text>
      ) : null}
      <NumberedSteps steps={exercise?.steps ?? []} />
    </Sheet>
  );
}
