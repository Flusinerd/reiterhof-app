import { useLocalSearchParams } from "expo-router";
import { Check, Trash2 } from "lucide-react-native";
import { useMemo, useState } from "react";
import { ActivityIndicator, Alert, View } from "react-native";

import { Button, Icon, Input, NumberStepper, PageHeader, Pill, Screen, Section, Segmented, Text } from "@/components/ui";
import {
  dressageError,
  useDressageCatalog,
  useGenerateTest,
  useRemoveRequirementSet,
  useSaveRequirementSet,
  type GenerateBody,
} from "@/lib/api/dressage";
import { useFocuses, useProfile } from "@/lib/api/training";
import {
  ARENA_SIZES,
  FALLBACK_FREESTYLE_CLASSES,
  PRACTICE_CLASSES,
  REIN_OPTIONS,
  allowedClasses,
  classCeiling,
  defaultClass,
  formatSeconds,
  gaitLabel,
  movementHint,
  requirementsReady,
  setSummary,
  setsOfClass,
  toggleMovement,
  updateMovement,
  type ArenaSize,
  type RequirementMovement,
  type TestKind,
} from "@/lib/dressage";
import { newSeed, setPreview } from "@/lib/dressage-preview";
import { pushOnce } from "@/lib/navigation";
import { classesAbove } from "@/lib/training-goal";

const KINDS = [
  { value: "practice", label: "Übungsaufgabe" },
  { value: "freestyle", label: "Kür" },
] as const;

const TEN = 10;

/**
 * "Neue Aufgabe": Übungsaufgabe or Kür, class, arena; for a practice test the length and a focus, for a
 * freestyle the editor of the requirements (nothing is prefilled: the movements and the time window come
 * from the rider's own Aufgabenheft). "Erstellen" generates a preview and opens the plan view.
 */
