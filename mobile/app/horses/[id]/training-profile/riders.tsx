import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { RiderRuleCard } from "@/components/training-rider-rules";
import { Button, Card, PageHeader, Screen, Text } from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import {
  draftFromProfile,
  draftToInput,
  validateSection,
  type ProfileDraft,
  type RiderDraft,
} from "@/lib/training-profile";

/** Editor of what each rider may do on the horse (JAN-99). */
export default function RidersEditor() {
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

  const patchRider = (userId: string, p: Partial<RiderDraft>) => {
    setError(null);
    setDraft((d) => (d ? { ...d, riders: d.riders.map((r) => (r.userId === userId ? { ...r, ...p } : r)) } : d));
  };

  const submit = () => {
    const check = validateSection(draft, "riders");
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
      <PageHeader eyebrow={data.horse_name} title="Reitbeteiligungen" description="Was jede Reitbeteiligung darf." />

      {draft.riders.length === 0 ? (
        <Text variant="body" tone="muted">
          Keine Reitbeteiligung eingetragen.
        </Text>
      ) : (
        <>
          {draft.riders.map((r) => (
            <RiderRuleCard key={r.userId} rider={r} editable onChange={(p) => patchRider(r.userId, p)} />
          ))}
          <View className="gap-3">
            {shownError ? (
              <Text variant="secondary" tone="danger" accessibilityRole="alert">
                {shownError}
              </Text>
            ) : null}
            <Button label="Speichern" size="lg" fullWidth loading={save.isPending} onPress={submit} />
          </View>
        </>
      )}
    </Screen>
  );
}
