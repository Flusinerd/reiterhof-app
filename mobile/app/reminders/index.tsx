import { useQueryClient } from "@tanstack/react-query";
import type { Href } from "expo-router";
import { ActivityIndicator, RefreshControl, View } from "react-native";

import { ReminderHead } from "@/components/reminder-head";
import { ReminderRow } from "@/components/reminder-row";
import { Button, Card, Screen, SectionTitle, Text } from "@/components/ui";
import { useEffectMutation } from "@/lib/api/mutation";
import { reminderKeys, remindersApi, useReminders } from "@/lib/api/reminders";
import { pushOnce } from "@/lib/navigation";
import { mutationKeys } from "@/lib/query-keys";
import { useKeyedErrors } from "@/lib/use-keyed-errors";
import { usePendingIds } from "@/lib/use-pending";
import { groupByDay, reminderErrorMessage, type ReminderItem, type RemindersResponse } from "@/lib/reminders";
import { useColors } from "@/lib/theme";
import { useRefreshOnFocus } from "@/lib/use-refresh-on-focus";

/** Reminder center (JAN-69): tonight's blanket check, then the reminders of today and of each day of the week. */
export default function Reminders() {
  const c = useColors();
  const queryClient = useQueryClient();
  const reminders = useReminders("week");
  useRefreshOnFocus([reminderKeys.all]);
  // Every row that is being dismissed shows its own spinner, and a failure shows under that row.
  const dismissing = usePendingIds(mutationKeys.reminderDismiss, (item: ReminderItem) => item.id);
  const [errors, setError] = useKeyedErrors();

  const dismiss = useEffectMutation({
    mutationFn: (item: ReminderItem) => remindersApi.dismiss(item.id),
    mutationKey: mutationKeys.reminderDismiss,
    onMutate: (item) => setError(item.id, null),
    onSuccess: (_data, item) => {
      // Remove it at once; the refetch of the effect confirms.
      queryClient.setQueryData<RemindersResponse>(reminderKeys.list("week"), (old) =>
        old
          ? { ...old, groups: old.groups.map((g) => ({ ...g, items: g.items.filter((i) => i.id !== item.id) })) }
          : old,
      );
    },
    onError: (e, item) => setError(item.id, reminderErrorMessage(e)),
    effects: () => [{ type: "reminderDismissed" }],
    errorEffects: () => [{ type: "reminderDismissed" }],
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
            dismissing={dismissing.has(item.id)}
            error={errors[item.id]}
            onOpen={(screen) => pushOnce(screen as Href)}
            onDismiss={(it) => dismiss.mutateAsync(it).catch(() => undefined)}
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

      {reminders.isPending ? (
        <ActivityIndicator color={c.primary.DEFAULT} />
      ) : reminders.isError ? (
        <Card className="gap-3">
          <Text variant="body">{reminderErrorMessage(reminders.error)}</Text>
          <Button label="Erneut versuchen" variant="outline" onPress={() => reminders.refetch()} />
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
