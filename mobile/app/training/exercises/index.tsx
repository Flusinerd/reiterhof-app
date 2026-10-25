import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { useMemo, useState } from "react";
import { ActivityIndicator, ScrollView, View } from "react-native";

import { ExerciseListItem } from "@/components/exercise-list-item";
import { Button, PageHeader, Pill, Screen, Section, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { useFocuses } from "@/lib/api/training";
import {
  NO_FILTER,
  applyFilter,
  disciplineLabel,
  filterOptions,
  hasFilter,
  levelLabel,
  phaseLabel,
  phaseOptions,
  tagLabel,
  toggleFilter,
  type ExerciseFilter,
} from "@/lib/tracking-exercises";
import { useExerciseList } from "@/lib/tracking-queries";
import { errorMessage } from "@/lib/api";
import { useColors } from "@/lib/theme";

/**
 * Exercise library (JAN-58): browse and filter by focus, phase (Lösen, Arbeiten, Entspannen),
 * library, level and goal. The focus is matched by the server (it knows the tags of a focus), the
 * other filters work on the list. With a horse (`?horse=`) exercises the stable cannot set up are
 * dimmed.
 */
export default function ExerciseLibrary() {
  const router = useRouter();
  const { horse } = useLocalSearchParams<{ horse?: string }>();
  const c = useColors();
  const [filter, setFilter] = useState<ExerciseFilter>(NO_FILTER);
  const focuses = useFocuses();
  // The filter pills come from the whole list; a chosen focus asks the server for its own list.
  const list = useExerciseList({ horse });
  const focused = useExerciseList({ horse, focus: filter.focus }, !!filter.focus);

  const all = list.data ?? [];
  const options = useMemo(() => filterOptions(all), [all]);
  const phases = useMemo(() => phaseOptions(all), [all]);
  const base = filter.focus ? (focused.data ?? []) : all;
  const shown = useMemo(() => applyFilter(base, filter), [base, filter]);
  const entries = focuses.data ?? [];
  const waiting = !!filter.focus && focused.isPending;

  return (
    <Screen back>
      <PageHeader eyebrow="Training" title="Übungen" description="Nach Schwerpunkt, Phase und Niveau." />

      {list.isPending ? (
        <ActivityIndicator accessibilityLabel="Lädt" color={c.primary.DEFAULT} />
      ) : list.isError ? (
        <View className="gap-3">
          <Text variant="body" tone="danger" accessibilityRole="alert">
            {errorMessage(list.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void list.refetch()} />
        </View>
      ) : (
        <>
          {entries.length > 0 ? (
            <FilterGroup title="Schwerpunkt">
              {entries.map((e) => (
                <Pill
                  key={e.key}
                  label={e.label}
                  selected={filter.focus === e.key}
                  onPress={() => setFilter(toggleFilter(filter, "focus", e.key))}
                />
              ))}
            </FilterGroup>
          ) : null}
          {phases.length > 0 ? (
            <FilterGroup title="Phase">
              {phases.map((p) => (
                <Pill
                  key={p}
                  label={phaseLabel(p)}
                  selected={filter.phase === p}
                  onPress={() => setFilter(toggleFilter(filter, "phase", p))}
                />
              ))}
            </FilterGroup>
          ) : null}
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
            title={waiting ? "Übungen" : `${shown.length} ${shown.length === 1 ? "Übung" : "Übungen"}`}
            action={hasFilter(filter) ? <Button label="Zurücksetzen" size="sm" variant="ghost" onPress={() => setFilter(NO_FILTER)} /> : undefined}
          >
            {waiting ? (
              <ActivityIndicator accessibilityLabel="Lädt" color={c.primary.DEFAULT} className="self-start" />
            ) : filter.focus && focused.isError ? (
              <View className="gap-3">
                <Text variant="body" tone="danger" accessibilityRole="alert">
                  {errorMessage(focused.error)}
                </Text>
                <Button label="Erneut versuchen" variant="outline" onPress={() => void focused.refetch()} />
              </View>
            ) : shown.length === 0 ? (
              <Text variant="body" tone="muted">
                {all.length === 0 ? "Noch keine Übungen." : "Keine Treffer."}
              </Text>
            ) : (
              <RowList>
                {shown.map((e) => (
                  <ExerciseListItem key={e.id} exercise={e} onPress={() => router.push(`/training/exercises/${e.id}` as Href)} />
                ))}
              </RowList>
            )}
          </Section>
        </>
      )}
    </Screen>
  );
}

/** A filter row: small eyebrow (several of them sit close together) and one line of pills. */
function FilterGroup({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View className="-mb-2 gap-2">
      <Text variant="eyebrow">{title}</Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2">
        {children}
      </ScrollView>
    </View>
  );
}
