import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { WeekStructureRows } from "@/components/training-week-structure";
import { Button, Card, Divider, Input, PageHeader, Screen, Section, Text } from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import { ACTIVITIES, activityLabel } from "@/lib/training";
import { draftFromProfile, draftToInput, validateSection, type ProfileDraft } from "@/lib/training-profile";

/** Editor of the fixed weekdays and the weekly goals (JAN-93, JAN-99). */
export default function StructureEditor() {
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

  const submit = () => {
    const check = validateSection(draft, "structure");
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
      <PageHeader
        eyebrow={data.horse_name}
        title="Feste Tage und Ziele"
        description="Die Planung hält sich daran; nur Sicherheitsregeln gehen vor."
      />

      <WeekStructureRows days={draft.days} modes={draft.modes} editable onChange={(days) => patch({ days })} />

      <Section
        title="Wochenziele"
        description="Einheiten pro Woche (Mo–So). Die Planung füllt sie auf und plant nicht mehr fordernde Einheiten oder mehr einer Aktivität als angegeben. Leer = kein Ziel."
      >
        <Card padded={false}>
          <QuotaRow
            label="Fordernd"
            value={draft.quotaDemanding}
            first
            onChange={(quotaDemanding) => patch({ quotaDemanding })}
          />
          <QuotaRow
            label="Aktive Erholung (mindestens)"
            value={draft.quotaRecovery}
            onChange={(quotaRecovery) => patch({ quotaRecovery })}
          />
          {ACTIVITIES.filter((a) => draft.modes[a].mode !== "off").map((a) => (
            <QuotaRow
              key={a}
              label={activityLabel(a)}
              value={draft.quotaActivities[a]}
              onChange={(v) => patch({ quotaActivities: { ...draft.quotaActivities, [a]: v } })}
            />
          ))}
        </Card>
        <Text variant="caption">
          Die Belastung steigt höchstens um etwa 20 % gegenüber dem Schnitt der letzten zwei Wochen.
        </Text>
      </Section>

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

function QuotaRow({
  label,
  value,
  first,
  onChange,
}: {
  label: string;
  value: string;
  first?: boolean;
  onChange: (v: string) => void;
}) {
  return (
    <View>
      {first ? null : <Divider />}
      <View className="min-h-12 flex-row items-center gap-3 px-4 py-2">
        <Text variant="body" className="flex-1">
          {label}
        </Text>
        <Input
          accessibilityLabel={`${label}, pro Woche`}
          keyboardType="number-pad"
          placeholder="–"
          value={value}
          maxLength={1}
          onChangeText={onChange}
          className="w-16 text-center"
        />
      </View>
    </View>
  );
}
