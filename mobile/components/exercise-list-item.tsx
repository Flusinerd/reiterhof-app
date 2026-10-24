import { ListRow } from "@/components/ui/list-row";
import type { ExerciseSummary } from "@/lib/api/training";
import { disciplineLabel, levelLabel, stepCountLabel, tagLabel } from "@/lib/tracking-exercises";

/** One row of the exercise library: level as eyebrow, title, discipline, steps and goals. */
export function ExerciseListItem({ exercise, onPress }: { exercise: ExerciseSummary; onPress: () => void }) {
  const detail = [
    disciplineLabel(exercise.discipline),
    stepCountLabel(exercise.step_count),
    ...exercise.goal_tags.map(tagLabel),
  ]
    .filter(Boolean)
    .join(" · ");
  return (
    <ListRow
      eyebrow={exercise.level ? levelLabel(exercise.level) : undefined}
      eyebrowTone="brass"
      title={exercise.title}
      detail={detail}
      accessibilityLabel={`${exercise.title}, ${levelLabel(exercise.level)}, ${stepCountLabel(exercise.step_count)}`}
      onPress={onPress}
    />
  );
}
