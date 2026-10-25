import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { ChevronDown, ChevronUp } from "lucide-react-native";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { FocusRows } from "@/components/training-focus-rows";
import { Button, DateField, PageHeader, Screen, Section, Text } from "@/components/ui";
import { RowList } from "@/components/ui/list-row";
import { trainingError, useFocuses, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { useAuth } from "@/lib/auth";
import { horseRoutes } from "@/lib/horse-format";
import { localDate } from "@/lib/presence-format";
import {
  FOCUS_LIMIT_OWNER,
  catalogPhaseOf,
  defaultUntil,
  focusLabel,
  focusNote,
  focusWarnings,
  limitNotice,
  moveFocus,
  phaseHint,
  selectionCaption,
  setFocusUntil,
  toggleFocus,
  validateFocuses,
} from "@/lib/training-focus";
import { draftFromProfile, draftToInput, type ProfileDraft } from "@/lib/training-profile";
import { weekStart } from "@/lib/training";

/**
 * Editor of the horse's focuses ("Schwerpunkte"): up to three from the catalog, grouped "Skala der
 * Ausbildung" and "Praxis", in the order of their priority, each with an optional end date. A hint
 * names the horse's phase, a note under a focus warns when it is above that phase. Choosing
 * Turniervorbereitung starts with the next show as its end.
 */
export default function FocusesEditor() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { me } = useAuth();
  const profile = useProfile(id);
  const catalog = useFocuses();
  const save = useSaveProfile(id ?? "");
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const data = profile.data;
  useEffect(() => {
    if (data?.exists) setDraft((d) => d ?? draftFromProfile(data));
  }, [data]);
  const toSetup = !!data && !data.exists && data.can_edit;
  useEffect(() => {
    if (toSetup && id) router.replace(horseRoutes.trainingSetup(id) as Href);
  }, [toSetup, id, router]);

  if (profile.isError) {
    return (
      <Screen back>
        <View className="gap-3">
          <Text variant="secondary">{trainingError(profile.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </View>
      </Screen>
    );
  }
  if (data && !data.can_edit) {
    return (
      <Screen back>
        <View>
          <Text variant="secondary">Nur Besitzer und Admins können die Schwerpunkte ändern.</Text>
        </View>
      </Screen>
    );
  }
  if (!data || !draft) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }
  if (draft.focuses === undefined) {
    return (
      <Screen back>
        <PageHeader eyebrow={data.horse_name} title="Schwerpunkte" />
        <Text variant="body" tone="muted">
          Schwerpunkte gibt es erst mit einer neueren Version des Servers.
        </Text>
      </Screen>
    );
  }

  const focuses = draft.focuses;
  const entries = catalog.data ?? [];
  const today = localDate(new Date(), me?.stable?.timezone ?? "Europe/Berlin");
  const showDates = data.shows.map((s) => s.date);
  const catalogPhase = catalogPhaseOf(data.phase);
  const hint = phaseHint(data.horse_name, data.phase, data.focus_hints, entries);

  const patch = (next: typeof focuses) => {
    setError(null);
    setDraft((d) => (d ? { ...d, focuses: next } : d));
  };
  const toggle = (key: string) => {
    const result = toggleFocus(focuses, key, FOCUS_LIMIT_OWNER, defaultUntil(key, showDates, today));
    setNotice(result.full ? limitNotice(FOCUS_LIMIT_OWNER) : null);
    if (!result.full) patch(result.drafts);
  };

  // A note under a chosen focus: the server's warning, else the hint that it is above the phase.
  const noteOf = (key: string) =>
    focusNote(
      entries.find((e) => e.key === key),
      data.warnings,
      catalogPhase,
      data.horse_name,
    );
  const notes: Record<string, string> = {};
  for (const f of focuses) {
    const note = noteOf(f.key);
    if (note) notes[f.key] = note;
  }
  // Warnings of the server for focuses that are not (or no longer) chosen here stay out of the way.
  const unchosen = focusWarnings(data.warnings).filter((w) => !focuses.some((f) => f.key === w.key));

  const submit = () => {
    const invalid = validateFocuses(focuses, FOCUS_LIMIT_OWNER);
    if (invalid) {
      setError(invalid);
      return;
    }
    // The block of four weeks starts with the first focuses (the server fills it in too when it is empty).
    const cycleStart = focuses.length > 0 && !draft.cycleStart ? weekStart(today) : draft.cycleStart;
    const result = draftToInput({ ...draft, cycleStart });
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    return save
      .mutateAsync(result.input as ProfileInput)
      .then(() => router.back())
      .catch(() => undefined); // the error shows from the mutation state
  };

  const shownError = error ?? (save.isError ? trainingError(save.error) : null);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader
        eyebrow={data.horse_name}
        title="Schwerpunkte"
        description={`Woran soll ${data.horse_name} gerade arbeiten? Bis zu ${FOCUS_LIMIT_OWNER}, der erste hat Vorrang.`}
      >
        {hint ? <Text variant="secondary">{hint}</Text> : null}
      </PageHeader>

      {focuses.length > 0 ? (
        <Section title="Deine Auswahl" description={selectionCaption(focuses.length, FOCUS_LIMIT_OWNER)}>
          <RowList>
            {focuses.map((f, i) => {
              const label = focusLabel(f.key, entries);
              return (
                <View key={f.key} className="gap-2 py-3">
                  <View className="flex-row items-center gap-3">
                    <Text variant="heroNumberSm" tone="brass" className="w-8 text-[28px] leading-[32px]" accessibilityElementsHidden>
                      {i + 1}
                    </Text>
                    <View className="flex-1 gap-0.5">
                      <Text variant="bodyStrong">{label}</Text>
                      {notes[f.key] ? (
                        <Text variant="secondary" tone="accent">
                          {notes[f.key]}
                        </Text>
                      ) : null}
                    </View>
                    <Button
                      variant="ghost"
                      size="icon"
                      icon={ChevronUp}
                      accessibilityLabel={`${label} nach oben`}
                      disabled={i === 0}
                      onPress={() => patch(moveFocus(focuses, i, -1))}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      icon={ChevronDown}
                      accessibilityLabel={`${label} nach unten`}
                      disabled={i === focuses.length - 1}
                      onPress={() => patch(moveFocus(focuses, i, 1))}
                    />
                  </View>
                  <View className="gap-1 pl-11">
                    <Text variant="caption">Bis (optional)</Text>
                    <DateField
                      accessibilityLabel={`Bis, ${label}`}
                      placeholder="Kein Datum"
                      clearable
                      minimumDate={new Date()}
                      value={f.until}
                      onChange={(until) => patch(setFocusUntil(focuses, f.key, until))}
                    />
                    {f.key === "turnier" && f.until === "" ? (
                      <Text variant="caption">Am besten das Datum des Turniers. Ohne Datum bleibt der Schwerpunkt, bis du ihn herausnimmst.</Text>
                    ) : null}
                  </View>
                </View>
              );
            })}
          </RowList>
        </Section>
      ) : (
        <Text variant="secondary">{selectionCaption(0, FOCUS_LIMIT_OWNER)}. Tippe auf einen Schwerpunkt, um ihn zu wählen.</Text>
      )}

      {notice ? (
        <Text variant="secondary" tone="accent" accessibilityRole="alert">
          {notice}
        </Text>
      ) : null}
      {unchosen.map((w) => (
        <Text key={w.key} variant="secondary" tone="accent">
          {w.text}
        </Text>
      ))}

      {catalog.isPending ? (
        <ActivityIndicator accessibilityLabel="Lädt" className="self-start" />
      ) : catalog.isError ? (
        <View className="gap-3">
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {trainingError(catalog.error)}
          </Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void catalog.refetch()} />
        </View>
      ) : (
        <FocusRows catalog={entries} selected={focuses.map((f) => f.key)} onToggle={toggle} />
      )}

      <View className="gap-3">
        {shownError ? (
          <Text variant="secondary" tone="danger" accessibilityRole="alert">
            {shownError}
          </Text>
        ) : null}
        <Button label="Speichern" size="lg" fullWidth loading={save.isPending} onPress={submit} />
      </View>
    </Screen>
  );
}
