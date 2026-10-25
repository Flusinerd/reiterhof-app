import { useLocalSearchParams, useRouter, type Href } from "expo-router";
import { Check } from "lucide-react-native";
import { useEffect, useState } from "react";
import { Pressable, View } from "react-native";

import { DurationField, NoteField, TickRow } from "@/components/finish-parts";
import { Button, Card, PageHeader, Pill, Screen, Section, Switch, Text } from "@/components/ui";
import { useHorse } from "@/lib/api/horses";
import { trainingError, useCreateSession, useTrainingHorses, type CreatedSession, type SessionInput } from "@/lib/api/training";
import { useAuth } from "@/lib/auth";
import {
  FEELING_OPTIONS,
  buildSessionBody,
  durationCaption,
  exercisePractised,
  finishEyebrow,
  notePlaceholder,
  otherPerson,
  parseMinutes,
  savedTitle,
  stepsCaption,
  visibleLabel,
  type Feeling,
} from "@/lib/session-finish";
import { useExercise } from "@/lib/tracking-queries";
import { clearFinishExtras, loadFinishExtras, type FinishExtras } from "@/lib/tracking-store";
import { ACTIVITIES, activityLabel, gaitShareRows, parseFinishParams, reinSummary, type Activity, type FinishParams } from "@/lib/training";

/**
 * "Wie war's?" after the ride (docs/domains/training.md, "Finish"): the minutes and the place come
 * from the tracker and can be corrected, one tap for the feeling, the exercises of the plan are
 * ticked as far as they were done, an optional note, one button.
 */
export default function FinishSession() {
  const raw = useLocalSearchParams<Record<string, string | string[]>>();
  const params = parseFinishParams(raw);
  if (!params) {
    return (
      <Screen back>
        <Text variant="body">Angaben fehlen.</Text>
      </Screen>
    );
  }
  return <Finish params={params} />;
}

