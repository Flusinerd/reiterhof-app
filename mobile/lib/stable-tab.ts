// Pure helpers of the "Stall" tab and the request rows (no React Native imports, unit-tested):
// the sentence about who is at the stable, the board of horses with tonight's blanket status, and
// the texts of a help request shown as a message. All user-facing text is German.

import { firstName, progressText, reminderHint, type Today, type TodayHorse } from "./blankets.ts";
import { dayHeading } from "./reminders.ts";
import {
  acceptLabel,
  describeRule,
  formatDay,
  formatWhen,
  helperCountText,
  parseDate,
  requestTitle,
  addDays,
  type HelpRequest,
} from "./requests.ts";

// --- who is here ---------------------------------------------------------------------------

export type StablePerson = {
  id: string;
  name: string;
  colorKey: string | null;
  /** The signed-in user; written "Du" in the sentence and listed first. */
  me?: boolean;
};

type PresenceLike = {
  me: { open_visit: unknown | null };
  here: readonly { user_id: string; name: string; avatar_color: string | null }[];
};

/** Everybody at the stable right now: me first when I am checked in, then the others by name. */
export function herePeople(
  overview: PresenceLike | undefined,
  me: { id: string; name: string; avatar_color: string | null } | null | undefined,
): StablePerson[] {
  if (!overview) return [];
  const people: StablePerson[] = [];
  if (overview.me.open_visit && me) people.push({ id: me.id, name: me.name, colorKey: me.avatar_color, me: true });
  for (const p of overview.here) people.push({ id: p.user_id, name: p.name, colorKey: p.avatar_color });
  return people;
}

const MAX_NAMES = 3;

function joinNames(names: readonly string[]): string {
  if (names.length <= 1) return names[0] ?? "";
  return `${names.slice(0, -1).join(", ")} und ${names[names.length - 1]}`;
}

/** The sentence in two parts, so the screen can set the names strong: "Anna, Mia und Lea" and "sind da". */
export function presenceParts(
  people: readonly Pick<StablePerson, "name" | "me">[],
  loaded = true,
  failed = false,
): { names: string; rest: string } {
  // Nothing loaded and the request failed: say so (the row retries on a tap) instead of "loading" for ever.
  if (!loaded && failed) return { names: "Anwesenheit nicht geladen.", rest: "Tippen, um es noch mal zu versuchen." };
  if (!loaded) return { names: "Wird geladen …", rest: "" };
  if (people.length === 0) return { names: "Niemand im Stall", rest: "" };
  const names = people.map((p) => (p.me ? "Du" : firstName(p.name)));
  const shown = names.slice(0, MAX_NAMES);
  const more = names.length - shown.length;
  const parts = more > 0 ? [...shown, more === 1 ? "eine weitere Person" : `${more} weitere`] : shown;
  const verb = names.length === 1 ? (names[0] === "Du" ? "bist" : "ist") : "sind";
  return { names: joinNames(parts), rest: `${verb} da` };
}

/**
 * "Anna, Mia und Lea sind da", "Du bist da", "Anna ist da", "Du, Anna, Mia, Lea und 2 weitere sind da",
 * "Niemand im Stall". Without data: "Wird geladen …".
 */
export function presenceSentence(people: readonly Pick<StablePerson, "name" | "me">[], loaded = true, failed = false): string {
  const { names, rest } = presenceParts(people, loaded, failed);
  return rest ? `${names} ${rest}` : names;
}

/** The faces of the row: at most `max`, and how many more people there are. */
export function presenceFaces<T>(people: readonly T[], max = 4): { faces: T[]; extra: number } {
  return { faces: people.slice(0, max), extra: Math.max(0, people.length - max) };
}

// --- blanket board -------------------------------------------------------------------------

export type BoardStatus = "done" | "open" | "mine_open";

export type BoardHorse = {
  id: string;
  name: string;
  colorKey: string | null;
  status: BoardStatus;
  /** Spoken description: "Luna: dein Pferd, noch offen". */
  label: string;
};

/** One entry per horse in the order of the server: done, open, or one of my own horses still open. */
export function blanketBoard(horses: readonly TodayHorse[], mineIds: ReadonlySet<string> = new Set()): BoardHorse[] {
  return horses.map((h) => {
    const mine = h.is_mine || mineIds.has(h.horse.id);
    const status: BoardStatus = h.done ? "done" : mine ? "mine_open" : "open";
    const label =
      status === "done"
        ? `${h.horse.name}: versorgt`
        : status === "mine_open"
          ? `${h.horse.name}: dein Pferd, noch offen`
          : `${h.horse.name}: noch offen`;
    return { id: h.horse.id, name: h.horse.name, colorKey: h.horse.color_key, status, label };
  });
}

/** "Noch 4 Pferde offen · Erinnerung um 20:30", or "Alle versorgt" when nothing is open. */
export function boardLine(today: Pick<Today, "progress" | "reminder_time">): string {
  const open = today.progress.total - today.progress.done;
  if (today.progress.total === 0 || open <= 0) return progressText(today.progress);
  return `${progressText(today.progress)} · ${reminderHint(today.reminder_time)}`;
}

