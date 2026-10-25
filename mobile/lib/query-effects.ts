// What a change makes stale. Every mutation of the app names the effect it has and every realtime event
// names the keys it refreshes; the screens never list query keys themselves. Pure, no React Native
// imports (unit-tested, see query-effects.test.ts, which also checks that every input of "Heute" is
// reached by the effects that change it).
//
// Why a table and not a call per screen: the same rows are read by several screens (a health item by the
// health list, the horse page, the reminder center and Heute; a session by the week, the diary and the
// reminders), and a mutation only knows its own screen. Adding a mutation means adding or extending an
// effect here, with a test.

import { deletedHorses } from "./horse-delete.ts";
import {
  blanketKeys,
  CONSENTS_KEY,
  dressageKeys,
  horseKeys,
  ME_KEY,
  observationKeys,
  PRESENCE_KEY,
  rehaKeys,
  reminderKeys,
  requestKeys,
  trainingKeys,
  weatherKeys,
} from "./query-keys.ts";

/** A query key, as TanStack Query takes it (a prefix matches every key below it). */
export type Key = readonly unknown[];

export type Effect =
  /** A health item was added, edited, deleted, marked done or undone. */
  | { type: "health" }
  /** A horse was created (it is a new horse for the lists, the board, the roles and the request options). */
  | { type: "horseCreated" }
  /** A horse was edited: name, color, photo, owner, helper note. */
  | { type: "horseUpdated"; horseId: string }
  /**
   * A horse was deleted for good. Its own queries are removed instead of refetched (they would answer 404,
   * see `effectRemovedKeys`), everything that listed or counted it is refetched.
   */
  | { type: "horseDeleted"; horseId: string }
  /** A rider was set or removed. */
  | { type: "riders"; horseId: string }
  | { type: "emergency"; horseId: string }
  | { type: "documents"; horseId: string }
  /** A training session was logged (finish screen, quick log). */
  | { type: "session"; horseId: string }
  /** A week day was taken, released, planned or applied from a plan. */
  | { type: "weekSlots"; horseId: string }
  | { type: "trainingProfile"; horseId: string }
  /** The person's own focuses on a horse ("Mein Schwerpunkt"): it changes the plan of the horse. */
  | { type: "myFocus"; horseId: string }
  /** The stable's training equipment: availability of exercises and the plans of every horse. */
  | { type: "trainingEquipment" }
  /** A practice test or freestyle was saved, edited or deleted, or a requirement set changed (the catalog lists the sets). */
  | { type: "dressageTests"; horseId: string }
  /** A reha plan or day changed. */
  | { type: "reha"; horseId: string }
  /** An observation was reported or its status changed; horseId is unknown for some calls. */
  | { type: "observation"; horseId?: string }
  /** A blanket state was set; closedRequests when it completed blanket requests. */
  | { type: "blanketState"; closedRequests: boolean }
  /** Blankets, rules, overrides or the cover window changed. */
  | { type: "blanketPlan" }
  | { type: "reminderTime" }
  /** A help request was created, edited, accepted, withdrawn, done, cancelled, thanked or reminded. */
  | { type: "request" }
  | { type: "reminderDismissed" }
  | { type: "notificationKind"; kind: string }
  | { type: "presence" }
  | { type: "consent"; kind: string }
  /** The user's own name or color changed. */
  | { type: "profile" };

export type EffectType = Effect["type"];

const TRAINING_PLAN_KIND = "training_plan";

