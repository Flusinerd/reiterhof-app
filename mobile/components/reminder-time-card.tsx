import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Minus, Plus } from "lucide-react-native";
import { useEffect, useState } from "react";
import { View } from "react-native";

import { Button, Card, Text } from "@/components/ui";
import { blanketKeys } from "@/lib/api/blankets";
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

  const save = useMutation({
    mutationFn: (time: string) => remindersApi.setReminderTime(time),
    onMutate: () => setError(null),
    onSuccess: (info) => {
      queryClient.setQueryData(reminderKeys.reminderTime, info);
      setDraft(null);
      // The blanket tab and the reminder center show the time too.
      void queryClient.invalidateQueries({ queryKey: blanketKeys.all });
      void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
    },
    onError: (e) => setError(reminderErrorMessage(e)),
  });

  if (query.isPending) return <Card><Text variant="body" tone="muted">Wird geladen ...</Text></Card>;
  if (query.isError || !saved) {
    return (
      <Card className="gap-3">
        <Text variant="body" tone="danger">
          {reminderErrorMessage(query.error)}
        </Text>
        <Button label="Erneut versuchen" variant="outline" onPress={() => void query.refetch()} />
      </Card>
    );
  }

  const shown = draft ?? saved;
  const changed = draft !== null && draft !== saved;
  return (
    <Card className="gap-4">
      <View className="gap-1">
        <Text variant="bodyStrong">Erinnerung fürs Decken</Text>
        <Text variant="secondary">
          Zu dieser Uhrzeit erinnert die App die letzte Person im Stall, wenn noch Pferde ohne Decken-Eintrag sind. Danach alle 15 Minuten bis 22:00 Uhr.
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
            <Text variant="heroNumberSm" accessibilityLabel={`${shown} Uhr`}>
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
          Aktuell um <Text variant="bodyStrong">{saved} Uhr</Text>. Ändern können das nur Verwalter des Stalls.
        </Text>
      )}
    </Card>
  );
}
