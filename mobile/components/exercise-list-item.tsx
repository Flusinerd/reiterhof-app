import { ChevronRight } from "lucide-react-native";
import { View } from "react-native";

import { Badge, Icon, PressableCard, Text } from "@/components/ui";
import type { ExerciseSummary } from "@/lib/api/training";
import { disciplineLabel, levelLabel, stepCountLabel, tagLabel } from "@/lib/tracking-exercises";

/** One row of the exercise library. */
export function ExerciseListItem({ exercise, onPress }: { exercise: ExerciseSummary; onPress: () => void }) {
  return (
    <PressableCard
      shape="tile"
      className="gap-2"
      accessibilityLabel={`${exercise.title}, ${levelLabel(exercise.level)}, ${stepCountLabel(exercise.step_count)}`}
      onPress={onPress}
    >
      <View className="flex-row items-center gap-3">
        <Text variant="bodyStrong" className="flex-1">
          {exercise.title}
        </Text>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </View>
      <Text variant="secondary">
        {[disciplineLabel(exercise.discipline), stepCountLabel(exercise.step_count)].filter(Boolean).join(" · ")}
      </Text>
      <View className="flex-row flex-wrap gap-2">
        {exercise.level ? <Badge variant="primary" label={levelLabel(exercise.level)} /> : null}
        {exercise.goal_tags.map((t) => (
          <Badge key={t} variant="neutral" label={tagLabel(t)} />
        ))}
      </View>
    </PressableCard>
  );
}