/** The query keys an effect makes stale. */
export function effectKeys(e: Effect): Key[] {
  switch (e.type) {
    case "health":
      // The "also due" hint of other horses depends on the dates of this one: all health lists.
      return [horseKeys.healthOfAll(), reminderKeys.all];
    case "horseCreated":
      return [horseKeys.all, trainingKeys.horses(), blanketKeys.today, ME_KEY, requestKeys.options];
    case "horseDeleted":
      return [horseKeys.all, trainingKeys.horses(), blanketKeys.all, reminderKeys.all, requestKeys.all, ME_KEY, PRESENCE_KEY];
    case "horseUpdated":
      return [
        horseKeys.all,
        trainingKeys.horses(),
        trainingKeys.profile(e.horseId),
        blanketKeys.all,
        observationKeys.all,
        rehaKeys.horse(e.horseId),
        requestKeys.all,
        reminderKeys.all,
        ME_KEY,
      ];
    case "riders":
      return [horseKeys.all, trainingKeys.all, ME_KEY, reminderKeys.all];
    case "emergency":
      return [horseKeys.emergency(e.horseId)];
    case "documents":
      return [horseKeys.documents(e.horseId)];
    case "session":
      return [
        trainingKeys.weekOf(e.horseId),
        trainingKeys.todayOf(e.horseId),
        trainingKeys.horses(),
        horseKeys.diary(e.horseId),
        reminderKeys.all,
        // The profile's level progress counts rated sessions.
        trainingKeys.profile(e.horseId),
        trainingKeys.sessions(e.horseId),
      ];
    case "weekSlots":
      return [trainingKeys.weekOf(e.horseId), trainingKeys.todayOf(e.horseId), trainingKeys.horses(), reminderKeys.all];
    case "trainingProfile":
      // The phase of the profile decides which of the person's own focuses are above it.
      return [trainingKeys.weekOf(e.horseId), trainingKeys.todayOf(e.horseId), trainingKeys.horses(), trainingKeys.myFocus(e.horseId)];
    case "myFocus":
      return [trainingKeys.myFocus(e.horseId), trainingKeys.weekOf(e.horseId), trainingKeys.todayOf(e.horseId)];
    case "trainingEquipment":
      return [trainingKeys.equipment(), trainingKeys.exercises(), trainingKeys.weekOfAll(), ["training", "today"]];
    case "dressageTests":
      // The week planner links a saved test to a show day.
      return [dressageKeys.list(e.horseId), dressageKeys.detailOfAll(), dressageKeys.catalog(), trainingKeys.weekOf(e.horseId)];
    case "reha":
      return [rehaKeys.horse(e.horseId), trainingKeys.all, requestKeys.all, reminderKeys.all, observationKeys.all];
    case "observation":
      return [observationKeys.all, e.horseId ? horseKeys.diary(e.horseId) : horseKeys.diaryOfAll()];
    case "blanketState":
      return [blanketKeys.all, reminderKeys.all, ...(e.closedRequests ? [requestKeys.all] : [])];
    case "blanketPlan":
      return [blanketKeys.all];
    case "reminderTime":
      return [reminderKeys.reminderTime, blanketKeys.all, reminderKeys.all];
    case "request":
      return [requestKeys.all, reminderKeys.all];
    case "reminderDismissed":
      return [reminderKeys.all];
    case "notificationKind":
      return [reminderKeys.notifications, ...(e.kind === TRAINING_PLAN_KIND ? [reminderKeys.all] : [])];
    case "presence":
      return [PRESENCE_KEY];
    case "consent":
      return [
        CONSENTS_KEY,
        // Revoking presence sharing hides the person on the server; push changes the notification switches.
        ...(e.kind === "presence_sharing" ? [ME_KEY, PRESENCE_KEY] : []),
        ...(e.kind === "push" ? [reminderKeys.notifications] : []),
      ];
    case "profile":
      return [ME_KEY, horseKeys.members(), PRESENCE_KEY, requestKeys.all, horseKeys.all];
    default: {
      const unreachable: never = e;
      return unreachable;
    }
  }
}

/**
 * The queries that only exist for one horse and are removed from the cache when it is deleted: its record,
 * emergency card, health list, documents, diary, training data, blanket plan and history, observations,
 * reha plans and tests. Empty for every other effect.
 */
export function effectRemovedKeys(e: Effect): Key[] {
  if (e.type !== "horseDeleted") return [];
  const h = e.horseId;
  return [
    horseKeys.detail(h),
    horseKeys.emergency(h),
    horseKeys.health(h),
    horseKeys.documents(h),
    horseKeys.diary(h),
    trainingKeys.profile(h),
    trainingKeys.sessions(h),
    trainingKeys.myFocus(h),
    trainingKeys.weekOf(h),
    trainingKeys.todayOf(h),
    [...trainingKeys.exercises(), h],
    blanketKeys.plan(h),
    blanketKeys.history(h),
    observationKeys.list(h),
    rehaKeys.horse(h),
    dressageKeys.list(h),
  ];
}

/** The lists (arrays of `{id}`) that drop the horse at once, so no screen keeps rendering it while the refetch runs. */
export function effectPrunedKeys(e: Effect): Key[] {
  return e.type === "horseDeleted" ? [horseKeys.list(), trainingKeys.horses()] : [];
}

/** `list` without the horse; anything that is not a list stays as it is. */
export function withoutHorse<T>(list: T, horseId: string): T {
  if (!Array.isArray(list)) return list;
  return list.filter((item) => (item as { id?: unknown } | null)?.id !== horseId) as T;
}

