import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { Button, Card, PageHeader, Pill, RangeStepper, Screen, Switch, Text } from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import {
  MAX_MINUTES_OPTIONS,
  draftFromProfile,
  draftToInput,
  validateSection,
  type ProfileDraft,
} from "@/lib/training-profile";

/** The standard limits plus the stored one when it is none of them (in order, "Unbegrenzt" last). */
function maxMinutesOptions(stored: number): { value: number; label: string }[] {
  if (MAX_MINUTES_OPTIONS.some((o) => o.value === stored)) return [...MAX_MINUTES_OPTIONS];
  const extra = { value: stored, label: `${stored} Min.` };
  const limited = MAX_MINUTES_OPTIONS.filter((o) => o.value > 0);
  const index = limited.findIndex((o) => o.value > stored);
  const before = index < 0 ? limited : limited.slice(0, index);
  const after = index < 0 ? [] : limited.slice(index);
  return [...before, extra, ...after, ...MAX_MINUTES_OPTIONS.filter((o) => o.value === 0)];
}

/** Editor of the weekly rhythm (JAN-99): sessions, rest days, longest session and rest after a show. */
export default function RhythmEditor() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const profile = useProfile(id);
  const save = useSaveProfile(id ?? "");
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [error, setError] = useState<string | null>(null);

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
        <Card className="gap-3">
          <Text variant="secondary">{trainingError(profile.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </Card>
      </Screen>
    );
  }
  if (data && !data.can_edit) {
    return (
      <Screen back>
        <Card>
          <Text variant="secondary">Nur Besitzer und Admins können das Profil ändern.</Text>
        </Card>
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

  const patch = (p: Partial<ProfileDraft>) => {
    setError(null);
    setDraft((d) => (d ? { ...d, ...p } : d));
  };
  const num = (text: string) => Number(text);
  // Sessions and rest days must fit into one week: the other range gives way.
  const setSessions = (min: number, max: number) =>
    patch({
      sessionsMin: String(min),
      sessionsMax: String(max),
      restDaysMin: String(Math.min(num(draft.restDaysMin), 7 - min)),
    });
  const setRestDays = (min: number, max: number) => {
    const sessionsMin = Math.min(num(draft.sessionsMin), 7 - min);
    patch({
      restDaysMin: String(min),
      restDaysMax: String(max),
      sessionsMin: String(sessionsMin),
      sessionsMax: String(Math.max(num(draft.sessionsMax), sessionsMin)),
    });
  };

  const submit = () => {
    const check = validateSection(draft, "rhythm");
    if (!check.ok) {
      setError(check.error);
      return;
    }
    const result = draftToInput(draft);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    save.mutate(result.input as ProfileInput, { onSuccess: () => router.back() });
  };

  const shownError = error ?? (save.isError ? trainingError(save.error) : null);

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader eyebrow={data.horse_name} title="Rhythmus" description="Wie viel pro Woche." />
      <RangeStepper
        label="Einheiten pro Woche"
        min={num(draft.sessionsMin)}
        max={num(draft.sessionsMax)}
        lowerBound={1}
        upperBound={7}
        onChange={setSessions}
      />
      <RangeStepper
        label="Ruhetage pro Woche"
        min={num(draft.restDaysMin)}
        max={num(draft.restDaysMax)}
        lowerBound={0}
        upperBound={6}
        onChange={setRestDays}
      />
      <View className="gap-2">
        <Text variant="label">Höchstdauer pro Einheit</Text>
        <View className="flex-row flex-wrap gap-2">
          {maxMinutesOptions(data.rhythm.max_minutes).map((o) => (
            <Pill
              key={o.value}
              label={o.label}
              selected={draft.maxMinutes === String(o.value)}
              onPress={() => patch({ maxMinutes: String(o.value) })}
            />
          ))}
        </View>
      </View>
      <Switch
        label="Ruhetag nach dem Turnier"
        value={draft.restAfterShow}
        onValueChange={(restAfterShow) => patch({ restAfterShow })}
      />
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
