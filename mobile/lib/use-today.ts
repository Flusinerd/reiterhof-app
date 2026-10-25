// Data of the "Heute" screen: the week plans of the user's horses, tonight's blankets and the
// reminder center, merged by buildToday (lib/today.ts).

import { useQueries, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";

import { useToday as useBlanketsToday } from "./api/blankets";
import { useReminders } from "./api/reminders";
import { trainingApi, trainingKeys, useTrainingHorses } from "./api/training";
import { weatherKeys } from "./api/weather";
import { useAuth } from "./auth";
import { buildToday, todayLoad, type HorsePlan, type TodayLoad, type TodayView } from "./today.ts";

function localParts(timeZone: string, now: Date): { today: string; clock: string; tomorrow: string } {
  const today = new Intl.DateTimeFormat("en-CA", { timeZone, year: "numeric", month: "2-digit", day: "2-digit" }).format(now);
  const clock = new Intl.DateTimeFormat("de-DE", { timeZone, hour: "2-digit", minute: "2-digit", hour12: false }).format(now);
  const d = new Date(`${today}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + 1);
  return { today, clock, tomorrow: d.toISOString().slice(0, 10) };
}

export type TodayData = {
  /** Only built when every query has answered (see todayLoad): never from the empty lists of a failed request. */
  view: TodayView | null;
  timeZone: string;
  load: TodayLoad;
  refreshing: boolean;
  /** Reloads everything; the promise settles when that is over (a retry button spins until then). */
  refresh: () => Promise<unknown>;
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

  const load = todayLoad([horses, blankets, reminders, ...weeks]);
  const ready = load.state === "ready";
  const weekData = weeks.map((w) => w.data);
  const weeksVersion = weeks.map((w) => w.dataUpdatedAt).join(",");
  const view = useMemo(() => {
    if (!ready) return null;
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
  }, [ready, blankets.data, reminders.data, weeksVersion, list, today, clock, timeZone]);

  return {
    view,
    timeZone,
    load,
    refreshing: horses.isRefetching || blankets.isRefetching || reminders.isRefetching || weeks.some((w) => w.isRefetching),
    refresh: () =>
      Promise.all([
        horses.refetch(),
        blankets.refetch(),
        reminders.refetch(),
        ...weeks.map((w) => w.refetch()),
        queryClient.invalidateQueries({ queryKey: weatherKeys.warnings }), // shown on Heute, too
      ]),
    horses: list.map((h) => ({ id: h.id, name: h.name, color_key: h.color_key })),
  };
}
