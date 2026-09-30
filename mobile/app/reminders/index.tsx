import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { ReminderHead } from "@/components/reminder-head";
import { ReminderRow } from "@/components/reminder-row";
import { Button, Card, Divider, Screen, Section, Text } from "@/components/ui";
import { BLANKET_STATE_EVENT } from "@/lib/api/blankets";
import { reminderKeys, remindersApi, useReminders } from "@/lib/api/reminders";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { groupByDay, reminderErrorMessage, type ReminderItem, type RemindersResponse } from "@/lib/reminders";
import { colors } from "@/lib/theme";

const EMPTY_TEXT = { today: "Heute steht nichts an.", week: "Diese Woche steht nichts an." } as const;

/** Reminder center (JAN-69): tonight's blanket check, then today's and this week's reminders. */
export default function Reminders() {
  const queryClient = useQueryClient();
  const reminders = useReminders("week");
  useInvalidateOnEvents({
    [BLANKET_STATE_EVENT]: [reminderKeys.all],
    "request.changed": [reminderKeys.all],
  });
  const [error, setError] = useState<string | null>(null);
  const [dismissingId, setDismissingId] = useState<string | null>(null);

  const dismiss = useMutation({
    mutationFn: (item: ReminderItem) => remindersApi.dismiss(item.id),
    onMutate: (item) => {
      setError(null);
      setDismissingId(item.id);
    },
    onSuccess: (_data, item) => {
      // Remove it at once; the refetch below confirms.
      queryClient.setQueryData<RemindersResponse>(reminderKeys.list("week"), (old) =>
        old
          ? { ...old, groups: old.groups.map((g) => ({ ...g, items: g.items.filter((i) => i.id !== item.id) })) }
          : old,
      );
    },
    onError: (e) => setError(reminderErrorMessage(e)),
    onSettled: () => {
      setDismissingId(null);
      void queryClient.invalidateQueries({ queryKey: reminderKeys.all });
    },
  });

  const data = reminders.data;
  const timeZone = data?.timezone ?? "Europe/Berlin";

  function rows(items: ReminderItem[]) {
    return items.map((item, i) => (
      <View key={item.id}>
        {i > 0 ? <Divider /> : null}
        <ReminderRow
          item={item}
          timeZone={timeZone}
          dismissing={dismissingId === item.id}
          onOpen={(screen) => router.push(screen as Href)}
          onDismiss={(it) => dismiss.mutate(it)}
        />
      </View>
    ));
  }

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl
          refreshing={reminders.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: reminderKeys.all })}
          tintColor={colors.primary.DEFAULT}
        />
      }
    >
      <ReminderHead check={data?.blanket_check} />

      {error ? (
        <Text variant="bodySm" tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}

      {reminders.isPending ? (
        <ActivityIndicator color={colors.primary.DEFAULT} />
      ) : reminders.isError ? (
        <Card className="gap-3">
          <Text variant="body">{reminderErrorMessage(reminders.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void reminders.refetch()} />
        </Card>
      ) : (
        (data?.groups ?? []).map((group) => (
          <Section key={group.key} title={group.label}>
            {group.items.length === 0 ? (
              <Text variant="body" tone="muted">
                {EMPTY_TEXT[group.key]}
              </Text>
            ) : group.key === "today" ? (
              <Card padded={false}>{rows(group.items)}</Card>
            ) : (
              // "Diese Woche": one card per day with the day as heading.
              groupByDay(group.items, timeZone, data?.today ?? "").map((section) => (
                <View key={section.day} className="gap-2">
                  <Text variant="caption" className="px-1">
                    {section.heading}
                  </Text>
                  <Card padded={false}>{rows(section.items)}</Card>
                </View>
              ))
            )}
          </Section>
        ))
      )}
    </Screen>
  );
}
