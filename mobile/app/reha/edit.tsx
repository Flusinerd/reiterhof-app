import { router, useLocalSearchParams } from "expo-router";
import { ChevronDown, ChevronUp, Pencil, Plus, Trash2 } from "lucide-react-native";
import { useEffect, useMemo, useState } from "react";
import { View } from "react-native";

import { HorseError, HorseLoading } from "@/components/horse-query-state";
import { RehaPhaseSheet } from "@/components/reha-phase-sheet";
import { Button, Card, Divider, Hero, Input, Screen, SectionLabel, Text } from "@/components/ui";
import { useObservation } from "@/lib/api/observations";
import { diagnosisFromObservation } from "@/lib/observations";
import { rehaError, useCreatePlan, useReha, useUpdatePlan } from "@/lib/api/reha";
import {
  dateRange,
  daysText,
  draftFromPlan,
  draftToInput,
  emptyPlanDraft,
  movePhase,
  newPhaseDraft,
  phaseSummary,
  placePhases,
  putPhase,
  removePhase,
  totalDays,
  type PhaseDraft,
  type PlanDraft,
} from "@/lib/reha";
import { formatDate, isValidDate } from "@/lib/training";

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <View className="gap-1">
      <Text variant="secondary">{label}</Text>
      {children}
      {hint ? <Text variant="caption">{hint}</Text> : null}
    </View>
  );
}

/** A phase row's numbers for the list: summary and, when the dates are computable, the date range. */
function phaseRow(d: PhaseDraft) {
  const days = Number(d.days);
  return {
    summary: phaseSummary({
      activity: d.activity,
      days: Number.isFinite(days) ? days : 0,
      min_minutes: Number(d.minMinutes) || 0,
      max_minutes: Number(d.maxMinutes) || 0,
    }),
    days: Number.isFinite(days) && days > 0 ? Math.floor(days) : 0,
  };
}

/**
 * Create or edit a reha plan (owner and admins). `?horse=<id>` creates, `?horse=<id>&plan=<id>` edits
 * the active plan; `?observation=<id>` links a new plan to an observation (JAN-68).
 */
