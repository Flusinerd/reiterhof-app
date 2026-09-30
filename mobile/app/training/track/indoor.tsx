import { Stack, useLocalSearchParams, useRouter } from "expo-router";
import { ArrowLeftRight, Play } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Pressable, View } from "react-native";

import { ExerciseChecklist } from "@/components/exercise-checklist";
import { GaitChip, ProgressBar, TrackingControls, useLeaveGuard } from "@/components/tracking-parts";
import { Button, LivePanel, PageHeader, Screen, Section, Text } from "@/components/ui";
import { cn } from "@/lib/cn";
import type { Gait } from "@/lib/gait";
import { isWeb } from "@/lib/platform";
import { activityLabel, formatClock, isActivity } from "@/lib/training";
import { formatReinMinutes, gaitLabel, reinLabel } from "@/lib/tracking-format";
import type { Snapshot } from "@/lib/tracking-persist";
import { useExercise } from "@/lib/tracking-queries";
import { useStoredSession, useTrackingSession } from "@/lib/tracking-session";
import { targetLabel, targetProgress } from "@/lib/tracking";

type Params = { horse?: string; activity?: string; minutes?: string; exercise?: string; resume?: string };

const CORRECTION_GAITS: Gait[] = ["walk", "trot", "canter"];

/** Session in the hall or arena (JAN-65): no GPS, gait from the accelerometer, big buttons. */
export default function IndoorTrack() {
  const params = useLocalSearchParams<Params>();
  const stored = useStoredSession(params.resume === "1");
  if (stored.loading) {
    return (
      <Screen scroll={false} className="justify-center">
        <ActivityIndicator accessibilityLabel="Einheit wird geladen" />
      </Screen>
    );
  }
  const snapshot = params.resume === "1" ? stored.snapshot : null;
  const horse = snapshot?.horse ?? params.horse;
  const activity = snapshot?.activity ?? params.activity;
  if (!horse || !isActivity(activity)) {
    return (
      <Screen back>
        <Text variant="body">Angaben fehlen. Starte die Einheit im Tab „Training“.</Text>
      </Screen>
    );
  }
  const minutes = snapshot?.targetMinutes ?? (Number(params.minutes) > 0 ? Math.round(Number(params.minutes)) : undefined);
  return (
    <IndoorRun
      horse={horse}
      activity={activity}
      exerciseId={snapshot?.exerciseId ?? params.exercise}
      targetMinutes={minutes}
      snapshot={snapshot}
    />
  );
}

function IndoorRun({
  horse,
  activity,
  exerciseId,
  targetMinutes,
  snapshot,
}: {
  horse: string;
  activity: string;
  exerciseId?: string;
  targetMinutes?: number;
  snapshot: Snapshot | null;
}) {
  const router = useRouter();
  const t = useTrackingSession({ mode: "indoor", horse, activity, exerciseId, targetMinutes, initial: snapshot });
  const exercise = useExercise(exerciseId);
  const [started, setStarted] = useState(snapshot !== null);
  const [busy, setBusy] = useState(false);
  useLeaveGuard(started, t.pause);

  const start = async () => {
    setBusy(true);
    await t.begin();
    setBusy(false);
    setStarted(true);
  };

  const resume = async () => {
    setBusy(true);
    await t.resume();
    setBusy(false);
  };

  const finish = async () => {
    setBusy(true);
    const { params } = await t.finish();
    router.replace({ pathname: "/training/session/finish", params });
  };

  const label = activityLabel(activity);
  const seconds = t.stats.activeSeconds;
  const steps = exercise.data?.steps ?? [];

  if (!started) {
    return (
      <Screen back>
        <PageHeader
          eyebrow={label}
          title="Bereit"
          description={targetMinutes ? `Ziel: ${targetMinutes} Minuten.` : "Die Zeit läuft, sobald du startest."}
        />
        <View className="gap-2">
          <Text variant="bodyStrong">Gangart und Hand</Text>
          <Text variant="secondary">
            Das Telefon erkennt Schritt, Trab und Galopp an der Bewegung, ohne GPS. Trag es am Körper. Liegt die
            Erkennung falsch, tipp die richtige Gangart an. „Handwechsel“ zählt die Zeit auf der anderen Hand.
          </Text>
          {isWeb ? (
            <Text variant="secondary" tone="accent">
              Bildschirm anlassen, sonst liefert der Browser keine Bewegungsdaten.
            </Text>
          ) : null}
        </View>
        {exerciseId && exercise.data ? (
          <Text variant="secondary">Übung: {exercise.data.title}</Text>
        ) : null}
        <Button label="Einheit starten" icon={Play} size="lg" fullWidth loading={busy} onPress={start} />
      </Screen>
    );
  }

  return (
    <Screen menu={false}>
      <Stack.Screen options={{ gestureEnabled: false }} />
      <LivePanel eyebrow={`${label}${t.paused ? " · pausiert" : ""}`} value={formatClock(seconds)}>
        {t.paused ? (
          <Text variant="secondary" className="text-white/70">
            Pausiert.
          </Text>
        ) : (
          <GaitChip gait={t.gait} />
        )}
        {targetMinutes ? (
          <View className="gap-2">
            <ProgressBar value={targetProgress(seconds, targetMinutes)} />
            <Text variant="secondary" className="text-white/70">
              {`Ziel ${targetMinutes} Min. · ${targetLabel(seconds, targetMinutes)}`}
            </Text>
          </View>
        ) : null}
      </LivePanel>

      <Section title="Gangart korrigieren">
        <View className="flex-row gap-3">
        {CORRECTION_GAITS.map((g) => (
          <Pressable
            key={g}
            accessibilityRole="button"
            accessibilityLabel={`Gangart ist ${gaitLabel(g)}`}
            accessibilityState={{ selected: t.gait === g, disabled: t.paused || t.windows.length === 0 }}
            disabled={t.paused || t.windows.length === 0}
            onPress={() => t.correct(g)}
            className={cn(
              "h-24 flex-1 items-center justify-center rounded-tile border",
              t.gait === g ? "border-primary bg-primary-soft" : "border-border bg-card active:bg-background",
              (t.paused || t.windows.length === 0) && "opacity-50",
            )}
          >
            <Text variant="bodyStrong" className={cn("text-title", t.gait === g && "text-primary-deep")}>
              {gaitLabel(g)}
            </Text>
          </Pressable>
        ))}
        </View>
        <Text variant="secondary">
          {t.sensorAvailable === false
            ? "Kein Bewegungssensor, keine Gangart-Erkennung."
            : "Nur tippen, wenn die Erkennung falsch liegt."}
        </Text>
      </Section>

      <Section title="Hand">
        <View className="flex-row items-center justify-between">
          <Text variant="bodyStrong">{t.rein ? reinLabel(t.rein) : "–"}</Text>
          <Text variant="secondary">{formatReinMinutes(t.reinMinutes)} auf dieser Hand</Text>
        </View>
        <Button
          label="Handwechsel"
          icon={ArrowLeftRight}
          size="lg"
          variant="secondary"
          fullWidth
          disabled={t.paused}
          onPress={t.changeRein}
        />
      </Section>

      {exercise.data && steps.length > 0 ? (
        <Section title="Ablauf">
          <ExerciseChecklist title={exercise.data.title} steps={steps} checked={t.checked} onToggle={t.toggleStep} />
        </Section>
      ) : null}

      <TrackingControls paused={t.paused} busy={busy} onPause={t.pause} onResume={resume} onFinish={finish} />
    </Screen>
  );
}
