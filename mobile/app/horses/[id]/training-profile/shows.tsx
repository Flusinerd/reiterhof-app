import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { Plus, Trash2 } from "lucide-react-native";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { Button, Card, DateField, Input, PageHeader, Screen, Section, Text } from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import { horseRoutes } from "@/lib/horse-format";
import {
  draftFromProfile,
  draftToInput,
  validateSection,
  type ProfileDraft,
  type ShowDraft,
} from "@/lib/training-profile";

/** Editor of the shows and the end of the season (JAN-99). */
export default function ShowsEditor() {
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
  const patchShow = (i: number, p: Partial<ShowDraft>) => patch({ shows: draft.shows.map((s, j) => (j === i ? { ...s, ...p } : s)) });

  const submit = () => {
    // The picker leaves a date empty until one is chosen; the section validator words that as a format error.
    if (draft.shows.some((s) => s.date === "")) {
      setError("Jedes Turnier braucht ein Datum.");
      return;
    }
    const check = validateSection(draft, "shows");
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
      <PageHeader eyebrow={data.horse_name} title="Turniere" description="Termine und das Saisonende." />

      {draft.shows.length === 0 ? <Text variant="body" tone="muted">Keine Turniere eingetragen.</Text> : null}
      {draft.shows.map((s, i) => (
        <Card key={i} className="gap-3">
          <DateField
            accessibilityLabel={`Datum, Turnier ${i + 1}`}
            value={s.date}
            onChange={(date) => patchShow(i, { date })}
          />
          <Input
            accessibilityLabel={`Name, Turnier ${i + 1}`}
            placeholder="Name"
            value={s.name}
            maxLength={80}
            onChangeText={(name) => patchShow(i, { name })}
          />
          <Input
            accessibilityLabel={`Klassen, Turnier ${i + 1}`}
            placeholder="Klassen (optional)"
            value={s.classes}
            maxLength={200}
            onChangeText={(classes) => patchShow(i, { classes })}
          />
          <Input
            accessibilityLabel={`Helfer, Turnier ${i + 1}`}
            placeholder="Helfer (optional)"
            value={s.helper}
            maxLength={200}
            onChangeText={(helper) => patchShow(i, { helper })}
          />
          <Button
            label="Entfernen"
            variant="ghost"
            size="sm"
            icon={Trash2}
            accessibilityLabel={`Turnier ${i + 1} entfernen`}
            onPress={() => patch({ shows: draft.shows.filter((_, j) => j !== i) })}
          />
        </Card>
      ))}
      <Button
        label="Turnier hinzufügen"
        variant="outline"
        icon={Plus}
        fullWidth
        onPress={() => patch({ shows: [...draft.shows, { date: "", name: "", classes: "", helper: "" }] })}
      />

      <Section title="Saisonende">
        <DateField
          accessibilityLabel="Saisonende"
          placeholder="Nicht festgelegt"
          clearable
          value={draft.seasonEnd}
          onChange={(seasonEnd) => patch({ seasonEnd })}
        />
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
