// The "Heute" screen (docs/design-system.md, "Abendlicht"): one next task and a timeline of today
// and tomorrow for the user's own horses, merged from the week plan, tonight's blankets and the
// reminder center. Pure helpers, no React Native imports (unit-tested). German UI text.

import type { Recommendation as BlanketRecommendation, TodayHorse } from "./blankets.ts";
import { formatClock, localDate } from "./presence-format.ts";
import type { ReminderItem } from "./reminders.ts";
import { activityLabel, type DayStatus } from "./training.ts";

/** The parts of a week plan day the timeline needs (see WeekDay in lib/api/training.ts). */
export type PlanDay = {
  date: string;
  status: DayStatus;
  is_me: boolean;
  user: { id: string; name: string } | null;
  activity: string | null;
  label?: string;
  minutes: number;
  exercise: { id: string; title: string } | null;
  show: { name: string; time?: string; tests: { name: string; time?: string }[] } | null;
};

export type HorsePlan = { horseId: string; horseName: string; days: readonly PlanDay[] };

export type TodayInput = {
  /** Stable-local calendar day, YYYY-MM-DD. */
  today: string;
  /** Stable-local time, "HH:MM". */
  nowClock: string;
  timeZone: string;
  /** Week plans of the user's horses (owner or rider). */
  plans: readonly HorsePlan[];
  /** Tonight's blankets of the user's horses and the stable's reminder time. */
  blankets: { reminderTime: string; horses: readonly TodayHorse[] } | null;
  /** Items of the reminder center (all groups). */
  reminders: readonly ReminderItem[];
};

export type EntryKind = "training" | "show" | "blanket" | "reminder";
export type EntryTone = "default" | "danger" | "warning";

export type TodayEntry = {
  id: string;
  kind: EntryKind;
  /** The reminder center's kind (health_due, medication, helper, ...) for reminder entries. */
  reminderKind?: string;
  day: "today" | "tomorrow";
  /** "HH:MM", or null for things without a time (overdue, all day, training without a time). */
  time: string | null;
  title: string;
  detail: string;
  tone: EntryTone;
  done: boolean;
  /** The user has to do it (own training, own horse's blanket, own reminders). */
  mine: boolean;
  /** App route to open, "" = none. */
  screen: string;
  horseId?: string;
  horseName?: string;
};

export type TodayView = {
  /** The one thing to do next, or null on a calm day. */
  next: TodayEntry | null;
  today: TodayEntry[];
  tomorrow: TodayEntry[];
};

