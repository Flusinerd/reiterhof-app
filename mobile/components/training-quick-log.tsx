import { useState } from "react";
import { View } from "react-native";

import { Button, Pill, Sheet, Text } from "@/components/ui";
import { trainingError, useCreateSession, useWeek } from "@/lib/api/training";
import { ACTIVITIES, QUICK_DURATIONS, activityLabel, type Activity } from "@/lib/training";
import { quickLogBody } from "@/lib/training-focus";

/**
 * "Nur eintragen" (JAN-60): log a session in three taps. Tap 1: activity chip, tap 2: duration
 * chip, tap 3: save. Nothing is preselected so that the flow is always the same.
 */
export function QuickLogSheet({
  horseId,
  horseName,
  hidden,
  open,
  onOpenChange,
  onSaved,
}: {
  horseId: string;
  horseName: string;
  /** Activities that must not be offered (profile or rider rules). */
  hidden: readonly Activity[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called after the sheet closed, with what was logged (the screen behind says so). */
  onSaved?: (saved: { activity: Activity; minutes: number }) => void;
}) {
  const [activity, setActivity] = useState<Activity | null>(null);
  const [minutes, setMinutes] = useState<number | null>(null);
  const create = useCreateSession(horseId);
  // Today's slot gives the session its focus, as the finish screen does (the rotation reads it).
  const week = useWeek(horseId);
  const choices = ACTIVITIES.filter((a) => !hidden.includes(a));

  const close = (next: boolean) => {
    if (!next) {
      setActivity(null);
      setMinutes(null);
      create.reset();
    }
    onOpenChange(next);
  };

  // The promise locks the button from the tap on (a second tap would log the ride twice); the error shows below.
  const save = () => {
    if (!activity || !minutes) return;
    return create
      .mutateAsync(quickLogBody({ activity, minutes, days: week.data?.days }))
      .then(() => {
        close(false);
        onSaved?.({ activity, minutes });
      })
      .catch(() => undefined);
  };

  return (
    <Sheet open={open} onOpenChange={close} title="Nur eintragen" description={`Was hast du mit ${horseName} gemacht?`}>
      <Text variant="eyebrow">Was</Text>
      <View className="-mt-2 flex-row flex-wrap gap-2">
        {choices.map((a) => (
          <Pill key={a} label={activityLabel(a)} selected={activity === a} onPress={() => setActivity(a)} />
        ))}
      </View>
      <Text variant="eyebrow">Wie lange</Text>
      <View className="-mt-2 flex-row flex-wrap gap-2">
        {QUICK_DURATIONS.map((m) => (
          <Pill key={m} label={`${m} Min.`} selected={minutes === m} onPress={() => setMinutes(m)} />
        ))}
      </View>
      {create.isError ? (
        <Text variant="secondary" tone="danger" accessibilityRole="alert">
          {trainingError(create.error)}
        </Text>
      ) : null}
      <Button
        label="Speichern"
        fullWidth
        disabled={!activity || !minutes}
        loading={create.isPending}
        onPress={save}
      />
    </Sheet>
  );
}
