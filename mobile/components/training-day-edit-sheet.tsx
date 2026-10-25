import { Check, ChevronDown, ChevronUp, Sparkles } from "lucide-react-native";
import { useEffect, useState } from "react";
import { Pressable, ScrollView, View } from "react-native";

import { Button, Icon, Input, Pill, Sheet, Text } from "@/components/ui";
import type { ExerciseSummary } from "@/lib/api/training";
import { libraryOf, phaseLabel } from "@/lib/tracking-exercises";
import {
  ACTIVITIES,
  activityLabel,
  exerciseLibraries,
  formatDayLong,
  minuteChoices,
  type Activity,
} from "@/lib/training";
import { LOCATIONS, MISSING_LABEL, SETUP_CHOICES, equipmentOkOf, isAvailable, setupChoice, setupHint } from "@/lib/training-equipment";
import {
  UNIT_PARTS,
  focusChoices,
  phaseFitsPart,
  unitSplit,
  type FocusEntry,
  type UnitExercise,
  type UnitPartKey,
} from "@/lib/training-focus";
import {
  pickActivity,
  pickCustomFocus,
  pickFocusKey,
  pickMinutes,
  pickNoFocus,
  pickRest,
  pickUnitExercise,
  type DayEdit,
} from "@/lib/training-plan";

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
  /** The focus catalog; empty on an older backend (then the focus is free text only). */
  focuses?: readonly FocusEntry[];
  /** The stable's equipment is known: offers "Aufbau heute" for saved days. */
  setupOffered?: boolean;
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
 * Edits one day of the week: activity or rest day, minutes, focus (a chip from the catalog or the
 * owner's own text), the three parts of the unit with an exercise each, "Aufbau heute" and (for
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
  focuses = [],
  setupOffered = false,
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
  // "Eigener Text" is chosen (also while the text is still empty); the part whose exercises are listed.
  const [custom, setCustom] = useState(isCustom(value));
  const [openPart, setOpenPart] = useState<UnitPartKey | null>(null);
  // The parent builds a new value object on every render; compare by content.
  const valueKey = JSON.stringify(value);
  useEffect(() => {
    setEdit(value);
    setCustom(isCustom(value));
    setOpenPart(null);
  }, [open, date, valueKey]);

  const rest = edit.status === "rest" || edit.activity === null;
  const choices = ACTIVITIES.filter((a) => allowed.includes(a) || a === edit.activity);
  const libraries = edit.activity ? exerciseLibraries(edit.activity, discipline) : [];
  const inLibraries = exercises.filter((e) => libraries.includes(libraryOf(e)));
  const person = people.find((p) => p.id === edit.userId);
  const nothingChosen = edit.status === "planned" && edit.activity === null;

  const split = edit.activity ? unitSplit(edit.activity, edit.minutes) : null;
  // The three parts are offered where the backend plans units (it sent a catalog or this day's unit).
  const unitOffered = split !== null && (focuses.length > 0 || edit.unit !== undefined);
  const hasSetup = edit.activity !== null && LOCATIONS.some((l) => l.key === edit.activity);

  /** The exercises that fit a part (library and phase); the one chosen stays in the list even when it does not fit. */
  const partChoices = (part: UnitPartKey, current: UnitExercise | null): UnitExercise[] => {
    const list: UnitExercise[] = inLibraries.filter((e) => phaseFitsPart(e.phase, part)).map((e) => ({ id: e.id, title: e.title }));
    return current && !list.some((x) => x.id === current.id) ? [current, ...list] : list;
  };
  const availableOf = (id: string) => {
    const found = exercises.find((e) => e.id === id);
    return found ? isAvailable(found) : true;
  };

  // The single exercise of older backends: every exercise of the activity's libraries.
  const singleChoices: UnitExercise[] = inLibraries.map((e) => ({ id: e.id, title: e.title }));
  const singleItems =
    edit.exercise && !singleChoices.some((e) => e.id === edit.exercise!.id) ? [edit.exercise, ...singleChoices] : singleChoices;

  return (
    <Sheet
      open={open}
      onOpenChange={onOpenChange}
      title={formatDayLong(date)}
      description={mode === "draft" ? "Gilt erst beim Übernehmen." : "Gilt sofort nach dem Speichern."}
    >
      <View className="gap-2">
        <Text variant="eyebrow">Was</Text>
        <View className="flex-row flex-wrap gap-2">
          {choices.map((a) => (
            <Pill
              key={a}
              label={activityLabel(a)}
              selected={edit.activity === a && !rest}
              onPress={() => setEdit((e) => pickActivity(e, a, discipline))}
            />
          ))}
          <Pill label="Ruhetag" selected={edit.status === "rest"} onPress={() => setEdit((e) => pickRest(e, value.focusKey))} />
        </View>
      </View>

      {!rest && edit.activity ? (
        <>
          <View className="gap-2">
            <Text variant="eyebrow">Wie lange</Text>
            <View className="flex-row flex-wrap gap-2">
              {minuteChoices(edit.activity, edit.minutes).map((m) => (
                <Pill key={m} label={`${m} Min.`} selected={edit.minutes === m} onPress={() => setEdit((e) => pickMinutes(e, m))} />
              ))}
            </View>
          </View>

          <View className="gap-2">
            <Text variant="eyebrow">Schwerpunkt</Text>
            {focuses.length > 0 ? (
              <View className="flex-row flex-wrap gap-2">
                <Pill
                  label="Kein"
                  selected={!edit.focusKey && !custom}
                  onPress={() => {
                    setCustom(false);
                    setEdit((e) => pickNoFocus(e, value.focusKey));
                  }}
                />
                {focusChoices(focuses, edit.activity, edit.focusKey).map((f) => (
                  <Pill
                    key={f.key}
                    label={f.label}
                    selected={edit.focusKey === f.key}
                    onPress={() => {
                      setCustom(false);
                      setEdit((e) => pickFocusKey(e, f.key));
                    }}
                  />
                ))}
                <Pill
                  label="Eigener Text"
                  selected={custom && !edit.focusKey}
                  onPress={() => {
                    setCustom(true);
                    setEdit((e) => pickCustomFocus(e, value.focusKey));
                  }}
                />
              </View>
            ) : null}
            {focuses.length === 0 || (custom && !edit.focusKey) ? (
              <Input
                value={edit.focus}
                onChangeText={(focus) => setEdit((e) => ({ ...e, focus }))}
                maxLength={80}
                placeholder="z. B. Übergänge Schritt-Trab"
                accessibilityLabel="Schwerpunkt"
              />
            ) : null}
          </View>

          {unitOffered && split ? (
            <View className="gap-1">
              <Text variant="eyebrow">Stundenbild</Text>
              {UNIT_PARTS.map(({ key, label }) => {
                const part = edit.unit?.[key];
                const current = part?.exercise ?? null;
                const minutes = part?.minutes ?? split[key];
                const isOpen = openPart === key;
                return (
                  <View key={key} className="border-b border-divider">
                    <Pressable
                      accessibilityRole="button"
                      accessibilityLabel={`${label}, ${minutes} Minuten: ${current ? current.title : "keine Übung"}`}
                      accessibilityState={{ expanded: isOpen }}
                      accessibilityHint={isOpen ? "Auswahl schließen" : "Übung wählen"}
                      onPress={() => setOpenPart(isOpen ? null : key)}
                      className="min-h-[56px] flex-row items-center gap-3 py-2 active:opacity-70"
                    >
                      <Text variant="heroNumberSm" tone="brass" className="w-10 text-[28px] leading-[32px]" accessibilityElementsHidden>
                        {minutes}
                      </Text>
                      <View className="flex-1 gap-0.5">
                        <Text variant="eyebrow">{label}</Text>
                        <Text variant="bodyStrong">{current ? current.title : "Keine Übung"}</Text>
                      </View>
                      <Icon as={isOpen ? ChevronUp : ChevronDown} size={20} className="text-muted" />
                    </Pressable>
                    {isOpen ? (
                      <View className="pb-2">
                        <ChoiceRow
                          title="Keine Übung"
                          selected={current === null}
                          onPress={() => {
                            setEdit((e) => pickUnitExercise(e, key, null));
                            setOpenPart(null);
                          }}
                        />
                        {partChoices(key, current).map((x) => (
                          <ChoiceRow
                            key={x.id}
                            title={x.title}
                            selected={current?.id === x.id}
                            missing={!availableOf(x.id)}
                            onPress={() => {
                              setEdit((e) => pickUnitExercise(e, key, x));
                              setOpenPart(null);
                            }}
                          />
                        ))}
                        {partChoices(key, current).length === 0 ? (
                          <Text variant="caption" className="px-1 py-2">
                            Keine Übung für {phaseLabel(key)} in dieser Bibliothek.
                          </Text>
                        ) : null}
                      </View>
                    ) : null}
                  </View>
                );
              })}
            </View>
          ) : singleItems.length > 0 ? (
            <View className="gap-2">
              <Text variant="eyebrow">Übung</Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerClassName="gap-2">
                <Pill label="Keine" selected={edit.exercise === null} onPress={() => setEdit((e) => ({ ...e, exercise: null }))} />
                {singleItems.map((x) => (
                  <Pill
                    key={x.id}
                    label={x.title}
                    className={availableOf(x.id) ? undefined : "opacity-50"}
                    selected={edit.exercise?.id === x.id}
                    onPress={() => setEdit((e) => ({ ...e, exercise: { id: x.id, title: x.title } }))}
                  />
                ))}
              </ScrollView>
            </View>
          ) : null}

          {mode === "planned" && setupOffered && hasSetup ? (
            <View className="gap-2">
              <Text variant="eyebrow">Aufbau heute</Text>
              <View className="flex-row flex-wrap gap-2">
                {SETUP_CHOICES.map((c) => (
                  <Pill
                    key={c.value}
                    label={c.label}
                    selected={setupChoice(edit.equipmentOk) === c.value}
                    onPress={() => setEdit((e) => ({ ...e, equipmentOk: equipmentOkOf(c.value) }))}
                  />
                ))}
              </View>
              <Text variant="caption">{setupHint(setupChoice(edit.equipmentOk))}</Text>
            </View>
          ) : null}
        </>
      ) : null}

      {mode === "planned" && !rest ? (
        <View className="gap-2">
          <Text variant="eyebrow">Wer</Text>
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

/** Free text without a catalog focus: "Eigener Text" starts out chosen. */
function isCustom(value: DayEdit): boolean {
  return value.focus !== "" && !value.focusKey;
}

/** One exercise in the list of a part: a tick when chosen, dimmed with "Ausstattung fehlt" when the stable cannot set it up. */
function ChoiceRow({
  title,
  selected,
  missing = false,
  onPress,
}: {
  title: string;
  selected: boolean;
  missing?: boolean;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`${title}${missing ? `, ${MISSING_LABEL}` : ""}`}
      accessibilityState={{ selected }}
      onPress={onPress}
      className="min-h-touch flex-row items-center gap-3 px-1 py-2 active:opacity-70"
    >
      <View className={missing ? "flex-1 gap-0.5 opacity-50" : "flex-1 gap-0.5"}>
        <Text variant="body" className={selected ? "font-sans-semibold" : undefined}>
          {title}
        </Text>
        {missing ? <Text variant="caption">{MISSING_LABEL}</Text> : null}
      </View>
      {selected ? <Icon as={Check} size={18} className="text-brass" /> : null}
    </Pressable>
  );
}
