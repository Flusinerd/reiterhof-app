import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { ExternalLink } from "lucide-react-native";
import { ActivityIndicator, Linking, View } from "react-native";

import { NumberedSteps } from "@/components/exercise-steps";
import { Badge, Button, Icon, PageHeader, Screen, Section, Text } from "@/components/ui";
import { ListRow, RowList } from "@/components/ui/list-row";
import { errorMessage } from "@/lib/api";
import { useTrainingEquipment } from "@/lib/api/training";
import {
  exerciseIntensityLabel,
  exercisePhaseText,
  levelLabel,
  libraryLabel,
  libraryOf,
  minutesText,
  sourceItems,
  stepCountLabel,
  tagLabel,
} from "@/lib/tracking-exercises";
import { equipmentChips, equipmentList, missingText } from "@/lib/training-equipment";
import { useColors } from "@/lib/theme";
import { useExercise } from "@/lib/tracking-queries";

const CHIP_VARIANT = { ok: "neutral", shared: "info", missing: "accent" } as const;

/**
 * One exercise of the library: what it is for ("Wozu"), phase, minutes, intensity, the equipment
 * (what the stable lacks is marked), the steps ("Ablauf"), prerequisites, alternatives without
 * set-up, the next progression and the sources, which open in the browser.
 */
export default function ExerciseDetail() {
  const router = useRouter();
  const { id } = useLocalSearchParams<{ id: string }>();
  const c = useColors();
  const exercise = useExercise(id);
  const equipment = useTrainingEquipment();
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
  const go = (exerciseId: string) => router.push(`/training/exercises/${exerciseId}` as Href);
  // Phase, minutes and intensity as label and value; what the backend does not send is left out.
  const facts = [
    ["Phase", exercisePhaseText(e.phase)],
    ["Dauer", minutesText(e.minutes)],
    ["Belastung", exerciseIntensityLabel(e.intensity)],
  ].filter(([, value]) => value !== "");
  const chips = equipmentChips(e.equipment, equipment.data, libraryOf(e));
  const missing = missingText(chips);
  const requires = e.requires ?? [];
  const alternatives = e.alternatives ?? [];
  const sources = sourceItems(e.sources);

  return (
    <Screen back>
      <PageHeader
        eyebrow={[libraryLabel(libraryOf(e)), levelLabel(e.level)].filter(Boolean).join(" · ")}
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

      {e.summary ? (
        <Section title="Wozu">
          <Text variant="body">{e.summary}</Text>
        </Section>
      ) : null}

      {facts.length > 0 ? (
        <RowList>
          {facts.map(([label, value]) => (
            <View key={label} className="min-h-[48px] flex-row items-center justify-between gap-4 py-3">
              <Text variant="secondary">{label}</Text>
              <Text variant="bodyStrong" className="flex-1 text-right">
                {value}
              </Text>
            </View>
          ))}
        </RowList>
      ) : null}

      {chips.length > 0 ? (
        <View className="gap-2">
          <Text variant="eyebrow">Ausstattung</Text>
          <View className="flex-row flex-wrap gap-2">
            {chips.map((chip) => (
              <Badge key={chip.key} variant={CHIP_VARIANT[chip.status]} label={chip.text} />
            ))}
          </View>
          {missing ? (
            <Text variant="secondary" tone="accent">
              {missing}
            </Text>
          ) : null}
        </View>
      ) : null}

      <Section title="Ablauf">
        {steps.length === 0 ? (
          <Text variant="body" tone="muted">
            Noch kein Ablauf.
          </Text>
        ) : (
          <NumberedSteps steps={steps} />
        )}
      </Section>

      {requires.length > 0 ? (
        <Section title="Voraussetzungen" description="Das sollte sitzen, bevor die Übung geplant wird.">
          <RowList>
            {requires.map((r) => (
              <ListRow key={r.id} title={r.title} accessibilityLabel={`Voraussetzung: ${r.title}`} onPress={() => go(r.id)} />
            ))}
          </RowList>
        </Section>
      ) : null}

      {alternatives.length > 0 ? (
        <Section title="Alternativen ohne Aufbau">
          <RowList>
            {alternatives.map((a) => (
              <ListRow
                key={a.id}
                title={a.title}
                detail={equipmentList(a.equipment) || undefined}
                accessibilityLabel={`Alternative: ${a.title}`}
                onPress={() => go(a.id)}
              />
            ))}
          </RowList>
        </Section>
      ) : null}

      {e.next ? (
        <Section title="Als Nächstes">
          <ListRow
            eyebrow={[libraryLabel(e.next.discipline), levelLabel(e.next.level)].filter(Boolean).join(" · ")}
            title={e.next.title}
            accessibilityLabel={`Nächste Stufe: ${e.next.title}`}
            onPress={() => go(e.next!.id)}
          />
          <Text variant="secondary">Nach „Sitzt“ kommt die nächste Stufe.</Text>
        </Section>
      ) : (
        <Text variant="secondary">Letzte Stufe der Reihe.</Text>
      )}

      {sources.length > 0 ? (
        <Section title="Quellen">
          <RowList>
            {sources.map((s, i) =>
              s.url ? (
                <ListRow
                  key={`${i}-${s.url}`}
                  leading={<Icon as={ExternalLink} size={20} className="text-muted" />}
                  title={s.title}
                  detail={s.detail || undefined}
                  chevron={false}
                  accessibilityLabel={`Quelle öffnen: ${s.title}`}
                  onPress={() => void Linking.openURL(s.url!).catch(() => undefined)}
                />
              ) : (
                <ListRow key={`${i}-${s.title}`} title={s.title} />
              ),
            )}
          </RowList>
        </Section>
      ) : null}
    </Screen>
  );
}
