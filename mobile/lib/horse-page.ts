// Pure logic of the horse page (docs/design-system.md, "Abendlicht"): hero texts, the three rows of
// "Heute für <Name>", the week strip and the diary. No React Native imports (unit-tested with
// node --test). German UI text.

import type { Override, Recommendation } from "./blankets.ts";
import { recommendationTitle } from "./blankets.ts";
import {
  ageText,
  dueDetailText,
  dueLevel,
  dueText,
  formatDate,
  healthKindLabel,
  horseRoutes,
  joinParts,
  sexLabel,
} from "./horse-format.ts";
import { categoryLabel, bodyPartLabel, observationRoute, statusLabel, urgencyLabel } from "./observations.ts";
import { activityLabel, addDays, FEEL_OPTIONS, formatDayLong, WEEKDAYS_LONG, WEEKDAYS_SHORT, type DayStatus } from "./training.ts";

// --- hero ----------------------------------------------------------------------------------

export type HeroHorse = {
  name: string;
  sex: string | null;
  birth_year: number | null;
  box: string | null;
  breed: string | null;
  owner: { name: string } | null;
  is_mine: boolean;
  i_ride: boolean;
};

/** Line above the name: "Stute · 11 Jahre · Box 12" (whatever is known). */
export function heroEyebrow(h: Pick<HeroHorse, "sex" | "birth_year" | "box">, now: Date): string {
  return joinParts([sexLabel(h.sex), ageText(h.birth_year, now), h.box ? `Box ${h.box}` : null]);
}

/** Line below the name: whose horse it is and the breed, e.g. "Dein Pferd · Hannoveraner". */
export function heroSubline(h: Pick<HeroHorse, "owner" | "is_mine" | "i_ride" | "breed">): string {
  const who = h.is_mine
    ? "Dein Pferd"
    : h.i_ride
      ? joinParts(["Du reitest", h.owner ? `Besitzer: ${h.owner.name}` : null])
      : h.owner
        ? `Besitzer: ${h.owner.name}`
        : "";
  return joinParts([who, h.breed]);
}

/** Label of the photo button: add a first photo or change the existing one. */
export function photoActionLabel(hasPhoto: boolean): string {
  return hasPhoto ? "Foto ändern" : "+ Foto hinzufügen";
}

// --- week strip ----------------------------------------------------------------------------

/** The parts of a week day the page needs (see WeekDay in lib/api/training.ts). */
export type PageDay = {
  date: string;
  status: DayStatus;
  is_today?: boolean;
  is_me: boolean;
  user: { id: string; name: string } | null;
  activity: string | null;
  label?: string;
  minutes: number;
  rest_reason?: string;
  exercise: { id: string; title: string } | null;
  show: { name: string; time?: string; tests: { name: string; time?: string }[] } | null;
  can_take?: boolean;
};

export type StripTone = "done" | "planned" | "show" | "rest" | "empty";

export type StripItem = {
  key: string;
  label: string;
  tone: StripTone;
  isToday: boolean;
  accessibilityLabel: string;
};

function weekdayIdx(date: string): number {
  return (new Date(`${date}T00:00:00Z`).getUTCDay() + 6) % 7;
}

/** Seven small marks: done, planned, show (brass), rest, nothing; today is highlighted. */
export function horseWeekStrip(days: readonly PageDay[], today: string): StripItem[] {
  return days.map((d) => {
    let tone: StripTone;
    let what: string;
    const act = d.activity ? activityLabel(d.activity) : "";
    if (d.show) {
      tone = "show";
      what = `Turnier ${d.show.name}`;
    } else if (d.status === "done") {
      tone = "done";
      what = act ? `erledigt, ${act}` : "erledigt";
    } else if (d.status === "rest") {
      tone = "rest";
      what = "Ruhetag";
    } else if (d.activity && (d.status === "planned" || d.status === "today" || d.status === "open")) {
      tone = "planned";
      what = `geplant, ${act}`;
    } else {
      tone = "empty";
      what = "nichts geplant";
    }
    const isToday = d.date === today;
    return {
      key: d.date,
      label: WEEKDAYS_SHORT[weekdayIdx(d.date)] ?? "",
      tone,
      isToday,
      accessibilityLabel: `${isToday ? "Heute" : formatDayLong(d.date)}: ${what}`,
    };
  });
}

// --- "Heute für <Name>" --------------------------------------------------------------------

export type RowAction = { label: string; kind: "start" | "blanket" | "done"; itemId?: string };

export type TodayRow = {
  key: "training" | "blanket" | "health";
  eyebrow: string;
  eyebrowTone: "muted" | "brass" | "success" | "danger" | "accent";
  title: string;
  detail?: string;
  action?: RowAction;
  /** Where tapping the row leads, "" = nowhere. */
  route: string;
};