export default function NewDressageTest() {
  const { horse } = useLocalSearchParams<{ horse?: string }>();
  const horseId = horse ?? "";
  const profile = useProfile(horseId);
  const catalog = useDressageCatalog();
  const focuses = useFocuses();
  const generate = useGenerateTest(horseId);
  const saveSet = useSaveRequirementSet(horseId);
  const removeSet = useRemoveRequirementSet(horseId);

  const [kind, setKind] = useState<TestKind>("practice");
  const [pickedClass, setPickedClass] = useState<string | null>(null);
  const [arena, setArena] = useState<ArenaSize>("20x40");
  const [length, setLength] = useState<"short" | "normal">("normal");
  const [focus, setFocus] = useState<string | null>(null);
  const [movements, setMovements] = useState<RequirementMovement[]>([]);
  const [minTen, setMinTen] = useState(30);
  const [maxTen, setMaxTen] = useState(36);
  const [setId, setSetId] = useState<string | null>(null);
  const [label, setLabel] = useState("");
  const [savedHint, setSavedHint] = useState(false);

  const level = profile.data?.level;
  const goal = profile.data?.goal_level ?? classesAbove(level)[0] ?? null;
  const ceiling = classCeiling(level, goal);
  const classes = useMemo(() => {
    const fromCatalog = catalog.data?.classes.map((c) => c.key) ?? [];
    const freestyle = fromCatalog.length > 0 ? fromCatalog : [...FALLBACK_FREESTYLE_CLASSES];
    return allowedClasses(kind === "practice" ? PRACTICE_CLASSES : freestyle, ceiling);
  }, [kind, catalog.data, ceiling]);
  const cls = pickedClass && classes.includes(pickedClass) ? pickedClass : defaultClass(classes, level);

  const sets = setsOfClass(catalog.data?.requirement_sets ?? [], cls);
  const allMovements = catalog.data?.movements ?? [];
  const groups = (["walk", "trot", "canter", "halt"] as const)
    .map((g) => ({ gait: g, items: allMovements.filter((m) => m.gait === g) }))
    .filter((g) => g.items.length > 0);
  const labelOf = (key: string) => allMovements.find((m) => m.key === key)?.label ?? key;
  const ready = kind === "practice" || requirementsReady(movements, minTen * TEN, maxTen * TEN);
  const noCatalog = kind === "freestyle" && catalog.isError;

  const loadSet = (id: string) => {
    const s = sets.find((x) => x.id === id);
    if (!s) return;
    setSetId(s.id);
    setLabel(s.label);
    setMovements(s.movements);
    setMinTen(Math.max(6, Math.round(s.min_seconds / TEN)));
    setMaxTen(Math.max(6, Math.round(s.max_seconds / TEN)));
    setSavedHint(false);
  };

  const setBody = () => ({ class: cls, label: label.trim() || `Kür ${cls}`, movements, min_seconds: minTen * TEN, max_seconds: maxTen * TEN });

  const submit = async () => {
    const request: Omit<GenerateBody, "save"> =
      kind === "practice"
        ? { kind, class: cls, arena, length, seed: newSeed(), ...(focus ? { focus_key: focus } : {}) }
        : { kind, class: cls, arena, seed: newSeed(), requirements: { movements, min_seconds: minTen * TEN, max_seconds: maxTen * TEN } };
    const { plan } = await generate.mutateAsync({ ...request, save: false });
    setPreview({ horseId, request, plan });
    pushOnce({ pathname: "/training/tests/[id]", params: { id: "preview", horse: horseId } });
  };

  const confirmRemoveSet = (id: string, name: string) =>
    Alert.alert("Set löschen?", `„${name}“ wird für den ganzen Stall gelöscht.`, [
      { text: "Abbrechen", style: "cancel" },
      {
        text: "Löschen",
        style: "destructive",
        onPress: () =>
          void removeSet.mutateAsync(id).then(
            () => {
              if (setId === id) setSetId(null);
            },
            (e) => Alert.alert("Das hat nicht geklappt", dressageError(e)),
          ),
      },
    ]);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader eyebrow="Aufgaben und Küren" title="Neue Aufgabe" />

      <Segmented
        items={KINDS}
        value={kind}
        onValueChange={(v) => {
          setKind(v as TestKind);
          generate.reset();
        }}
      />

      <Section title="Klasse">
        <View className="flex-row flex-wrap gap-2">
          {classes.map((c) => (
            <Pill key={c} label={c} selected={cls === c} onPress={() => setPickedClass(c)} />
          ))}
        </View>
      </Section>

      <Section title="Viereck">
        <View className="flex-row flex-wrap gap-2">
          {ARENA_SIZES.map((a) => (
            <Pill key={a.value} label={a.label} selected={arena === a.value} onPress={() => setArena(a.value)} />
          ))}
        </View>
      </Section>

      {kind === "practice" ? (
        <>
          <Section title="Länge">
            <View className="flex-row flex-wrap gap-2">
              <Pill label="Kurz · 3 bis 4 Min." selected={length === "short"} onPress={() => setLength("short")} />
              <Pill label="Normal · 5 bis 7 Min." selected={length === "normal"} onPress={() => setLength("normal")} />
            </View>
          </Section>
          {(focuses.data ?? []).length > 0 ? (
            <Section title="Schwerpunkt" description="Optional: bevorzugt Lektionen, die dazu passen.">
              <View className="flex-row flex-wrap gap-2">
                {(focuses.data ?? []).map((f) => (
                  <Pill key={f.key} label={f.label} selected={focus === f.key} onPress={() => setFocus(focus === f.key ? null : f.key)} />
                ))}
              </View>
            </Section>
          ) : null}
        </>
      ) : (
        <Section title="Anforderungen" description="Aus deinem Aufgabenheft übernehmen: wähle die geforderten Lektionen und trage die Zeit ein.">
          {catalog.isPending ? <ActivityIndicator accessibilityLabel="Lädt" className="self-start" /> : null}
          {noCatalog ? (
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {dressageError(catalog.error)}
            </Text>
          ) : null}

          {sets.length > 0 ? (
            <View className="gap-2">
              <Text variant="eyebrow">Gespeicherte Sets</Text>
              <View className="flex-row flex-wrap gap-2">
                {sets.map((s) => (
                  <Pill key={s.id} label={s.label} selected={setId === s.id} onPress={() => loadSet(s.id)} />
                ))}
              </View>
              {setId && sets.some((s) => s.id === setId) ? (
                <View className="flex-row items-center justify-between gap-2">
                  <Text variant="bodySm" tone="muted" className="flex-1">
                    {setSummary(sets.find((s) => s.id === setId)!)}
                  </Text>
                  <Button
                    label="Set löschen"
                    icon={Trash2}
                    variant="ghost"
                    size="sm"
                    onPress={() => confirmRemoveSet(setId, sets.find((s) => s.id === setId)!.label)}
                  />
                </View>
              ) : null}
            </View>
          ) : null}

          {groups.map((g) => (
            <View key={g.gait} className="gap-2">
              <Text variant="eyebrow">{gaitLabel(g.gait)}</Text>
              <View className="flex-row flex-wrap gap-2">
                {g.items.map((m) => (
                  <Pill
                    key={m.key}
                    label={m.label}
                    selected={movements.some((x) => x.key === m.key)}
                    onPress={() => setMovements(toggleMovement(movements, m.key))}
                    accessibilityHint={movementHint(m)}
                  />
                ))}
              </View>
            </View>
          ))}

          {movements.length > 0 ? (
            <View className="gap-3">
              <Text variant="eyebrow">Deine Auswahl</Text>
              {movements.map((m) => {
                const info = allMovements.find((x) => x.key === m.key);
                return (
                  <View key={m.key} className="gap-2 border-t border-divider pt-3">
                    <Text variant="bodyStrong">{labelOf(m.key)}</Text>
                    {info && !info.class ? (
                      <Text variant="caption">Klasse nicht belegt</Text>
                    ) : null}
                    <NumberStepper
                      value={m.count}
                      min={1}
                      max={9}
                      onChange={(count) => setMovements(updateMovement(movements, m.key, { count }))}
                      accessibilityLabel={`Anzahl ${labelOf(m.key)}`}
                      format={(v) => `${v}×`}
                    />
                    <View className="flex-row flex-wrap gap-2">
                      {REIN_OPTIONS.map((r) => (
                        <Pill key={r.value} label={r.label} selected={m.rein === r.value} onPress={() => setMovements(updateMovement(movements, m.key, { rein: r.value }))} />
                      ))}
                    </View>
                  </View>
                );
              })}
            </View>
          ) : null}

          <View className="gap-3">
            <Text variant="eyebrow">Zeitfenster</Text>
            <View className="flex-row items-center justify-between gap-3">
              <Text variant="body">Mindestens</Text>
              <NumberStepper
                value={minTen}
                min={6}
                max={36}
                onChange={(v) => {
                  setMinTen(v);
                  if (maxTen < v) setMaxTen(v);
                }}
                accessibilityLabel="Mindestdauer"
                format={(v) => `${formatSeconds(v * TEN)} Min.`}
              />
            </View>
            <View className="flex-row items-center justify-between gap-3">
              <Text variant="body">Höchstens</Text>
              <NumberStepper
                value={maxTen}
                min={6}
                max={36}
                onChange={(v) => {
                  setMaxTen(v);
                  if (minTen > v) setMinTen(v);
                }}
                accessibilityLabel="Höchstdauer"
                format={(v) => `${formatSeconds(v * TEN)} Min.`}
              />
            </View>
          </View>

          {movements.length > 0 ? (
            <View className="gap-2">
              <Input value={label} onChangeText={setLabel} placeholder="Name des Sets, z. B. Kür Herbst" accessibilityLabel="Name des Sets" maxLength={60} />
              <View className="flex-row items-center gap-3">
                <Button
                  label={setId ? "Set aktualisieren" : "Set für den Stall speichern"}
                  variant="outline"
                  size="sm"
                  disabled={!ready}
                  onPress={() =>
                    saveSet.mutateAsync({ ...setBody(), ...(setId ? { id: setId } : {}) }).then(
                      (s) => {
                        setSetId(s.id || setId);
                        setSavedHint(true);
                      },
                      () => undefined,
                    )
                  }
                />
                {savedHint && !saveSet.isError ? (
                  <View className="flex-row items-center gap-1.5">
                    <Icon as={Check} size={16} className="text-success" />
                    <Text variant="bodySm" tone="success">
                      Gespeichert
                    </Text>
                  </View>
                ) : null}
              </View>
              {saveSet.isError ? (
                <Text variant="secondary" tone="danger" accessibilityRole="alert">
                  {dressageError(saveSet.error)}
                </Text>
              ) : null}
            </View>
          ) : null}
        </Section>
      )}

      <View className="gap-3">
        <Button label="Erstellen" fullWidth disabled={!ready || !horseId || noCatalog} onPress={() => submit().catch(() => undefined)} />
        {generate.isError ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {dressageError(generate.error)}
          </Text>
        ) : null}
        {kind === "freestyle" && !ready ? <Text variant="caption">Wähle mindestens eine Lektion und ein Zeitfenster.</Text> : null}
      </View>
    </Screen>
  );
}