/** True when `prefix` is `key` or a prefix of it (how TanStack Query matches keys). */
export function isPrefixOf(prefix: Key, key: Key): boolean {
  return prefix.length <= key.length && prefix.every((part, i) => JSON.stringify(part) === JSON.stringify(key[i]));
}

/**
 * The keys of several effects without doubles and without keys that a shorter key already covers
 * (["horses"] makes ["horses", "health", h] redundant): fewer invalidations, same result.
 */
export function uniqueKeys(keys: readonly Key[]): Key[] {
  const out: Key[] = [];
  for (const key of keys) {
    if (out.some((k) => isPrefixOf(k, key))) continue;
    for (let i = out.length - 1; i >= 0; i--) if (isPrefixOf(key, out[i] as Key)) out.splice(i, 1);
    out.push(key);
  }
  return out;
}

/** All keys that a list of effects makes stale. */
export function keysOf(effects: readonly Effect[]): Key[] {
  return uniqueKeys(effects.flatMap(effectKeys));
}

// --- realtime events ------------------------------------------------------------------------------

/** The events of the backend (docs/architecture.md, "Realtime") that make queries stale. */
export const REALTIME_EVENT_TYPES = [
  "blanket_state.changed",
  "blanket_plan.changed",
  "request.changed",
  "presence.changed",
  "observation.changed",
  "reha.changed",
  "health.changed",
  "training.changed",
  "horse.changed",
] as const;

export type RealtimeEventType = (typeof REALTIME_EVENT_TYPES)[number];

/** The synthetic event after a reconnect: events may have been missed, everything is stale. */
export const RESYNC_EVENT = "resync";

function horseOf(data: unknown): string | undefined {
  if (typeof data !== "object" || data === null) return undefined;
  const id = (data as { horse_id?: unknown }).horse_id;
  return typeof id === "string" && id !== "" ? id : undefined;
}

/**
 * The keys an event makes stale, "all" for a resync, [] for an unknown event. An event without a
 * horse id refreshes the keys of every horse.
 */
export function eventKeys(type: string, data: unknown): Key[] | "all" {
  if (type === RESYNC_EVENT) return "all";
  const horse = horseOf(data);
  switch (type as RealtimeEventType) {
    case "blanket_state.changed":
      return keysOf([{ type: "blanketState", closedRequests: false }]);
    case "blanket_plan.changed":
      return keysOf([{ type: "blanketPlan" }]);
    case "request.changed":
      return keysOf([{ type: "request" }]);
    case "presence.changed":
      return keysOf([{ type: "presence" }]);
    case "observation.changed":
      return keysOf([{ type: "observation", horseId: horse }]);
    case "reha.changed":
      return horse ? keysOf([{ type: "reha", horseId: horse }]) : uniqueKeys([rehaKeys.all, trainingKeys.all, reminderKeys.all, observationKeys.all]);
    case "health.changed":
      return keysOf([{ type: "health" }]);
    case "training.changed":
      return horse
        ? keysOf([{ type: "session", horseId: horse }])
        : uniqueKeys([trainingKeys.all, horseKeys.diaryOfAll(), reminderKeys.all]);
    case "horse.changed":
      // Our own delete: its mutation refreshes everything itself (effect horseDeleted); a prefix like ["horses"]
      // here would refetch the horse's own queries, which answer 404 now.
      if (horse && deletedHorses.has(horse)) return [];
      return horse
        ? keysOf([{ type: "horseUpdated", horseId: horse }, { type: "riders", horseId: horse }])
        : uniqueKeys([horseKeys.all, trainingKeys.all, blanketKeys.all, ME_KEY, reminderKeys.all]);
    default:
      return [];
  }
}

// --- push notifications ----------------------------------------------------------------------------

/**
 * The keys a push that arrives while the app is open makes stale, by the `kind` in its data (the
 * backend's push.Kind* constants). The reminder center shows the pushes that were sent, so every kind
 * refreshes it.
 */
export function pushKeys(kind: unknown): Key[] {
  const keys: Key[] = [reminderKeys.all];
  switch (kind) {
    case "helper":
    case "new_request":
      keys.push(requestKeys.all);
      break;
    case "observation":
    case "urgent_observation":
      keys.push(observationKeys.all, horseKeys.diaryOfAll());
      break;
    case "last_person":
    case "weather_change":
      keys.push(blanketKeys.all);
      break;
    case "weather_warning":
      keys.push(blanketKeys.all, weatherKeys.warnings);
      break;
  }
  return uniqueKeys(keys);
}
