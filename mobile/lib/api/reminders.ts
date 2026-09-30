// Reminder center and notification settings (backend: internal/reminders;
// docs/domains/reminders.md) with React Query keys and hooks.

import { useQuery } from "@tanstack/react-query";

import { authed } from "../api";
import type {
  NotificationKind,
  NotificationSettings,
  ReminderTimeInfo,
  RemindersResponse,
} from "../reminders.ts";

export type ReminderRange = "today" | "week";

export const reminderKeys = {
  all: ["reminders"] as const,
  list: (range: ReminderRange) => ["reminders", "list", range] as const,
  notifications: ["settings", "notifications"] as const,
  reminderTime: ["settings", "reminder-time"] as const,
};

export const remindersApi = {
  list: (range: ReminderRange = "week") => authed.get<RemindersResponse>(`/api/v1/reminders?range=${range}`),
  /** 204. Only the user's own stored reminders can be dismissed. */
  dismiss: (id: string) => authed.post<void>(`/api/v1/reminders/${encodeURIComponent(id)}/dismiss`),
  notifications: () => authed.get<NotificationSettings>("/api/v1/settings/notifications"),
  setNotification: (kind: string, enabled: boolean) =>
    authed.put<NotificationKind>(`/api/v1/settings/notifications/${encodeURIComponent(kind)}`, { enabled }),
  reminderTime: () => authed.get<ReminderTimeInfo>("/api/v1/stables/reminder-time"),
  /** Admin only (403 otherwise). "HH:MM" between 16:00 and 22:00. */
  setReminderTime: (reminderTime: string) =>
    authed.put<ReminderTimeInfo>("/api/v1/stables/reminder-time", { reminder_time: reminderTime }),
};

export function useReminders(range: ReminderRange = "week") {
  return useQuery({ queryKey: reminderKeys.list(range), queryFn: () => remindersApi.list(range) });
}

export function useNotificationSettings() {
  return useQuery({ queryKey: reminderKeys.notifications, queryFn: remindersApi.notifications });
}

export function useReminderTime() {
  return useQuery({ queryKey: reminderKeys.reminderTime, queryFn: remindersApi.reminderTime });
}
