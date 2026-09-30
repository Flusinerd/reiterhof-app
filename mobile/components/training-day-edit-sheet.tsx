import { Sparkles } from "lucide-react-native";
import { useEffect, useState } from "react";
import { ScrollView, View } from "react-native";

import { Button, Input, Pill, Sheet, Text } from "@/components/ui";
import type { ExerciseSummary } from "@/lib/api/training";
import {
  ACTIVITIES,
  DEFAULT_MINUTES,
  MINUTE_CHOICES,
  activityLabel,
  exerciseLibrary,
  formatDayLong,
  minuteChoices,
  type Activity,
} from "@/lib/training";
import type { DayEdit } from "@/lib/training-plan";

export type DayEditSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  date: string;
  /** "draft": the change only affects the proposal; "planned": it is stored right away. */
  mode: "draft" | "planned";
  value: DayEdit;
  /** Activities the profile allows (mode is not off). */
  allowed: readonly Activity[];
  /** "Ich" and the horse's riders; in draft mode only used to name the claimed person. */
  people: { id: string; name: string }[];
  exercises: readonly ExerciseSummary[];
  discipline: string;
  busy: boolean;
  reproposing: boolean;
  error: string | null;
  /** Saved days that nobody planned or claimed have nothing to release. */
  canRemove?: boolean;
  onSave: (edit: DayEdit) => void;
  /** Gets the state shown in the sheet, so that the parent can exclude what is shown. */
  onRepropose: (current: DayEdit) => void;
  /** Draft: remove the day from the draft. Planned: release the day (status open). */
  onRemove: () => void;
};

/**
 * Edits one day of the week: activity or rest day, minutes, focus, library exercise and (for
 * stored days) the person. In draft mode nothing is stored before "Übernehmen" in the week view.
 */
export function DayEditSheet({
  open,
  onOpenChange,
  date,
  mode,
  value,
  allowed,
  people,
  exercises,
  discipline,
  busy,
  reproposing,
  error,
  canRemove = true,
  onSave,
  onRepropose,
  onRemove,
}: DayEditSheetProps) {
  const [edit, setEdit] = useState<DayEdit>(value);
  // The parent builds a new value object on every render; compare by content.
  const valueKey = JSON.stringify(value);
  useEffect(() => {
    setEdit(value);
  }, [open, date, valueKey]);

  const rest = edit.status === "rest" || edit.activity === null;
  const choices = ACTIVITIES.filter((a) => allowed.includes(a) || a === edit.activity);
  const library = edit.activity ? exerciseLibrary(edit.activity, discipline) : "";
  const libraryExercises = library ? exercises.filter((e) => e.discipline === library) : [];
  const exerciseItems =
    edit.exercise && !libraryExercises.some((e) => e.id === edit.exercise!.id)
      ? [{ id: edit.exercise.id, title: edit.exercise.title }, ...libraryExercises]
      : libraryExercises;
  const person = people.find((p) => p.id === edit.userId);
  const nothingChosen = edit.status === "planned" && edit.activity === null;

  const pickActivity = (activity: Activity) =>
    setEdit((e) => ({
      ...e,
      status: "planned",
      activity,
      minutes: MINUTE_CHOICES[activity].includes(e.minutes) ? e.minutes : DEFAULT_MINUTES[activity],
      exercise:
        e.activity && exerciseLibrary(e.activity, discipline) === exerciseLibrary(activity, discipline) ? e.exercise : null,
    }));
  const pickRest = () => setEdit((e) => ({ ...e, status: "rest", activity: null, minutes: 0, focus: "", exercise: null }));

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={formatDayLong(date)}
      description={mode === "draft" ? "Gilt erst beim Übernehmen." : "Gilt sofort nach dem Speichern."}
    >
      <View className="gap-2">
        <Text variant="label">Was</Text>
        <View className="flex-row flex-wrap gap-2">
          {choices.map((a) => (
            <Pill key={a} label={activityLabel(a)} selected={edit.activity === a && !rest} onPress={() => pickActivity(a)} />
          ))}
          <Pill label="Ruhetag" selected={edit.status === "rest"} onPress={pickRest} />
        </View>
      </View>

      {!rest && edit.activity ? (
        <>
          <View className="gap-2">
            <Text variant="label">Wie lange</Text>
            <View className="flex-row flex-wrap gap-2">
              {minuteChoices(edit.activity, edit.minutes).map((m) => (
                <Pill key={m} label={`${m} Min.`} selected={edit.minutes === m} onPress={() => setEdit((e) => ({ ...e, minutes: m }))} />
              ))}
            </View>
          </View>

          <View className="gap-2">
            <Text variant="label">Schwerpunkt</Text>
            <Input
              value={edit.focus}
              onChangeText={(focus) => setEdit((e) => ({ ...e, focus }))}
              maxLength={80}
              placeholder="z. B. Übergänge Schritt-Trab"
              accessibilityLabel="Schwerpunkt"
            />
          </View>

          {library !== "" ? (
            <View className="gap-2">
              <Text variant="label">Übung</Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2">
                <Pill label="Keine" selected={edit.exercise === null} onPress={() => setEdit((e) => ({ ...e, exercise: null }))} />
                {exerciseItems.map((x) => (
                  <Pill
                    key={x.id}
                    label={x.title}
                    selected={edit.exercise?.id === x.id}
                    onPress={() => setEdit((e) => ({ ...e, exercise: { id: x.id, title: x.title } }))}
                  />
                ))}
              </ScrollView>
            </View>
          ) : null}
        </>
      ) : null}

      {mode === "planned" && !rest ? (
        <View className="gap-2">
          <Text variant="label">Wer</Text>
          <View className="flex-row flex-wrap gap-2">
            <Pill label="Niemand" selected={edit.userId === null} onPress={() => setEdit((e) => ({ ...e, userId: null }))} />
            {people.map((p) => (
              <Pill key={p.id} label={p.name} selected={edit.userId === p.id} onPress={() => setEdit((e) => ({ ...e, userId: p.id }))} />
            ))}
          </View>
        </View>
      ) : null}
      {mode === "draft" && !rest && person ? <Text variant="secondary">Bleibt eingetragen: {person.name}</Text> : null}

      {rest && person ? (
        <Text variant="secondary">
          {mode === "draft"
            ? `Ruhetag wird nicht gespeichert, ${person.name} bleibt eingetragen.`
            : `Ruhetag entfernt die Eintragung von ${person.name}.`}
        </Text>
      ) : null}

      {error ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      <Button
        label={mode === "draft" ? "Fertig" : "Speichern"}
        fullWidth
        loading={busy}
        disabled={reproposing || nothingChosen}
        onPress={() => onSave(edit)}
      />
      <Button
        label="Neu vorschlagen"
        variant="outline"
        icon={Sparkles}
        fullWidth
        loading={reproposing}
        disabled={busy}
        onPress={() => onRepropose(edit)}
      />
      {canRemove ? (
        <Button
          label={mode === "draft" ? "Aus Entwurf entfernen" : "Freigeben"}
          variant="ghost"
          fullWidth
          disabled={busy || reproposing}
          onPress={onRemove}
        />
      ) : null}
    </Sheet>
  );
}
