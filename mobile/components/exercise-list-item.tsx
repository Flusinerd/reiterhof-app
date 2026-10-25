import { ListRow } from "@/components/ui/list-row";
import type { ExerciseSummary } from "@/lib/api/training";
import { exerciseRowDetail, levelLabel, stepCountLabel } from "@/lib/tracking-exercises";
import { MISSING_LABEL, isAvailable } from "@/lib/training-equipment";

/**
 * One row of the exercise library: level as eyebrow, title, then library, minutes, steps and goals.
 * An exercise the stable cannot set up (`available` false, only with a horse in context) is dimmed
 * and says "Ausstattung fehlt".
 */
export function ExerciseListItem({ exercise, onPress }: { exercise: ExerciseSummary; onPress: () => void }) {
  const available = isAvailable(exercise);
  return (
    <ListRow
      className={available ? undefined : "opacity-60"}
      eyebrow={exercise.level ? levelLabel(exercise.level) : undefined}
      eyebrowTone="brass"
      title={exercise.title}
      detail={exerciseRowDetail(exercise)}
      accessibilityLabel={[exercise.title, levelLabel(exercise.level), stepCountLabel(exercise.step_count), available ? "" : MISSING_LABEL]
        .filter(Boolean)
        .join(", ")}
      onPress={onPress}
    />
  );
}