export default function RehaEdit() {
  const { horse, plan: planParam, observation } = useLocalSearchParams<{ horse: string; plan?: string; observation?: string }>();
  const reha = useReha(horse);
  const create = useCreatePlan(horse ?? "");
  const update = useUpdatePlan(horse ?? "", planParam ?? "");
  // A new plan from an observation (JAN-68) starts with the diagnosis taken from it.
  const source = useObservation(observation ?? "", !!observation && !planParam);

  const [draft, setDraft] = useState<PlanDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);
  // Index of the phase in the sheet, -1 = a new one.
  const [editing, setEditing] = useState(-1);

  const view = reha.data;
  useEffect(() => {
    if (!view || draft) return;
    if (planParam && view.plan && view.plan.id === planParam) setDraft(draftFromPlan(view.plan));
    else if (!planParam) {
      // Wait for the observation; if it cannot be loaded the form just starts empty.
      if (observation && source.isPending) return;
      const base = emptyPlanDraft(view.date);
      setDraft(source.data && source.data.horse_id === horse ? { ...base, diagnosis: diagnosisFromObservation(source.data) } : base);
    }
  }, [view, draft, planParam, observation, horse, source.isPending, source.data]);

  const dated = useMemo(() => {
    if (!draft || !isValidDate(draft.startDate)) return null;
    return placePhases(draft.startDate, draft.phases.map((p) => phaseRow(p)));
  }, [draft]);

  // The sheet keeps its own copy while open; only a new opening (or another row) resets it.
  const sheetInitial = useMemo(
    () => (!draft ? null : editing >= 0 ? (draft.phases[editing] ?? null) : newPhaseDraft(draft.phases[draft.phases.length - 1])),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [sheetOpen, editing],
  );

  if (reha.isPending) {
    return (
      <Screen back>
        <HorseLoading />
      </Screen>
    );
  }
  if (reha.isError || !view) {
    return (
      <Screen back>
        <HorseError error={reha.error} onRetry={() => void reha.refetch()} />
      </Screen>
    );
  }
  if (!view.can_edit) {
    return (
      <Screen back>
        <Card>
          <Text variant="secondary">Nur der Besitzer und Admins können den Reha-Plan ändern.</Text>
        </Card>
      </Screen>
    );
  }
  if (!draft) {
    return (
      <Screen back>
        <Card>
          <Text variant="secondary">Dieser Reha-Plan ist nicht mehr aktiv und kann nicht mehr geändert werden.</Text>
        </Card>
      </Screen>
    );
  }

  const editingPlan = !!planParam;
  const busy = create.isPending || update.isPending;
  const patch = (p: Partial<PlanDraft>) => setDraft({ ...draft, ...p });
  const total = totalDays(phaseListDays(draft));
  const replacesActive = !editingPlan && view.has_active_plan;

  const submit = () => {
    const result = draftToInput(draft);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    const done = { onSuccess: () => router.back(), onError: (e: unknown) => setError(rehaError(e)) };
    if (editingPlan) update.mutate(result.input, done);
    else create.mutate({ ...result.input, ...(observation ? { observation_id: observation } : {}) }, done);
  };

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        tone="soft"
        eyebrow={view.horse_name}
        title={editingPlan ? "Reha-Plan bearbeiten" : "Reha-Plan anlegen"}
        description="Die Phasen laufen ab dem Startdatum nacheinander. In „Was heute?“ gilt dann nur noch die erlaubte Einheit."
      />

      <SectionLabel>Diagnose</SectionLabel>
      <View className="gap-4">
        <Field label="Diagnose">
          <Input
            accessibilityLabel="Diagnose"
            placeholder="z. B. Sehnenzerrung vorne links"
            value={draft.diagnosis}
            maxLength={200}
            onChangeText={(diagnosis) => patch({ diagnosis })}
          />
        </Field>
        <Field label="Tierarzt (optional)">
          <Input accessibilityLabel="Tierarzt" value={draft.vet} maxLength={120} onChangeText={(vet) => patch({ vet })} />
        </Field>
        <Field label="Start" hint="Format JJJJ-MM-TT">
          <Input
            accessibilityLabel="Startdatum"
            placeholder="JJJJ-MM-TT"
            autoCapitalize="none"
            value={draft.startDate}
            maxLength={10}
            onChangeText={(startDate) => patch({ startDate })}
          />
        </Field>
        <Field label="Kontrolltermin (optional)" hint="Zwei Tage vorher und am Morgen des Termins gibt es eine Erinnerung.">
          <Input
            accessibilityLabel="Kontrolltermin"
            placeholder="JJJJ-MM-TT"
            autoCapitalize="none"
            value={draft.checkupDate}
            maxLength={10}
            onChangeText={(checkupDate) => patch({ checkupDate })}
          />
        </Field>
      </View>

      <SectionLabel>Phasen</SectionLabel>
      {draft.phases.length === 0 ? (
        <Card>
          <Text variant="secondary">Noch keine Phase. Lege zum Beispiel „Boxenruhe“, „Schritt führen“ und „Schritt reiten“ an.</Text>
        </Card>
      ) : (
        <Card padded={false}>
          {draft.phases.map((p, i) => (
            <View key={i}>
              {i > 0 ? <Divider /> : null}
              <View className="gap-2 p-4">
                <View className="gap-1">
                  <Text variant="bodyStrong">{`${i + 1}. ${p.name.trim() || "Ohne Namen"}`}</Text>
                  <Text variant="secondary">{phaseRow(p).summary}</Text>
                  {dated?.[i] && dated[i].days > 0 ? <Text variant="caption">{dateRange(dated[i].start, dated[i].end)}</Text> : null}
                </View>
                <View className="flex-row flex-wrap gap-1">
                  <Button
                    size="icon"
                    variant="ghost"
                    icon={ChevronUp}
                    accessibilityLabel={`Phase ${i + 1} nach oben`}
                    disabled={i === 0}
                    onPress={() => patch({ phases: movePhase(draft.phases, i, -1) })}
                  />
                  <Button
                    size="icon"
                    variant="ghost"
                    icon={ChevronDown}
                    accessibilityLabel={`Phase ${i + 1} nach unten`}
                    disabled={i === draft.phases.length - 1}
                    onPress={() => patch({ phases: movePhase(draft.phases, i, 1) })}
                  />
                  <Button
                    size="icon"
                    variant="ghost"
                    icon={Pencil}
                    accessibilityLabel={`Phase ${i + 1} bearbeiten`}
                    onPress={() => {
                      setEditing(i);
                      setSheetOpen(true);
                    }}
                  />
                  <Button
                    size="icon"
                    variant="ghost"
                    icon={Trash2}
                    accessibilityLabel={`Phase ${i + 1} entfernen`}
                    onPress={() => patch({ phases: removePhase(draft.phases, i) })}
                  />
                </View>
              </View>
            </View>
          ))}
        </Card>
      )}
      <Button
        label="Phase hinzufügen"
        variant="outline"
        icon={Plus}
        fullWidth
        onPress={() => {
          setEditing(-1);
          setSheetOpen(true);
        }}
      />
      {dated && total > 0 ? (
        <Text variant="secondary">
          {daysText(total)} insgesamt, bis {formatDate(dated[dated.length - 1]?.end ?? draft.startDate)}.
        </Text>
      ) : null}

      <SectionLabel>Abbruchkriterien</SectionLabel>
      <Field label="Wann muss sofort abgebrochen werden? (optional)">
        <Input
          accessibilityLabel="Abbruchkriterien"
          placeholder="z. B. Lahmheit, Wärme oder Schwellung im Bein"
          multiline
          textAlignVertical="top"
          className="h-auto min-h-24 py-3"
          value={draft.abortCriteria}
          maxLength={1000}
          onChangeText={(abortCriteria) => patch({ abortCriteria })}
        />
      </Field>

      {replacesActive ? (
        <Text variant="secondary">Der bisherige aktive Plan wird beendet, sobald du diesen speicherst.</Text>
      ) : null}
      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button label={editingPlan ? "Änderungen speichern" : "Plan speichern"} size="lg" fullWidth loading={busy} onPress={submit} />

      <RehaPhaseSheet
        open={sheetOpen}
        onOpenChange={setSheetOpen}
        initial={sheetInitial}
        position={editing >= 0 ? editing + 1 : draft.phases.length + 1}
        onSave={(p) => patch({ phases: putPhase(draft.phases, editing, p) })}
      />
    </Screen>
  );
}

function phaseListDays(d: PlanDraft): { days: number }[] {
  return d.phases.map((p) => ({ days: phaseRow(p).days }));
}
