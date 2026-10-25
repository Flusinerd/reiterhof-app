import { useQueryClient } from "@tanstack/react-query";
import { Minus, Plus } from "lucide-react-native";
import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Text } from "@/components/ui";
import { useEffectMutation } from "@/lib/api/mutation";
import { reminderKeys, remindersApi, useReminderTime } from "@/lib/api/reminders";
import {
  REMINDER_TIME_MAX,
  REMINDER_TIME_MIN,
  reminderErrorMessage,
  stepReminderTime,
} from "@/lib/reminders";

/**
 * "Stallgasse": the time of the evening blanket reminder (`stables.reminder_time`). Admins change it
 * in quarter-hour steps (16:00 to 22:00) and save; everybody else only sees it.
 */
export function ReminderTimeCard() {
  const queryClient = useQueryClient();
  const query = useReminderTime();
  const saved = query.data?.reminder_time;
  const canEdit = query.data?.can_edit ?? false;
  const [draft, setDraft] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Follow the server value while nothing is being edited.
  useEffect(() => {
    if (draft !== null && draft === saved) setDraft(null);
  }, [draft, saved]);

  const save = useEffectMutation({
    mutationFn: (time: string) => remindersApi.setReminderTime(time),
    onMutate: () => setError(null),
    onSuccess: (info) => {
      queryClient.setQueryData(reminderKeys.reminderTime, info);
      setDraft(null);
    },
    onError: (e) => setError(reminderErrorMessage(e)),
    // The blanket tab and the reminder center show the time too.
    effects: () => [{ type: "reminderTime" }],
  });

  if (query.isPending) return <Text variant="body" tone="muted">Wird geladen ...</Text>;
  if (query.isError || !saved) {
    return (
      <View className="gap-3">
        <Text variant="body" tone="danger">
          {reminderErrorMessage(query.error)}
        </Text>
        <Button label="Erneut versuchen" variant="outline" onPress={() => query.refetch()} />
      </View>
    );
  }

  const shown = draft ?? saved;
  const changed = draft !== null && draft !== saved;
  return (
    <View className="gap-4 border-t border-divider pt-4">
      <View className="gap-1">
        <Text variant="bodyStrong">Erinnerung fürs Decken</Text>
        <Text variant="secondary">
          Erinnert die letzte Person im Stall, wenn Pferde ohne Eintrag offen sind. Danach alle 15 Minuten bis 22:00 Uhr.
        </Text>
      </View>
      {canEdit ? (
        <>
          <View className="flex-row items-center justify-between gap-4">
            <Button
              size="icon"
              variant="outline"
              icon={Minus}
              accessibilityLabel="15 Minuten früher"
              disabled={shown === REMINDER_TIME_MIN}
              onPress={() => setDraft(stepReminderTime(shown, -1))}
            />
            <Text variant="heroNumberSm" tone="brass" accessibilityLabel={`${shown} Uhr`}>
              {shown}
            </Text>
            <Button
              size="icon"
              variant="outline"
              icon={Plus}
              accessibilityLabel="15 Minuten später"
              disabled={shown === REMINDER_TIME_MAX}
              onPress={() => setDraft(stepReminderTime(shown, 1))}
            />
          </View>
          {error ? (
            <Text variant="bodySm" tone="danger" accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          {changed ? (
            <View className="flex-row gap-3">
              <Button label="Speichern" loading={save.isPending} onPress={() => save.mutate(shown)} />
              <Button label="Verwerfen" variant="ghost" disabled={save.isPending} onPress={() => setDraft(null)} />
            </View>
          ) : null}
        </>
      ) : (
        <Text variant="body">
          Aktuell um <Text variant="bodyStrong" tone="brass">{saved} Uhr</Text>. Nur Verwalter können das ändern.
        </Text>
      )}
    </View>
  );
}
