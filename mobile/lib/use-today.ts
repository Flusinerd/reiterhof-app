// Data of the "Heute" screen: the week plans of the user's horses, tonight's blankets and the
// reminder center, merged by buildToday (lib/today.ts).

import { useQueries, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

import { useToday as useBlanketsToday } from "./api/blankets";
import { useReminders } from "./api/reminders";
import { trainingApi, trainingKeys, useTrainingHorses } from "./api/training";
import { weatherKeys } from "./api/weather";
import { useAuth } from "./auth";
import { buildToday, type HorsePlan, type TodayView } from "./today.ts";

function localParts(timeZone: string, now: Date): { today: string; clock: string; tomorrow: string } {
  const today = new Intl.DateTimeFormat("en-CA", { timeZone, year: "numeric", month: "2-digit", day: "2-digit" }).format(now);
  const clock = new Intl.DateTimeFormat("de-DE", { timeZone, hour: "2-digit", minute: "2-digit", hour12: false }).format(now);
  const d = new Date(`${today}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + 1);
  return { today, clock, tomorrow: d.toISOString().slice(0, 10) };
}

export type TodayData = {
  view: TodayView | null;
  timeZone: string;
  loading: boolean;
  refreshing: boolean;
  refresh: () => void;
  /** The user's horses (owner or rider), for the greeting line and the horse tab. */
  horses: { id: string; name: string; color_key: string }[];
};

export function useTodayData(): TodayData {
  const { me } = useAuth();
  const queryClient = useQueryClient();
  const timeZone = me?.stable?.timezone ?? "Europe/Berlin";
  const { today, clock, tomorrow } = localParts(timeZone, new Date());
  // The week of tomorrow differs on Sundays; fetch it too so tomorrow's show is not missing.
  const starts = new Date(`${tomorrow}T00:00:00Z`).getUTCDay() === 1 ? [today, tomorrow] : [today];

  const horses = useTrainingHorses();
  const blankets = useBlanketsToday();
  const reminders = useReminders("week");
  const list = horses.data ?? [];
  const weeks = useQueries({
    queries: list.flatMap((h) =>
      starts.map((start) => ({
        queryKey: trainingKeys.week(h.id, start === today ? undefined : start),
        queryFn: () => trainingApi.week(h.id, start === today ? undefined : start),
      })),
    ),
  });

  const weeksReady = weeks.every((w) => !w.isPending);
  const weekData = weeks.map((w) => w.data);
  const weeksVersion = weeks.map((w) => w.dataUpdatedAt).join(",");
  const view = useMemo(() => {
    if (horses.isPending || blankets.isPending || reminders.isPending || !weeksReady) return null;
    const plans: HorsePlan[] = list.map((h) => ({
      horseId: h.id,
      horseName: h.name,
      days: weekData.filter((w) => w?.horse_id === h.id).flatMap((w) => w?.days ?? []),
    }));
    const mine = new Set(list.map((h) => h.id));
    return buildToday({
      today,
      nowClock: clock,
      timeZone,
      plans,
      blankets: blankets.data
        ? { reminderTime: blankets.data.reminder_time, horses: blankets.data.horses.filter((h) => mine.has(h.horse.id)) }
        : null,
      reminders: reminders.data?.groups.flatMap((g) => g.items) ?? [],
    });
    // weekData is a new array on every render; weeksVersion changes when its contents do.
  }, [horses.isPending, blankets.data, reminders.data, weeksReady, weeksVersion, list, today, clock, timeZone]);

  return {
    view,
    timeZone,
    loading: view === null,
    refreshing: horses.isRefetching || blankets.isRefetching || reminders.isRefetching || weeks.some((w) => w.isRefetching),
    refresh: () => {
      void horses.refetch();
      void blankets.refetch();
      void reminders.refetch();
      for (const w of weeks) void w.refetch();
      void queryClient.invalidateQueries({ queryKey: weatherKeys.warnings }); // shown on Heute, too
    },
    horses: list.map((h) => ({ id: h.id, name: h.name, color_key: h.color_key })),
  };
}
