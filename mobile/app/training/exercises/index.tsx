import { useRouter, type Href } from "expo-router";
import { useMemo, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { ExerciseListItem } from "@/components/exercise-list-item";
import { Button, PageHeader, Pill, Screen, Section, Text } from "@/components/ui";
import {
  NO_FILTER,
  applyFilter,
  disciplineLabel,
  filterOptions,
  hasFilter,
  levelLabel,
  tagLabel,
  toggleFilter,
  type ExerciseFilter,
} from "@/lib/tracking-exercises";
import { useExerciseList } from "@/lib/tracking-queries";
import { errorMessage } from "@/lib/api";
import { colors } from "@/lib/theme";

/** Exercise library (JAN-58): browse and filter by discipline, level and goal. */
export default function ExerciseLibrary() {
  const router = useRouter();
  const list = useExerciseList();
  const [filter, setFilter] = useState<ExerciseFilter>(NO_FILTER);

  const all = list.data ?? [];
  const options = useMemo(() => filterOptions(all), [all]);
  const shown = useMemo(() => applyFilter(all, filter), [all, filter]);

  return (
    <Screen back>
      <PageHeader title="Übungen" description="Für Halle und Platz." />

      {list.isPending ? (
        <ActivityIndicator accessibilityLabel="Lädt" color={colors.primary.DEFAULT} />
      ) : list.isError ? (
        <View className="gap-3">
          <Text variant="body" tone="danger" accessibilityRole="alert">
            {errorMessage(list.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void list.refetch()} />
        </View>
      ) : (
        <>
          {options.disciplines.length > 1 ? (
            <FilterGroup title="Disziplin">
              {options.disciplines.map((d) => (
                <Pill
                  key={d}
                  label={disciplineLabel(d)}
                  selected={filter.discipline === d}
                  onPress={() => setFilter(toggleFilter(filter, "discipline", d))}
                />
              ))}
            </FilterGroup>
          ) : null}
          {options.levels.length > 1 ? (
            <FilterGroup title="Niveau">
              {options.levels.map((l) => (
                <Pill
                  key={l}
                  label={levelLabel(l)}
                  selected={filter.level === l}
                  onPress={() => setFilter(toggleFilter(filter, "level", l))}
                />
              ))}
            </FilterGroup>
          ) : null}
          {options.tags.length > 0 ? (
            <FilterGroup title="Ziel">
              {options.tags.map((t) => (
                <Pill
                  key={t}
                  label={tagLabel(t)}
                  selected={filter.tag === t}
                  onPress={() => setFilter(toggleFilter(filter, "tag", t))}
                />
              ))}
            </FilterGroup>
          ) : null}

          <Section
            title={`${shown.length} ${shown.length === 1 ? "Übung" : "Übungen"}`}
            action={hasFilter(filter) ? <Button label="Zurücksetzen" size="sm" variant="ghost" onPress={() => setFilter(NO_FILTER)} /> : undefined}
          >
            {shown.length === 0 ? (
              <Text variant="body" tone="muted">
                {all.length === 0 ? "Noch keine Übungen." : "Keine Treffer."}
              </Text>
            ) : (
              shown.map((e) => (
                <ExerciseListItem key={e.id} exercise={e} onPress={() => router.push(`/training/exercises/${e.id}` as Href)} />
              ))
            )}
          </Section>
        </>
      )}
    </Screen>
  );
}

/** A filter row: small label (not a section heading, three of them sit close together) and pills. */
function FilterGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View className="-mb-2 gap-2">
      <Text variant="label">{title}</Text>
      <View className="flex-row flex-wrap gap-2">{children}</View>
    </View>
  );
}
