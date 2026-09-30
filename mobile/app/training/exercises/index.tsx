import { useRouter, type Href } from "expo-router";
import { useMemo, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { ExerciseListItem } from "@/components/exercise-list-item";
import { Button, Hero, Pill, Screen, SectionLabel, Text } from "@/components/ui";
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
      <Hero tone="soft" title="Übungen" description="Für Halle und Platz." />

      {list.isPending ? (
        <ActivityIndicator accessibilityLabel="Lädt" />
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

          <SectionLabel
            action={hasFilter(filter) ? <Button label="Zurücksetzen" size="sm" variant="ghost" onPress={() => setFilter(NO_FILTER)} /> : undefined}
          >
            {`${shown.length} ${shown.length === 1 ? "Übung" : "Übungen"}`}
          </SectionLabel>
          {shown.length === 0 ? (
            <Text variant="body" tone="muted">
              {all.length === 0 ? "Noch keine Übungen." : "Keine Treffer."}
            </Text>
          ) : (
            <View className="gap-3">
              {shown.map((e) => (
                <ExerciseListItem key={e.id} exercise={e} onPress={() => router.push(`/training/exercises/${e.id}` as Href)} />
              ))}
            </View>
          )}
        </>
      )}
    </Screen>
  );
}

function FilterGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View className="gap-2">
      <SectionLabel>{title}</SectionLabel>
      <View className="flex-row flex-wrap gap-2">{children}</View>
    </View>
  );
}
