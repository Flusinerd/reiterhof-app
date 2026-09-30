import { useLocalSearchParams } from "expo-router";
import { Plus, Trash2 } from "lucide-react-native";
import { useEffect, useState } from "react";
import { ActivityIndicator, View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import {
  Badge,
  Button,
  Card,
  Divider,
  Hero,
  Input,
  Pill,
  Screen,
  SectionLabel,
  Switch,
  Text,
} from "@/components/ui";
import { trainingError, useProfile, useSaveProfile, type ProfileInput } from "@/lib/api/training";
import {
  ACTIVITIES,
  ACTIVITY_MODES,
  DISCIPLINES,
  MAX_INTENSITY_OPTIONS,
  PROFILE_STATUSES,
  activityLabel,
  disciplineLabel,
  formatDate,
  statusLabel,
  type Activity,
} from "@/lib/training";
import {
  draftFromProfile,
  draftToInput,
  toggleActivity,
  type ProfileDraft,
  type RiderDraft,
  type ShowDraft,
} from "@/lib/training-profile";

/**
 * Training profile (JAN-55): editable for the owner (and admins), read-only for riders. Riders
 * only see their own rules.
 */
export default function TrainingProfile() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const profile = useProfile(id);
  const save = useSaveProfile(id ?? "");
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState(false);

  useEffect(() => {
    if (profile.data) setDraft(draftFromProfile(profile.data));
  }, [profile.data]);

  if (profile.isPending || (profile.data && !draft)) {
    return (
      <Screen back>
        <View className="items-center py-16">
          <ActivityIndicator />
        </View>
      </Screen>
    );
  }
  if (profile.isError || !profile.data || !draft) {
    return (
      <Screen back>
        <Card className="gap-3">
          <Text variant="secondary">{profile.isError ? trainingError(profile.error) : "Kein Profil gefunden."}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void profile.refetch()} />
        </Card>
      </Screen>
    );
  }

  const editable = profile.data.can_edit;
  const patch = (p: Partial<ProfileDraft>) => {
    setSavedAt(false);
    setDraft({ ...draft, ...p });
  };
  const patchMode = (a: Activity, p: Partial<ProfileDraft["modes"][Activity]>) =>
    patch({ modes: { ...draft.modes, [a]: { ...draft.modes[a], ...p } } });
  const patchShow = (i: number, p: Partial<ShowDraft>) =>
    patch({ shows: draft.shows.map((s, j) => (j === i ? { ...s, ...p } : s)) });
  const patchRider = (userId: string, p: Partial<RiderDraft>) =>
    patch({ riders: draft.riders.map((r) => (r.userId === userId ? { ...r, ...p } : r)) });

  const submit = () => {
    const result = draftToInput(draft);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    save.mutate(result.input as ProfileInput, { onSuccess: () => setSavedAt(true) });
  };

  const name = profile.data.horse_name;

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <Hero
        tone="soft"
        eyebrow={name}
        title="Trainingsprofil"
        description={
          editable
            ? "Bestimmt, was in „Was heute?“ vorgeschlagen wird und wie oft trainiert werden soll."
            : "Nur der Besitzer kann das Profil ändern. Du siehst hier, was für dich gilt."
        }
      >
        {!profile.data.exists ? <Badge variant="accent" label="Noch nicht angelegt" /> : null}
      </Hero>

      <SectionLabel>Status</SectionLabel>
      <View className="flex-row flex-wrap gap-2">
        {PROFILE_STATUSES.map((s) => (
          <Pill
            key={s.value}
            label={s.label}
            selected={draft.status === s.value}
            disabled={!editable}
            onPress={() => patch({ status: s.value })}
          />
        ))}
      </View>

      <SectionLabel>Disziplin und Niveau</SectionLabel>
      <View className="flex-row flex-wrap gap-2">
        {DISCIPLINES.map((d) => (
          <Pill
            key={d.value}
            label={d.label}
            selected={draft.discipline === d.value}
            disabled={!editable}
            onPress={() => patch({ discipline: d.value })}
          />
        ))}
      </View>
      {editable ? (
        <Input
          accessibilityLabel="Niveau"
          placeholder="Niveau, z. B. L"
          value={draft.level}
          maxLength={40}
          onChangeText={(level) => patch({ level })}
        />
      ) : (
        <Text variant="secondary">
          {disciplineLabel(draft.discipline)}
          {draft.level ? `, Niveau ${draft.level}` : ""}
        </Text>
      )}

      <SectionLabel>Aktivitäten</SectionLabel>
      <Card padded={false}>
        {ACTIVITIES.map((a, i) => {
          const m = draft.modes[a];
          return (
            <View key={a}>
              {i > 0 ? <Divider /> : null}
              <View className="gap-3 p-4">
                <View className="flex-row items-center gap-3">
                  <ActivityIcon activity={a} size={20} />
                  <Text variant="bodyStrong" className="flex-1">
                    {activityLabel(a)}
                  </Text>
                </View>
                <View className="flex-row flex-wrap gap-2">
                  {ACTIVITY_MODES.map((mode) => (
                    <Pill
                      key={mode.value}
                      label={mode.label}
                      selected={m.mode === mode.value}
                      disabled={!editable}
                      onPress={() => patchMode(a, { mode: mode.value })}
                    />
                  ))}
                </View>
                {m.mode === "conditional" ? (
                  editable ? (
                    <Input
                      accessibilityLabel={`Bedingung für ${activityLabel(a)}`}
                      placeholder="Bedingung, z. B. nur mit Begleitung"
                      value={m.note}
                      maxLength={200}
                      onChangeText={(note) => patchMode(a, { note })}
                    />
                  ) : (
                    <Text variant="secondary">Bedingung: {m.note}</Text>
                  )
                ) : null}
              </View>
            </View>
          );
        })}
      </Card>

      <SectionLabel>Turniere</SectionLabel>
      <View className="gap-3">
        {draft.shows.length === 0 ? <Text variant="secondary">Keine Turniere eingetragen.</Text> : null}
        {draft.shows.map((s, i) =>
          editable ? (
            <Card key={i} className="gap-3">
              <Input
                accessibilityLabel="Datum des Turniers"
                placeholder="Datum, z. B. 2026-05-17"
                value={s.date}
                autoCapitalize="none"
                onChangeText={(date) => patchShow(i, { date })}
              />
              <Input accessibilityLabel="Name des Turniers" placeholder="Name" value={s.name} onChangeText={(n) => patchShow(i, { name: n })} />
              <Input accessibilityLabel="Klassen" placeholder="Klassen (optional)" value={s.classes} onChangeText={(classes) => patchShow(i, { classes })} />
              <Input accessibilityLabel="Helfer" placeholder="Helfer (optional)" value={s.helper} onChangeText={(helper) => patchShow(i, { helper })} />
              <Button
                label="Entfernen"
                variant="ghost"
                size="sm"
                icon={Trash2}
                onPress={() => patch({ shows: draft.shows.filter((_, j) => j !== i) })}
              />
            </Card>
          ) : (
            <Card key={i} className="gap-1">
              <Text variant="bodyStrong">{s.name}</Text>
              <Text variant="secondary">{formatDate(s.date)}</Text>
              {s.classes ? <Text variant="secondary">Klassen: {s.classes}</Text> : null}
              {s.helper ? <Text variant="secondary">Helfer: {s.helper}</Text> : null}
            </Card>
          ),
        )}
        {editable ? (
          <Button
            label="Turnier hinzufügen"
            variant="outline"
            icon={Plus}
            onPress={() => patch({ shows: [...draft.shows, { date: "", name: "", classes: "", helper: "" }] })}
          />
        ) : null}
      </View>

      <SectionLabel>Saisonende</SectionLabel>
      {editable ? (
        <Input
          accessibilityLabel="Saisonende"
          placeholder="Datum, z. B. 2026-10-31 (optional)"
          value={draft.seasonEnd}
          autoCapitalize="none"
          onChangeText={(seasonEnd) => patch({ seasonEnd })}
        />
      ) : (
        <Text variant="secondary">{draft.seasonEnd ? formatDate(draft.seasonEnd) : "Nicht festgelegt"}</Text>
      )}

      <SectionLabel>Rhythmus</SectionLabel>
      <Card className="gap-4">
        <NumberRow
          label="Einheiten pro Woche"
          min={draft.sessionsMin}
          max={draft.sessionsMax}
          editable={editable}
          onMin={(sessionsMin) => patch({ sessionsMin })}
          onMax={(sessionsMax) => patch({ sessionsMax })}
        />
        <NumberRow
          label="Ruhetage pro Woche"
          min={draft.restDaysMin}
          max={draft.restDaysMax}
          editable={editable}
          onMin={(restDaysMin) => patch({ restDaysMin })}
          onMax={(restDaysMax) => patch({ restDaysMax })}
        />
        <View className="gap-1.5">
          <Text variant="secondary">Höchstdauer pro Einheit in Minuten (0 = keine Grenze)</Text>
          {editable ? (
            <Input
              accessibilityLabel="Höchstdauer in Minuten"
              keyboardType="number-pad"
              value={draft.maxMinutes}
              onChangeText={(maxMinutes) => patch({ maxMinutes })}
            />
          ) : (
            <Text variant="body">{draft.maxMinutes} Min.</Text>
          )}
        </View>
        <Switch
          label="Ruhetag nach dem Turnier"
          value={draft.restAfterShow}
          disabled={!editable}
          onValueChange={(restAfterShow) => patch({ restAfterShow })}
        />
      </Card>

      {draft.riders.length > 0 ? (
        <>
          <SectionLabel>{editable ? "Reitbeteiligungen" : "Meine Regeln"}</SectionLabel>
          <View className="gap-3">
            {draft.riders.map((r) => (
              <Card key={r.userId} className="gap-3">
                <Text variant="bodyStrong">{r.name || "Reitbeteiligung"}</Text>
                <Text variant="secondary">Erlaubte Aktivitäten</Text>
                <View className="flex-row flex-wrap gap-2">
                  {ACTIVITIES.map((a) => (
                    <Pill
                      key={a}
                      label={activityLabel(a)}
                      selected={r.activities.includes(a)}
                      disabled={!editable}
                      onPress={() => patchRider(r.userId, { activities: toggleActivity(r.activities, a) })}
                    />
                  ))}
                </View>
                <Text variant="secondary">Höchste Belastung</Text>
                <View className="flex-row flex-wrap gap-2">
                  {MAX_INTENSITY_OPTIONS.map((o) => (
                    <Pill
                      key={o.value}
                      label={o.label}
                      selected={r.maxIntensity === o.value}
                      disabled={!editable}
                      onPress={() => patchRider(r.userId, { maxIntensity: o.value })}
                    />
                  ))}
                </View>
                <Switch
                  label="Darf allein ausreiten"
                  value={r.mayHackAlone}
                  disabled={!editable}
                  onValueChange={(mayHackAlone) => patchRider(r.userId, { mayHackAlone })}
                />
                <Switch
                  label="Darf Turniere reiten"
                  value={r.mayRideShows}
                  disabled={!editable}
                  onValueChange={(mayRideShows) => patchRider(r.userId, { mayRideShows })}
                />
              </Card>
            ))}
          </View>
        </>
      ) : null}

      {editable ? (
        <View className="gap-3">
          {error ? (
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          {save.isError ? (
            <Text variant="secondary" tone="danger" accessibilityRole="alert">
              {trainingError(save.error)}
            </Text>
          ) : null}
          {savedAt ? <Text variant="secondary" tone="primary">Gespeichert. Status: {statusLabel(draft.status)}.</Text> : null}
          <Button label="Speichern" size="lg" fullWidth loading={save.isPending} onPress={submit} />
        </View>
      ) : null}
    </Screen>
  );
}

function NumberRow({
  label,
  min,
  max,
  editable,
  onMin,
  onMax,
}: {
  label: string;
  min: string;
  max: string;
  editable: boolean;
  onMin: (v: string) => void;
  onMax: (v: string) => void;
}) {
  return (
    <View className="gap-1.5">
      <Text variant="secondary">{label}</Text>
      {editable ? (
        <View className="flex-row items-center gap-3">
          <Input accessibilityLabel={`${label}, von`} keyboardType="number-pad" value={min} onChangeText={onMin} className="w-20 text-center" />
          <Text variant="body">bis</Text>
          <Input accessibilityLabel={`${label}, bis`} keyboardType="number-pad" value={max} onChangeText={onMax} className="w-20 text-center" />
        </View>
      ) : (
        <Text variant="body">
          {min} bis {max}
        </Text>
      )}
    </View>
  );
}
