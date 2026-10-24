import { useMutation, useQueryClient } from "@tanstack/react-query";
import { router, type Href } from "expo-router";
import { useState } from "react";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { ReminderHead } from "@/components/reminder-head";
import { ReminderRow } from "@/components/reminder-row";
import { Button, Card, Screen, SectionTitle, Text } from "@/components/ui";
import { BLANKET_STATE_EVENT } from "@/lib/api/blankets";
import { reminderKeys, remindersApi, useReminders } from "@/lib/api/reminders";
import { useInvalidateOnEvents } from "@/lib/realtime";
import { groupByDay, reminderErrorMessage, type ReminderItem, type RemindersResponse } from "@/lib/reminders";
import { useColors } from "@/lib/theme";

/** Reminder center (JAN-69): tonight's blanket check, then the reminders of today and of each day of the week. */
export default function Reminders() {
  const c = useColors();
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

  /** The rows of one day, on the page with hairlines. */
  function rows(items: ReminderItem[]) {
    return (
      <View>
        {items.map((item, i) => (
          <ReminderRow
            key={item.id}
            item={item}
            timeZone={timeZone}
            today={data?.today ?? ""}
            last={i === items.length - 1}
            dismissing={dismissingId === item.id}
            onOpen={(screen) => router.push(screen as Href)}
            onDismiss={(it) => dismiss.mutate(it)}
          />
        ))}
      </View>
    );
  }

  const todayItems = data?.groups.find((g) => g.key === "today")?.items ?? [];
  const weekItems = data?.groups.find((g) => g.key === "week")?.items ?? [];
  const weekDays = groupByDay(weekItems, timeZone, data?.today ?? "");

  return (
    <Screen
      back
      refreshControl={
        <RefreshControl
          refreshing={reminders.isRefetching}
          onRefresh={() => void queryClient.invalidateQueries({ queryKey: reminderKeys.all })}
          tintColor={c.primary.DEFAULT}
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
        <ActivityIndicator color={c.primary.DEFAULT} />
      ) : reminders.isError ? (
        <Card className="gap-3">
          <Text variant="body">{reminderErrorMessage(reminders.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => void reminders.refetch()} />
        </Card>
      ) : (
        <>
          <View className="gap-1">
            <SectionTitle>Heute</SectionTitle>
            {todayItems.length === 0 ? (
              <Text variant="body" tone="muted" className="py-2">
                Heute steht nichts an.
              </Text>
            ) : (
              rows(todayItems)
            )}
          </View>
          {weekDays.map((section) => (
            <View key={section.day} className="gap-1">
              <SectionTitle>{section.heading}</SectionTitle>
              {rows(section.items)}
            </View>
          ))}
          {weekDays.length === 0 ? (
            <Text variant="body" tone="muted">
              Diese Woche steht sonst nichts an.
            </Text>
          ) : null}
        </>
      )}
    </Screen>
  );
}