export type RowBlanket = {
  recommendation: Recommendation;
  rule_recommendation?: Recommendation;
  override?: Pick<Override, "day_until"> | null;
  done: boolean;
  state: { changed_by: { name: string } | null } | null;
  /** The blanket day, YYYY-MM-DD. */
  day: string;
};

export type RowHealthItem = {
  id: string;
  kind: string;
  label: string;
  due_date: string | null;
  daily_time: string | null;
  days_until_due: number | null;
};

export type RowsInput = {
  horseId: string;
  today: string;
  /** Week plan of the current week; null when the user may not see the training of the horse. */
  days: readonly PageDay[] | null;
  blanket: RowBlanket | null;
  health: readonly RowHealthItem[] | null;
  canManage: boolean;
};

function nextPlannedText(days: readonly PageDay[], today: string): string | undefined {
  const next = [...days]
    .filter((d) => d.date > today)
    .sort((a, b) => a.date.localeCompare(b.date))
    .find((d) => d.show || (d.activity && d.status !== "rest" && d.status !== "empty"));
  if (!next) return undefined;
  const when = next.date === addDays(today, 1) ? "Morgen" : (WEEKDAYS_LONG[weekdayIdx(next.date)] ?? "");
  const what = next.show ? next.show.name : activityLabel(next.activity ?? "");
  const who = next.is_me ? "mit dir" : next.user ? `mit ${next.user.name}` : "";
  return [when, what, who].filter(Boolean).join(" ");
}

function trainingRow(days: readonly PageDay[], horseId: string, today: string): TodayRow {
  const week = `/training/week?horse=${horseId}`;
  const day = days.find((d) => d.date === today);
  if (day?.show) {
    const tests = day.show.tests.map((t) => (t.time ? `${t.time} ${t.name}` : t.name));
    return {
      key: "training",
      eyebrow: "Turnier · heute",
      eyebrowTone: "brass",
      title: day.show.name,
      detail: joinParts([day.show.time, ...tests]) || undefined,
      route: week,
    };
  }
  if (day?.activity && day.status !== "rest" && day.status !== "empty") {
    const done = day.status === "done";
    const who = day.is_me ? "heute mit dir" : day.user ? `heute mit ${day.user.name}` : "heute offen";
    const name = day.label || activityLabel(day.activity);
    const title = joinParts([day.minutes ? `${name} ${day.minutes} Min.` : name, day.exercise?.title]);
    const mine = day.is_me || (!day.user && day.can_take !== false);
    return {
      key: "training",
      eyebrow: done ? "Training · erledigt" : `Training · ${who}`,
      eyebrowTone: done ? "success" : "muted",
      title,
      action: !done && mine ? { label: "Starten", kind: "start" } : undefined,
      route: !done && mine ? `/training?horse=${horseId}` : week,
    };
  }
  return {
    key: "training",
    eyebrow: "Training",
    eyebrowTone: "muted",
    title: day?.status === "rest" ? (day.rest_reason === "after_show" ? "Ruhetag nach dem Turnier" : "Ruhetag") : "Heute frei",
    detail: nextPlannedText(days, today),
    route: week,
  };
}

function blanketRow(b: RowBlanket, horseId: string, canManage: boolean): TodayRow | null {
  const rec = b.recommendation;
  if (rec.status !== "blanket" && rec.status !== "none") return null;
  const by = b.state?.changed_by?.name;
  const detail = b.done
    ? by
      ? `Erledigt von ${by}`
      : "Erledigt"
    : rec.manual
      ? joinParts([
          "Von Hand gewählt",
          b.override ? (b.override.day_until ? `bis ${formatDate(b.override.day_until)}` : "bis auf Weiteres") : null,
        ])
      : rec.note || undefined;
  return {
    key: "blanket",
    eyebrow: b.done ? "Heute Nacht · erledigt" : "Heute Nacht",
    eyebrowTone: b.done ? "success" : "muted",
    title: recommendationTitle(rec),
    detail,
    action: canManage ? { label: "Ändern", kind: "blanket" } : undefined,
    route: horseRoutes.blanketPlan(horseId),
  };
}

function healthRow(items: readonly RowHealthItem[], horseId: string, canManage: boolean): TodayRow | null {
  const next = items
    .filter((i) => i.due_date && i.days_until_due !== null && !i.daily_time)
    .sort((a, b) => (a.days_until_due ?? 0) - (b.days_until_due ?? 0))[0];
  if (!next) return null;
  const days = next.days_until_due as number;
  const level = dueLevel(days);
  const name = next.label || healthKindLabel(next.kind);
  return {
    key: "health",
    eyebrow: "Gesundheit",
    eyebrowTone: level === "overdue" ? "danger" : level === "soon" ? "accent" : "muted",
    title: `${name} ${dueText(days)}`,
    detail: level === "overdue" ? dueDetailText(days).replace(/ überfällig$/, "") : formatDate(next.due_date),
    action: canManage && (level === "overdue" || level === "soon") ? { label: "Erledigt", kind: "done", itemId: next.id } : undefined,
    route: horseRoutes.health(horseId),
  };
}