/** "Dein Pferd Luna ist noch offen." / "Deine Pferde Luna und Pepe sind noch offen." or null. */
export function boardMineLine(board: readonly BoardHorse[]): string | null {
  const names = board.filter((b) => b.status === "mine_open").map((b) => b.name);
  if (names.length === 0) return null;
  if (names.length === 1) return `Dein Pferd ${names[0]} ist noch offen.`;
  return `Deine Pferde ${joinNames(names)} sind noch offen.`;
}

// --- requests as messages ------------------------------------------------------------------

function relativeWord(date: string, today: string): string | null {
  if (date === today) return "heute";
  if (date === addDays(today, 1)) return "morgen";
  return null;
}

/**
 * When a request takes place: "morgen 07:00", "Sa, 31. Okt · 07:15–09:00", "Mo, 5. Okt – Mi, 7. Okt · 08:00".
 * `withDay` false is for lists grouped by day: only "07:00" or "ganztägig".
 */
export function requestWhen(r: Pick<HelpRequest, "date" | "date_end" | "time_from" | "time_to">, today: string, withDay = true): string {
  if (r.date_end && r.date_end !== r.date) return formatWhen(r);
  const time = r.time_from ? (r.time_to ? `${r.time_from}–${r.time_to}` : r.time_from) : "";
  if (!withDay) return time || "ganztägig";
  const word = relativeWord(r.date, today);
  if (word) return time ? `${word} ${time}` : word;
  const day = parseDate(r.date) ? formatDay(r.date) : r.date;
  return time ? `${day} · ${time}` : day;
}

/** "Jonas hilft", "Du und Jonas helfen", "Jonas, Mia und 1 weitere helfen" or "Noch niemand". */
export function helpersLine(r: Pick<HelpRequest, "helpers" | "helpers_count" | "helpers_needed" | "type" | "status">, userId?: string): string {
  if (r.status === "done") return "Erledigt";
  if (r.status === "cancelled") return "Abgesagt";
  if (r.helpers.length === 0) return "Noch niemand";
  const names = r.helpers.map((h) => (h.user_id === userId ? "Du" : firstName(h.name)));
  const shown = names.slice(0, 2);
  const rest = names.length - shown.length;
  const parts = rest > 0 ? [...shown, `${rest} weitere`] : shown;
  const verb = names.length === 1 ? (names[0] === "Du" ? "hilfst" : "hilft") : "helfen";
  const text = `${joinNames(parts)} ${verb}`;
  return r.helpers_needed > 1 ? `${text} · ${helperCountText(r.helpers_count, r.helpers_needed, r.type)}` : text;
}

export type RequestRowModel = {
  id: string;
  /** Who asked: the first name, or "Du". */
  name: string;
  colorKey: string | null;
  when: string;
  /** The message: the description, else what it is about. */
  text: string;
  /** Third line: what it is about (when there is a description) and where. */
  context: string;
  /** Label of the accept button, null when the user cannot accept. */
  action: string | null;
  helpers: string;
  state: "open" | "mine" | "helping" | "done" | "cancelled";
  /** Spoken description of the whole row. */
  a11y: string;
};

function messageText(r: HelpRequest): string {
  if (r.description) return r.description;
  const destination = r.payload?.destination;
  if (r.type === "ride_share" && typeof destination === "string" && destination) return `Mitfahrgelegenheit nach ${destination}`;
  return requestTitle(r);
}

/** The texts of one request row. `today` is the local YYYY-MM-DD; `withDay` false inside day groups. */
export function requestRow(r: HelpRequest, today: string, userId?: string, withDay = true): RequestRowModel {
  const name = r.is_creator ? "Du" : firstName(r.creator_name);
  const text = messageText(r);
  const context = [text === r.description ? requestTitle(r) : "", r.location, describeRule(r.recurring_rule, r.date)]
    .filter(Boolean)
    .join(" · ");
  const when = requestWhen(r, today, withDay);
  const state: RequestRowModel["state"] =
    r.status === "done" ? "done" : r.status === "cancelled" ? "cancelled" : r.is_creator ? "mine" : r.is_helper ? "helping" : "open";
  const helpers = helpersLine(r, userId);
  return {
    id: r.id,
    name,
    colorKey: r.creator_color,
    when,
    text,
    context,
    action: r.can_accept ? acceptLabel(r.type) : null,
    helpers,
    state,
    a11y: [`${name}, ${when}`, text, helpers].join(". "),
  };
}

export type RequestDay = { date: string; heading: string; items: HelpRequest[] };

/** Splits requests (sorted by date) into day groups with "Heute", "Morgen" or "Freitag, 2. Okt" as heading. */
export function groupRequestsByDay(requests: readonly HelpRequest[], today: string): RequestDay[] {
  const groups: RequestDay[] = [];
  for (const r of requests) {
    const last = groups[groups.length - 1];
    if (last && last.date === r.date) last.items.push(r);
    else groups.push({ date: r.date, heading: dayHeading(r.date, today), items: [r] });
  }
  return groups;
}
