import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { ArrowRight } from "lucide-react-native";
import { ActivityIndicator, View } from "react-native";

import { Badge, Button, Card, Divider, Icon, PageHeader, PressableCard, Screen, Section, Text } from "@/components/ui";
import { errorMessage } from "@/lib/api";
import { disciplineLabel, levelLabel, stepCountLabel, tagLabel } from "@/lib/tracking-exercises";
import { colors } from "@/lib/theme";
import { useExercise } from "@/lib/tracking-queries";

/** One exercise of the library: the steps ("Ablauf") and the next progression. */
export default function ExerciseDetail() {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const exercise = useExercise(id);
  const e = exercise.data;

  if (exercise.isPending) {
    return (
      <Screen back>
        <ActivityIndicator accessibilityLabel="Lädt" color={colors.primary.DEFAULT} />
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
        <Card padded={false}>
          {steps.map((step, i) => (
            <View key={i}>
              {i > 0 ? <Divider /> : null}
              <View className="flex-row gap-3 p-5">
                <View className="h-7 w-7 items-center justify-center rounded-pill bg-primary-soft">
                  <Text variant="caption" className="font-sans-semibold text-primary-deep">
                    {i + 1}
                  </Text>
                </View>
                <Text variant="body" className="flex-1">
                  {step}
                </Text>
              </View>
            </View>
          ))}
        </Card>
      )}
      </Section>

      {e.next ? (
        <Section title="Als Nächstes">
          <PressableCard
            shape="tile"
            className="flex-row items-center gap-3"
            accessibilityLabel={`Nächste Stufe: ${e.next.title}`}
            onPress={() => router.push(`/training/exercises/${e.next!.id}` as Href)}
          >
            <View className="flex-1 gap-1">
              <Text variant="bodyStrong">{e.next.title}</Text>
              <Text variant="secondary">
                {[disciplineLabel(e.next.discipline), levelLabel(e.next.level)].filter(Boolean).join(" · ")}
              </Text>
            </View>
            <Icon as={ArrowRight} size={20} className="text-muted" />
          </PressableCard>
          <Text variant="secondary">Nach „Sitzt“ kommt die nächste Stufe.</Text>
        </Section>
      ) : (
        <Text variant="secondary">Letzte Stufe der Reihe.</Text>
      )}
    </Screen>
  );
}
