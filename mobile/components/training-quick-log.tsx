import { useState } from "react";
import { View } from "react-native";

import { Button, Pill, Sheet, Text } from "@/components/ui";
import { trainingError, useCreateSession } from "@/lib/api/training";
import { ACTIVITIES, QUICK_DURATIONS, activityLabel, type Activity } from "@/lib/training";

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
  onSaved?: () => void;
}) {
  const [activity, setActivity] = useState<Activity | null>(null);
  const [minutes, setMinutes] = useState<number | null>(null);
  const create = useCreateSession(horseId);
  const choices = ACTIVITIES.filter((a) => !hidden.includes(a));

  const close = (next: boolean) => {
    if (!next) {
      setActivity(null);
      setMinutes(null);
      create.reset();
    }
    onOpenChange(next);
  };

  const save = () => {
    if (!activity || !minutes) return;
    create.mutate(
      { activity, minutes },
      {
        onSuccess: () => {
          close(false);
          onSaved?.();
        },
      },
    );
  };

  return (
    <Sheet open={open} onOpenChange={close} title="Nur eintragen" description={`Was hast du mit ${horseName} gemacht?`}>
      <View className="flex-row flex-wrap gap-2">
        {choices.map((a) => (
          <Pill key={a} label={activityLabel(a)} selected={activity === a} onPress={() => setActivity(a)} />
        ))}
      </View>
      <Text variant="label">Wie lange?</Text>
      <View className="flex-row flex-wrap gap-2">
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