/** Up to three rows: training today, tonight's blanket, the next health item. Empty ones are left out. */
export function buildHorseRows(input: RowsInput): TodayRow[] {
  const rows: TodayRow[] = [];
  if (input.days) rows.push(trainingRow(input.days, input.horseId, input.today));
  if (input.blanket) {
    const r = blanketRow(input.blanket, input.horseId, input.canManage);
    if (r) rows.push(r);
  }
  if (input.health) {
    const r = healthRow(input.health, input.horseId, input.canManage);
    if (r) rows.push(r);
  }
  return rows;
}

// --- diary ---------------------------------------------------------------------------------

export type DiaryLike = {
  type: "session" | "observation";
  id: string;
  at: string;
  day: string;
  user: { id: string; name: string };
  session: {
    label: string;
    minutes: number;
    note: string | null;
    feel: string | null;
    exercise_title: string | null;
  } | null;
  observation: {
    category: string | null;
    body_part: string | null;
    description: string | null;
    urgency: string;
    status: string;
    media_url: string | null;
    media_count: number;
  } | null;
};

export type DiaryDot = "success" | "accent" | "danger" | "muted";

export type DiaryView = {
  key: string;
  kind: "session" | "observation";
  /** "Gestern · Mia", "Dienstag · du". */
  when: string;
  title: string;
  /** Plain text under the title (the description of an observation). */
  body?: string;
  /** Small muted line (feel and exercise, status). */
  meta?: string;
  /** The rider's note, shown as an italic quote. */
  quote?: string;
  dot: DiaryDot;
  photoUrl: string | null;
  photoCount: number;
  /** Route to open, "" = none. */
  route: string;
};

/** "Heute", "Gestern", the weekday within the last week, else "12. Oktober". */
export function diaryDay(day: string, today: string): string {
  const diff = Math.round((Date.parse(`${today}T00:00:00Z`) - Date.parse(`${day}T00:00:00Z`)) / 86_400_000);
  if (diff <= 0) return "Heute";
  if (diff === 1) return "Gestern";
  if (diff < 7) return WEEKDAYS_LONG[weekdayIdx(day)] ?? "";
  // "Samstag, 12. Oktober" without the weekday.
  return formatDayLong(day).replace(/^[^,]+, /, "");
}

const FEEL_LABELS = Object.fromEntries(FEEL_OPTIONS.map((f) => [f.value, f.label]));

/** The display model of one diary entry. `meId` makes "du" out of the user's own name. */
export function diaryView(e: DiaryLike, ctx: { today: string; meId?: string }): DiaryView {
  const who = ctx.meId && e.user.id === ctx.meId ? "du" : e.user.name;
  const when = `${diaryDay(e.day, ctx.today)} · ${who}`;
  if (e.type === "observation" && e.observation) {
    const o = e.observation;
    const title = joinParts([categoryLabel(o.category), bodyPartLabel(o.body_part)]);
    return {
      key: `observation:${e.id}`,
      kind: "observation",
      when,
      title,
      body: o.description?.trim() || undefined,
      meta: joinParts([statusLabel(o.status), o.urgency === "info" ? null : urgencyLabel(o.urgency)]),
      dot: o.status === "done" ? "muted" : o.urgency === "urgent" ? "danger" : "accent",
      photoUrl: o.media_url,
      photoCount: o.media_count,
      route: observationRoute(e.id),
    };
  }
  const s = e.session;
  const name = s?.label ?? "Training";
  return {
    key: `session:${e.id}`,
    kind: "session",
    when,
    title: s && s.minutes > 0 ? `${name} ${s.minutes} Min.` : name,
    meta: joinParts([s?.feel ? FEEL_LABELS[s.feel] : null, s?.exercise_title]) || undefined,
    quote: s?.note?.trim() || undefined,
    dot: "success",
    photoUrl: null,
    photoCount: 0,
    route: "",
  };
}

/** Joins loaded pages into one list without doubles (a new entry shifts the pages). */
export function flattenDiary<T extends { type: string; id: string }>(pages: readonly { items: readonly T[] }[]): T[] {
  const seen = new Set<string>();
  const out: T[] = [];
  for (const page of pages) {
    for (const item of page.items) {
      const key = `${item.type}:${item.id}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push(item);
    }
  }
  return out;
}

/** Text of the empty diary. */
export const DIARY_EMPTY = "Noch keine Einträge. Sobald jemand trainiert oder etwas meldet, steht es hier.";
