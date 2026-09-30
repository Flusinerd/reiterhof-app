import { useQuery } from "@tanstack/react-query";
import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { Check, ChevronRight, TriangleAlert } from "lucide-react-native";
import { useEffect, useState } from "react";
import { View } from "react-native";

import { ActivityIcon } from "@/components/training-activity-icon";
import {
  Button,
  Card,
  Icon,
  Input,
  PageHeader,
  Pill,
  PressableCard,
  Screen,
  Section,
  Switch,
  Text,
} from "@/components/ui";
import {
  trainingApi,
  trainingError,
  useCreateSession,
  useTrainingHorses,
  type CreatedSession,
  type SessionInput,
} from "@/lib/api/training";
import { clearFinishExtras, loadFinishExtras, type FinishExtras } from "@/lib/tracking-store";
import {
  FEEL_OPTIONS,
  FOCUS_OPTIONS,
  activityLabel,
  feelPrompt,
  formatMinutes,
  gaitShareRows,
  parseFinishParams,
  reinSummary,
  type Feel,
} from "@/lib/training";

/** Finish screen of a session (JAN-61): summary, feel, focus, note, visibility, save. */
export default function FinishSession() {
  const router = useRouter();
  const raw = useLocalSearchParams<Record<string, string | string[]>>();
  const params = parseFinishParams(raw);
  const horses = useTrainingHorses();
  const create = useCreateSession(params?.horse ?? "");
  const exercise = useQuery({
    queryKey: ["training", "exercise", params?.exerciseId],
    queryFn: () => trainingApi.exercise(params!.exerciseId!),
    enabled: !!params?.exerciseId,
  });

  const [feel, setFeel] = useState<Feel | null>(null);
  const [focus, setFocus] = useState<1 | 2 | 3 | null>(null);
  const [note, setNote] = useState("");
  const [visible, setVisible] = useState(true);
  const [saved, setSaved] = useState<CreatedSession | null>(null);
  // Track and gait windows of a tracked session (too big for router params, see lib/tracking-store).
  const startedAt = params?.startedAt;
  const [extras, setExtras] = useState<FinishExtras | null>(null);
  const [extrasReady, setExtrasReady] = useState(!startedAt);
  useEffect(() => {
    if (!startedAt) return;
    let cancelled = false;
    void loadFinishExtras(startedAt).then((e) => {
      if (cancelled) return;
      setExtras(e);
      setExtrasReady(true);
    });
    return () => {
      cancelled = true;
    };
  }, [startedAt]);

  if (!params) {
    return (
      <Screen back>
        <Text variant="body">Angaben fehlen.</Text>
      </Screen>
    );
  }

  const horse = horses.data?.find((h) => h.id === params.horse);
  const horseName = horse?.name ?? "Das Pferd";
  const gaits = gaitShareRows(params.gaitShares);
  const rein = reinSummary(params.reinChanges);
  const leave = () => router.replace("/(tabs)/training");

  if (saved) {
    return (
      <Screen back>
        <PageHeader
          eyebrow="Gespeichert"
          title={`${activityLabel(saved.session.activity)}, ${formatMinutes(saved.session.minutes)}`}
          description="In der Woche eingetragen."
        />
        {saved.next_progression ? (
          <Section title="Als Nächstes">
            <Card className="gap-1">
              <Text variant="bodyStrong">{saved.next_progression.title}</Text>
              <Text variant="secondary">
                Nächste Stufe, wenn es sitzt.
              </Text>
            </Card>
          </Section>
        ) : null}
        <Button label="Fertig" size="lg" fullWidth icon={Check} onPress={leave} />
      </Screen>
    );
  }

  const save = () => {
    const body: SessionInput = {
      activity: params.activity,
      minutes: params.minutes,
      ...(params.startedAt ? { started_at: params.startedAt } : {}),
      ...(params.gaitShares ? { gait_shares: params.gaitShares } : {}),
      ...(params.reinChanges ? { rein_changes: params.reinChanges } : {}),
      ...(params.distanceM ? { distance_m: params.distanceM } : {}),
      ...(params.exerciseId ? { exercise_id: params.exerciseId } : {}),
      ...(extras?.track ? { track: extras.track } : {}),
      ...(extras?.gait_windows ? { gait_windows: extras.gait_windows } : {}),
      ...(feel ? { feel } : {}),
      ...(focus ? { focus_rating: focus } : {}),
      ...(note.trim() ? { note: note.trim() } : {}),
      visible_to_rider: visible,
    };
    create.mutate(body, {
      onSuccess: (created) => {
        setSaved(created);
        void clearFinishExtras();
      },
    });
  };

  return (
    <Screen back keyboardShouldPersistTaps="handled">
      <PageHeader
        eyebrow="Einheit abschließen"
        title={activityLabel(params.activity)}
        value={String(params.minutes)}
        valueSize="sm"
        unit="Minuten"
      >
        <View className="gap-1">
          {gaits.length > 0 ? (
            <View className="flex-row items-center gap-2">
              <ActivityIcon activity={params.activity} size={18} />
              <Text variant="secondary">{gaits.map((g) => `${g.label} ${g.percent} %`).join(" · ")}</Text>
            </View>
          ) : null}
          {rein ? (
            <Text variant="secondary">
              {`Linke Hand ${rein.leftPercent} % · Rechte Hand ${rein.rightPercent} % · ${rein.changes} Handwechsel`}
            </Text>
          ) : null}
          {exercise.data ? <Text variant="secondary">Übung: {exercise.data.title}</Text> : null}
        </View>
      </PageHeader>

      <Section title={feelPrompt(horseName)}>
        <View className="flex-row flex-wrap gap-2">
          {FEEL_OPTIONS.map((f) => (
            <Pill key={f.value} label={f.label} selected={feel === f.value} onPress={() => setFeel(feel === f.value ? null : f.value)} />
          ))}
        </View>
      </Section>

      <Section title="Wie lief der Fokus?">
        <View className="flex-row flex-wrap gap-2">
          {FOCUS_OPTIONS.map((f) => (
            <Pill key={f.value} label={f.label} selected={focus === f.value} onPress={() => setFocus(focus === f.value ? null : f.value)} />
          ))}
        </View>
      </Section>

      <Section title="Notiz">
        <Input
          accessibilityLabel="Notiz"
          placeholder="Was war dir wichtig?"
          value={note}
          onChangeText={setNote}
          multiline
          maxLength={2000}
          className="h-28 py-3"
          textAlignVertical="top"
        />
      </Section>

      {horse?.role === "owner" ? (
        <Switch
          label="Reitbeteiligung sieht das"
          description="In Woche und Verlauf sichtbar."
          value={visible}
          onValueChange={setVisible}
        />
      ) : null}

      <PressableCard
        shape="tile"
        className="flex-row items-center gap-3"
        accessibilityLabel="Etwas aufgefallen?"
        onPress={() => router.push(`/observations/new?horse=${params.horse}` as Href)}
      >
        <Icon as={TriangleAlert} size={20} className="text-accent" />
        <Text variant="bodyStrong" className="flex-1">
          Etwas aufgefallen?
        </Text>
        <Icon as={ChevronRight} size={20} className="text-muted" />
      </PressableCard>

      {create.isError ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(create.error)}
        </Text>
      ) : null}
      <Button label="Speichern" size="lg" fullWidth loading={create.isPending || !extrasReady} onPress={save} />
    </Screen>
  );
}