function Finish({ params }: { params: FinishParams }) {
  const router = useRouter();
  const { user } = useAuth();
  const horses = useTrainingHorses();
  const horseDetail = useHorse(params.horse);
  const create = useCreateSession(params.horse);
  const exercise = useExercise(params.exerciseId);

  const [activity, setActivity] = useState<Activity>(params.activity);
  const [changingPlace, setChangingPlace] = useState(false);
  const [minutesText, setMinutesText] = useState(String(params.minutes));
  const [feeling, setFeeling] = useState<Feeling | null>(null);
  const [practised, setPractised] = useState<boolean | null>(null);
  const [note, setNote] = useState("");
  const [visible, setVisible] = useState(true);
  const [saved, setSaved] = useState<CreatedSession | null>(null);

  // Track and gait windows of a tracked session (too big for router params, see lib/tracking-store).
  const startedAt = params.startedAt;
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

  const horse = horses.data?.find((h) => h.id === params.horse);
  const horseName = horse?.name ?? "Das Pferd";
  const other = otherPerson(horseDetail.data, user?.id);
  const minutes = parseMinutes(minutesText);
  const tracked = !!params.startedAt;
  const gaits = gaitShareRows(params.gaitShares);
  const rein = reinSummary(params.reinChanges);
  const exerciseTicked = practised ?? exercisePractised(params.stepsDone);
  // Back to the training screen of this horse that is still below in the stack (a `replace` stacked a second
  // one that showed the first horse). The week there is refreshed already: `create` waits for its effects.
  const leave = () => router.dismissTo({ pathname: "/training", params: { horse: params.horse } });

  if (saved) {
    return (
      <Screen back>
        <PageHeader
          eyebrow="Im Tagebuch"
          title={savedTitle(saved.session.activity, saved.session.minutes)}
          description="In der Woche eingetragen."
        />
        {saved.next_progression ? (
          <Section title="Als Nächstes">
            <Card className="gap-1">
              <Text variant="bodyStrong">{saved.next_progression.title}</Text>
              <Text variant="secondary">Nächste Stufe, wenn es sitzt.</Text>
            </Card>
          </Section>
        ) : null}
        <Button label="Fertig" size="lg" fullWidth icon={Check} onPress={leave} />
      </Screen>
    );
  }

  const save = () => {
    if (minutes === null) return;
    const body: SessionInput = {
      ...buildSessionBody({
        activity,
        minutes,
        startedAt: params.startedAt,
        gaitShares: params.gaitShares,
        reinChanges: params.reinChanges,
        distanceM: params.distanceM,
        exercise: params.exerciseId ? { id: params.exerciseId, practised: exerciseTicked } : undefined,
        feeling,
        note,
        visibleToRider: horse?.role === "owner" && other ? visible : undefined,
      }),
      ...(extras?.track ? { track: extras.track } : {}),
      ...(extras?.gait_windows ? { gait_windows: extras.gait_windows } : {}),
    };
    // The promise locks the button from the tap on: a second tap would store the session twice.
    return create
      .mutateAsync(body)
      .then((created) => {
        void clearFinishExtras();
        if (created.next_progression) setSaved(created);
        else leave();
      })
      .catch(() => undefined); // the error is shown from the mutation state
  };

  return (
    <Screen back menu={false} keyboardShouldPersistTaps="handled">
      <PageHeader eyebrow={finishEyebrow(horseName)} title="Wie war’s?" />

      <View className="gap-2">
        <DurationField value={minutesText} onChangeText={setMinutesText} caption={durationCaption(activity, tracked)} />
        {gaits.length > 0 ? (
          <Text variant="secondary">{gaits.map((g) => `${g.label} ${g.percent} %`).join(" · ")}</Text>
        ) : null}
        {rein ? (
          <Text variant="secondary">
            {`Linke Hand ${rein.leftPercent} % · Rechte Hand ${rein.rightPercent} % · ${rein.changes} Handwechsel`}
          </Text>
        ) : null}
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`Ort ändern, jetzt ${activityLabel(activity)}`}
          onPress={() => setChangingPlace((v) => !v)}
          className="min-h-touch justify-center self-start"
        >
          <Text variant="bodySm" tone="primary">
            {changingPlace ? "Fertig" : "Ort ändern"}
          </Text>
        </Pressable>
        {changingPlace ? (
          <View className="flex-row flex-wrap gap-2">
            {ACTIVITIES.map((a) => (
              <Pill key={a} label={activityLabel(a)} selected={activity === a} onPress={() => setActivity(a)} />
            ))}
          </View>
        ) : null}
      </View>

      <View className="gap-2">
        <Text variant="eyebrow">Gefühl</Text>
        <View className="flex-row flex-wrap gap-2">
          {FEELING_OPTIONS.map((f) => (
            <Pill key={f.value} label={f.label} selected={feeling === f.value} onPress={() => setFeeling(feeling === f.value ? null : f.value)} />
          ))}
        </View>
      </View>

      {exercise.data ? (
        <View>
          <Text variant="eyebrow">Übungen</Text>
          <TickRow
            label={exercise.data.title}
            hint={stepsCaption(params.stepsDone, exercise.data.step_count)}
            checked={exerciseTicked}
            onPress={() => setPractised(!exerciseTicked)}
          />
        </View>
      ) : null}

      <View className="gap-2">
        <NoteField value={note} onChangeText={setNote} placeholder={notePlaceholder(other)} />
        {horse?.role === "owner" && other ? (
          <Switch label={visibleLabel(other)} value={visible} onValueChange={setVisible} />
        ) : null}
      </View>

      {create.isError ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(create.error)}
        </Text>
      ) : null}
      <View className="gap-1">
        <Button
          label="Ins Tagebuch"
          size="lg"
          fullWidth
          disabled={minutes === null}
          loading={create.isPending || !extrasReady}
          onPress={save}
        />
        <Button
          label="Etwas aufgefallen?"
          variant="ghost"
          size="sm"
          fullWidth
          onPress={() => router.push(`/observations/new?horse=${params.horse}` as Href)}
        />
      </View>
    </Screen>
  );
}
