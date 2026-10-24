import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { ActivityIndicator, View } from "react-native";

import { NumberedSteps } from "@/components/exercise-steps";
import { Badge, Button, PageHeader, Screen, Section, Text } from "@/components/ui";
import { ListRow } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { disciplineLabel, levelLabel, stepCountLabel, tagLabel } from "@/lib/tracking-exercises";
import { useColors } from "@/lib/theme";
import { useExercise } from "@/lib/tracking-queries";

/** One exercise of the library: the steps ("Ablauf") and the next progression. */
export default function ExerciseDetail() {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const c = useColors();
  const exercise = useExercise(id);
  const e = exercise.data;

  if (exercise.isPending) {
    return (
      <Screen back>
        <ActivityIndicator accessibilityLabel="Lädt" color={c.primary.DEFAULT} />
      </Screen>
    );
  }
  if (exercise.isError || !e) {
    return (
      <Screen back>
        <Text variant="body" tone="danger" accessibilityRole="alert">
          {exercise.error ? errorMessage(exercise.error) : "Übung nicht gefunden."}
        </Text>
        <Button label="Erneut versuchen" variant="outline" onPress={() => void exercise.refetch()} />
      </Screen>
    );
  }

  const steps = e.steps ?? [];
  return (
    <Screen back>
      <PageHeader
        eyebrow={[disciplineLabel(e.discipline), levelLabel(e.level)].filter(Boolean).join(" · ")}
        title={e.title}
        description={stepCountLabel(steps.length || e.step_count)}
      >
        {e.goal_tags.length > 0 ? (
          <View className="flex-row flex-wrap gap-2">
            {e.goal_tags.map((t) => (
              <Badge key={t} variant="primary" label={tagLabel(t)} />
            ))}
          </View>
        ) : null}
      </PageHeader>

      <Section title="Ablauf">
        {steps.length === 0 ? (
          <Text variant="body" tone="muted">
            Noch kein Ablauf.
          </Text>
        ) : (
          <NumberedSteps steps={steps} />
        )}
      </Section>

      {e.next ? (
        <Section title="Als Nächstes">
          <ListRow
            eyebrow={[disciplineLabel(e.next.discipline), levelLabel(e.next.level)].filter(Boolean).join(" · ")}
            title={e.next.title}
            accessibilityLabel={`Nächste Stufe: ${e.next.title}`}
            onPress={() => router.push(`/training/exercises/${e.next!.id}` as Href)}
          />
          <Text variant="secondary">Nach „Sitzt“ kommt die nächste Stufe.</Text>
        </Section>
      ) : (
        <Text variant="secondary">Letzte Stufe der Reihe.</Text>
      )}
    </Screen>
  );
}