function addDay(day: string, n: number): string {
  const d = new Date(`${day}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

function activityText(day: PlanDay): string {
  return day.label || (day.activity ? activityLabel(day.activity) : "Training");
}

function joinDetail(parts: (string | null | undefined | false)[]): string {
  return parts.filter((p): p is string => !!p).join(" · ");
}

function trainingEntry(plan: HorsePlan, day: PlanDay, when: "today" | "tomorrow"): TodayEntry | null {
  if (day.show) {
    const tests = day.show.tests.map((t) => (t.time ? `${t.time} ${t.name}` : t.name));
    return {
      id: `show:${plan.horseId}:${day.date}`,
      kind: "show",
      day: when,
      time: day.show.time ?? null,
      title: `${day.show.name} · ${plan.horseName}`,
      detail: tests.join(" · "),
      tone: "default",
      done: false,
      mine: true,
      screen: `/training/week?horse=${plan.horseId}`,
      horseId: plan.horseId,
      horseName: plan.horseName,
    };
  }
  if (!day.activity || day.status === "rest" || day.status === "empty") return null;
  const who = day.is_me ? "mit dir" : day.user ? day.user.name : "noch offen";
  const done = day.status === "done";
  const doer = day.is_me || !day.user;
  return {
    id: `training:${plan.horseId}:${day.date}`,
    kind: "training",
    day: when,
    time: null,
    title: done && !day.is_me && day.user ? `${plan.horseName} · ${day.user.name} war da` : `${plan.horseName} · ${activityText(day)}`,
    detail: joinDetail([day.minutes ? `${day.minutes} Min.` : "", day.exercise?.title, done ? "erledigt" : who]),
    tone: "default",
    done,
    mine: doer && !done,
    screen: `/training?horse=${plan.horseId}`,
    horseId: plan.horseId,
    horseName: plan.horseName,
  };
}

function blanketTitle(rec: BlanketRecommendation): string | null {
  if (rec.status === "blanket") return rec.blanket?.name ?? "Decke";
  if (rec.status === "none") return "Keine Decke";
  return null;
}

function blanketEntry(h: TodayHorse, reminderTime: string): TodayEntry | null {
  const what = blanketTitle(h.recommendation);
  if (!what) return null;
  const by = h.state?.changed_by?.name;
  return {
    id: `blanket:${h.horse.id}`,
    kind: "blanket",
    day: "today",
    time: reminderTime,
    title: `Decke für ${h.horse.name}`,
    detail: h.done
      ? joinDetail([what, by ? `erledigt von ${by}` : "erledigt"])
      : joinDetail([what, h.recommendation.manual ? "von Hand gewählt" : "", "noch niemand eingetragen"]),
    tone: "default",
    done: h.done,
    mine: !h.done,
    screen: "/blankets/round",
    horseId: h.horse.id,
    horseName: h.horse.name,
  };
}

/**
 * Pushes are written to be read on a lock screen ("Fällig: Luna" / "Wurmkur: in 7 Tagen (…).").
 * In the timeline the thing comes first: "Wurmkur · Luna" / "in 7 Tagen (…)".
 */
export function reminderText(kind: string, title: string, body: string): { title: string; detail: string } {
  const horse = /^(Fällig|Medikament|Reha): (.+)$/.exec(title)?.[2];
  const trimmed = body.replace(/\.$/, "");
  if (horse && (kind === "health_due" || kind === "medication")) {
    const m = (kind === "health_due" ? /^(.+?): (.+)$/ : /^(.+?), (.+)$/).exec(trimmed);
    if (m) return { title: `${m[1]} · ${horse}`, detail: kind === "medication" ? "Medikament" : m[2] };
  }
  return { title, detail: trimmed };
}

function reminderEntry(item: ReminderItem, input: TodayInput): TodayEntry | null {
  // Training reminders duplicate the plan entries; blanket checks are the blanket entries.
  if (item.kind === "training_plan" || item.kind === "last_person") return null;
  const placed = item.sent_at ?? item.due_at;
  const localDay = localDate(item.due_at, input.timeZone);
  const overdue = localDay < input.today;
  const day = localDay <= input.today ? "today" : localDay === addDay(input.today, 1) ? "tomorrow" : null;
  if (!day) return null;
  const text = reminderText(item.kind, item.title, item.body);
  return {
    id: `reminder:${item.id}`,
    kind: "reminder",
    reminderKind: item.kind,
    day,
    time: item.all_day || overdue ? null : formatClock(placed, input.timeZone) || null,
    title: text.title,
    detail: text.detail,
    tone: overdue ? "danger" : item.kind === "weather_warning" ? "warning" : "default",
    done: false,
    mine: true,
    screen: item.screen,
  };
}

/** Timed entries in time order, then entries without a time; done ones keep their place. */
function byTime(a: TodayEntry, b: TodayEntry): number {
  if (a.time && b.time) return a.time.localeCompare(b.time) || a.id.localeCompare(b.id);
  if (a.time) return -1;
  if (b.time) return 1;
  return a.id.localeCompare(b.id);
}

/**
 * The next task: own training of today first (it is what owners plan the day around), then the
 * earliest open timed task from now on, then overdue things. Done entries never.
 */
function pickNext(entries: readonly TodayEntry[], nowClock: string): TodayEntry | null {
  const open = entries.filter((e) => e.day === "today" && e.mine && !e.done);
  return (
    open.find((e) => e.kind === "training") ??
    open.filter((e) => e.time && e.time >= nowClock).sort(byTime)[0] ??
    open.find((e) => e.tone === "danger") ??
    open.find((e) => e.time !== null) ??
    null
  );
}

export function buildToday(input: TodayInput): TodayView {
  const tomorrowDay = addDay(input.today, 1);
  const entries: TodayEntry[] = [];
  for (const plan of input.plans) {
    for (const day of plan.days) {
      const when = day.date === input.today ? "today" : day.date === tomorrowDay ? "tomorrow" : null;
      if (!when) continue;
      const e = trainingEntry(plan, day, when);
      if (e) entries.push(e);
    }
  }
  if (input.blankets) {
    for (const h of input.blankets.horses) {
      const e = blanketEntry(h, input.blankets.reminderTime);
      if (e) entries.push(e);
    }
  }
  const seen = new Set<string>();
  for (const item of input.reminders) {
    if (seen.has(item.id)) continue;
    seen.add(item.id);
    const e = reminderEntry(item, input);
    if (e) entries.push(e);
  }
  const next = pickNext(entries, input.nowClock);
  const rest = entries.filter((e) => e !== next);
  return {
    next,
    today: rest.filter((e) => e.day === "today").sort(byTime),
    tomorrow: rest.filter((e) => e.day === "tomorrow").sort(byTime),
  };
}

/** Eyebrow of the next task: "Als Nächstes · 17:00", "Überfällig", "Heute". */
export function nextEyebrow(entry: TodayEntry): string {
  if (entry.tone === "danger") return "Überfällig";
  if (entry.time) return `Als Nächstes · ${entry.time}`;
  return "Heute";
}

/** Button label of the next task. */
export function nextAction(entry: TodayEntry): string {
  switch (entry.kind) {
    case "training":
      return "Training starten";
    case "blanket":
      return "Runde starten";
    case "show":
      return "Zum Turnier";
    default:
      switch (entry.reminderKind) {
        case "health_due":
          return "Zum Termin";
        case "helper":
        case "new_request":
          return "Zur Anfrage";
        case "reha_checkup":
          return "Zum Reha-Plan";
        default:
          return "Ansehen";
      }
  }
}

/** Line under the greeting: "Alles erledigt", "3 Dinge heute", "1 Sache heute". */
export function todaySummary(view: TodayView): string {
  const open = [view.next, ...view.today].filter((e): e is TodayEntry => !!e && !e.done && e.mine).length;
  if (open === 0) return "Für heute ist alles erledigt.";
  return open === 1 ? "Eine Sache steht heute an." : `${open} Dinge stehen heute an.`;
}

/** What "Heute" needs to know about each query behind it (a React Query result fits). */
export type TodaySource = { isPending: boolean; isError: boolean; data: unknown; error: unknown };

export type TodayLoad =
  | { state: "loading" }
  /** A query failed and has no earlier answer to fall back on: its empty list would read as "all done". */
  | { state: "failed"; error: unknown }
  /** Everything is there. `stale` when a later refresh failed, so the view comes from the cache. */
  | { state: "ready"; stale: boolean };

/**
 * Whether the view of "Heute" may be built. A failed query is no empty answer: without this check the
 * screen says "Für heute ist alles erledigt." (or offers "Pferd anlegen") after a failed request and hides
 * what is due.
 */
export function todayLoad(sources: readonly TodaySource[]): TodayLoad {
  const failed = sources.find((s) => s.isError && s.data === undefined);
  if (failed) return { state: "failed", error: failed.error };
  if (sources.some((s) => s.isPending)) return { state: "loading" };
  return { state: "ready", stale: sources.some((s) => s.isError) };
}
