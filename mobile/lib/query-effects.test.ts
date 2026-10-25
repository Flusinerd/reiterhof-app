import assert from "node:assert/strict";
import { test } from "node:test";

import {
  effectKeys,
  eventKeys,
  isPrefixOf,
  keysOf,
  pushKeys,
  REALTIME_EVENT_TYPES,
  RESYNC_EVENT,
  uniqueKeys,
  type Effect,
  type EffectType,
  type Key,
} from "./query-effects.ts";
import { blanketKeys, horseKeys, observationKeys, reminderKeys, trainingKeys, weatherKeys } from "./query-keys.ts";

const h = "h1";

/** One sample of every effect type; typed as a record so a new type must be added here. */
const samples: Record<EffectType, Effect> = {
  health: { type: "health" },
  horseCreated: { type: "horseCreated" },
  horseUpdated: { type: "horseUpdated", horseId: h },
  riders: { type: "riders", horseId: h },
  emergency: { type: "emergency", horseId: h },
  documents: { type: "documents", horseId: h },
  session: { type: "session", horseId: h },
  weekSlots: { type: "weekSlots", horseId: h },
  trainingProfile: { type: "trainingProfile", horseId: h },
  myFocus: { type: "myFocus", horseId: h },
  trainingEquipment: { type: "trainingEquipment" },
  reha: { type: "reha", horseId: h },
  observation: { type: "observation", horseId: h },
  blanketState: { type: "blanketState", closedRequests: false },
  blanketPlan: { type: "blanketPlan" },
  reminderTime: { type: "reminderTime" },
  request: { type: "request" },
  reminderDismissed: { type: "reminderDismissed" },
  notificationKind: { type: "notificationKind", kind: "last_person" },
  presence: { type: "presence" },
  consent: { type: "consent", kind: "location_geofence" },
  profile: { type: "profile" },
};

const json = (keys: readonly Key[]) => keys.map((k) => JSON.stringify(k));

test("the keys of every effect (the table of docs/design-system.md, Feedback)", () => {
  const expected: Record<EffectType, string[]> = {
    health: ['["horses","health"]', '["reminders"]'],
    horseCreated: ['["horses"]', '["training","horses"]', '["blankets","today"]', '["me"]', '["requests","options"]'],
    horseUpdated: [
      '["horses"]',
      '["training","horses"]',
      '["training","profile","h1"]',
      '["blankets"]',
      '["observations"]',
      '["reha","h1"]',
      '["requests"]',
      '["reminders"]',
      '["me"]',
    ],
    riders: ['["horses"]', '["training"]', '["me"]', '["reminders"]'],
    emergency: ['["horses","emergency","h1"]'],
    documents: ['["horses","documents","h1"]'],
    session: [
      '["training","week","h1"]',
      '["training","today","h1"]',
      '["training","horses"]',
      '["horses","diary","h1"]',
      '["reminders"]',
      '["training","profile","h1"]',
      '["training","sessions","h1"]',
    ],
    weekSlots: ['["training","week","h1"]', '["training","today","h1"]', '["training","horses"]', '["reminders"]'],
    trainingProfile: ['["training","week","h1"]', '["training","today","h1"]', '["training","horses"]', '["training","my-focus","h1"]'],
    myFocus: ['["training","my-focus","h1"]', '["training","week","h1"]', '["training","today","h1"]'],
    trainingEquipment: ['["training","equipment"]', '["training","exercises"]', '["training","week"]', '["training","today"]'],
    reha: ['["reha","h1"]', '["training"]', '["requests"]', '["reminders"]', '["observations"]'],
    observation: ['["observations"]', '["horses","diary","h1"]'],
    blanketState: ['["blankets"]', '["reminders"]'],
    blanketPlan: ['["blankets"]'],
    reminderTime: ['["settings","reminder-time"]', '["blankets"]', '["reminders"]'],
    request: ['["requests"]', '["reminders"]'],
    reminderDismissed: ['["reminders"]'],
    notificationKind: ['["settings","notifications"]'],
    presence: ['["presence"]'],
    consent: ['["consents"]'],
    profile: ['["me"]', '["members"]', '["presence"]', '["requests"]', '["horses"]'],
  };
  for (const [type, effect] of Object.entries(samples)) {
    assert.deepEqual(json(effectKeys(effect)), expected[type as EffectType], type);
  }
});

test("effects that depend on a detail of the change", () => {
  assert.deepEqual(json(effectKeys({ type: "blanketState", closedRequests: true })), ['["blankets"]', '["reminders"]', '["requests"]']);
  // The training plan switch also changes the reminder center, the other kinds do not.
  assert.deepEqual(json(effectKeys({ type: "notificationKind", kind: "training_plan" })), ['["settings","notifications"]', '["reminders"]']);
  assert.deepEqual(json(effectKeys({ type: "consent", kind: "presence_sharing" })), ['["consents"]', '["me"]', '["presence"]']);
  assert.deepEqual(json(effectKeys({ type: "consent", kind: "push" })), ['["consents"]', '["settings","notifications"]']);
  assert.deepEqual(json(effectKeys({ type: "observation" })), ['["observations"]', '["horses","diary"]']);
});

// The inputs of "Heute" (lib/use-today.ts): the training horses, one week per horse (two on a
// Sunday), tonight's blankets and the reminder center.
const heute = {
  horses: trainingKeys.horses(),
  weeks: [trainingKeys.week("h1"), trainingKeys.week("h1", "2026-10-26"), trainingKeys.week("h2")],
  blankets: blanketKeys.today,
  reminders: reminderKeys.list("week"),
};

const reaches = (keys: readonly Key[], input: Key) => keys.some((k) => isPrefixOf(k, input));

test("every input of Heute is reached by the effects that change it", () => {
  const changes: { input: string; keys: Key[]; effects: Effect[] }[] = [
    {
      input: "reminders",
      keys: [heute.reminders],
      effects: [
        { type: "health" },
        { type: "reha", horseId: h },
        { type: "request" },
        { type: "blanketState", closedRequests: false },
        { type: "session", horseId: h },
        { type: "weekSlots", horseId: h },
        { type: "reminderTime" },
        { type: "reminderDismissed" },
        { type: "notificationKind", kind: "training_plan" },
        { type: "horseUpdated", horseId: h },
        { type: "riders", horseId: h },
      ],
    },
    {
      input: "training horses",
      keys: [heute.horses],
      effects: [
        { type: "session", horseId: h },
        { type: "weekSlots", horseId: h },
        { type: "trainingProfile", horseId: h },
        { type: "horseCreated" },
        { type: "horseUpdated", horseId: h },
        { type: "riders", horseId: h },
        { type: "reha", horseId: h },
      ],
    },
    {
      input: "the weeks of h1",
      keys: [heute.weeks[0] as Key, heute.weeks[1] as Key],
      effects: [
        { type: "session", horseId: "h1" },
        { type: "weekSlots", horseId: "h1" },
        { type: "trainingProfile", horseId: "h1" },
        { type: "reha", horseId: "h1" },
        { type: "riders", horseId: "h1" },
      ],
    },
    {
      input: "tonight's blankets",
      keys: [heute.blankets],
      effects: [
        { type: "blanketState", closedRequests: false },
        { type: "blanketPlan" },
        { type: "reminderTime" },
        { type: "horseCreated" },
        { type: "horseUpdated", horseId: h },
      ],
    },
  ];
  for (const { input, keys, effects } of changes) {
    for (const effect of effects) {
      for (const key of keys) {
        assert.ok(reaches(effectKeys(effect), key), `${effect.type} does not refresh ${input} (${JSON.stringify(key)})`);
      }
    }
  }
});

test("every input of Heute is reached by the realtime events that change it", () => {
  const changes: { input: string; keys: Key[]; events: string[] }[] = [
    {
      input: "reminders",
      keys: [heute.reminders],
      events: [
        "health.changed",
        "reha.changed",
        "request.changed",
        "blanket_state.changed",
        "training.changed",
        "horse.changed",
      ],
    },
    { input: "training horses", keys: [heute.horses], events: ["training.changed", "horse.changed", "reha.changed"] },
    {
      input: "the weeks of h1",
      keys: [heute.weeks[0] as Key, heute.weeks[1] as Key],
      events: ["training.changed", "reha.changed"],
    },
    {
      input: "tonight's blankets",
      keys: [heute.blankets],
      events: ["blanket_state.changed", "blanket_plan.changed", "horse.changed"],
    },
  ];
  for (const { input, keys, events } of changes) {
    for (const type of events) {
      const got = eventKeys(type, { horse_id: "h1" });
      assert.ok(Array.isArray(got), type);
      for (const key of keys) assert.ok(reaches(got as Key[], key), `${type} does not refresh ${input} (${JSON.stringify(key)})`);
    }
  }
});

test("a change of one horse does not refresh the weeks of another", () => {
  const other = heute.weeks[2] as Key; // h2
  assert.ok(!reaches(effectKeys({ type: "session", horseId: "h1" }), other));
  assert.ok(!reaches(effectKeys({ type: "weekSlots", horseId: "h1" }), other));
  assert.ok(!reaches(eventKeys("training.changed", { horse_id: "h1" }) as Key[], other));
  // Without a horse id the event cannot be scoped: everything of that kind is stale.
  assert.ok(reaches(eventKeys("training.changed", {}) as Key[], other));
  assert.ok(reaches(eventKeys("training.changed", null) as Key[], other));
});

test("events", () => {
  assert.deepEqual(json(eventKeys("blanket_state.changed", { horse_id: h, day: "2026-10-25" }) as Key[]), ['["blankets"]', '["reminders"]']);
  assert.deepEqual(json(eventKeys("blanket_plan.changed", { horse_id: h }) as Key[]), ['["blankets"]']);
  assert.deepEqual(json(eventKeys("request.changed", { id: "r1", kind: "done" }) as Key[]), ['["requests"]', '["reminders"]']);
  assert.deepEqual(json(eventKeys("presence.changed", null) as Key[]), ['["presence"]']);
  assert.deepEqual(json(eventKeys("observation.changed", { id: "o1", horse_id: h }) as Key[]), ['["observations"]', '["horses","diary","h1"]']);
  assert.deepEqual(json(eventKeys("reha.changed", { horse_id: h }) as Key[]), ['["reha","h1"]', '["training"]', '["requests"]', '["reminders"]', '["observations"]']);
  assert.deepEqual(json(eventKeys("health.changed", { horse_id: h }) as Key[]), ['["horses","health"]', '["reminders"]']);
  assert.deepEqual(json(eventKeys("training.changed", { horse_id: h }) as Key[]), [
    '["training","week","h1"]',
    '["training","today","h1"]',
    '["training","horses"]',
    '["horses","diary","h1"]',
    '["reminders"]',
    '["training","profile","h1"]',
    '["training","sessions","h1"]',
  ]);
  // A horse change covers its record, the rights of its riders and what is derived from names.
  const horse = eventKeys("horse.changed", { horse_id: h }) as Key[];
  for (const key of [horseKeys.detail(h), trainingKeys.horses(), trainingKeys.profile(h), blanketKeys.today, ["me"], reminderKeys.all, observationKeys.list(h)]) {
    assert.ok(reaches(horse, key), JSON.stringify(key));
  }
});

test("resync refreshes everything, unknown events nothing", () => {
  assert.equal(eventKeys(RESYNC_EVENT, null), "all");
  assert.deepEqual(eventKeys("something.else", { horse_id: h }), []);
});

test("every event of the backend is mapped to keys", () => {
  assert.deepEqual([...REALTIME_EVENT_TYPES].sort(), [
    "blanket_plan.changed",
    "blanket_state.changed",
    "health.changed",
    "horse.changed",
    "observation.changed",
    "presence.changed",
    "reha.changed",
    "request.changed",
    "training.changed",
  ]);
  for (const type of REALTIME_EVENT_TYPES) {
    const keys = eventKeys(type, { horse_id: h });
    assert.ok(Array.isArray(keys) && keys.length > 0, type);
  }
});

test("uniqueKeys drops doubles and keys that a shorter key covers", () => {
  assert.deepEqual(json(uniqueKeys([["a", "b"], ["a"], ["a", "b", "c"], ["x"], ["x"]])), ['["a"]', '["x"]']);
  assert.deepEqual(json(uniqueKeys([["a", "b", "c"], ["a", "b"]])), ['["a","b"]']);
  assert.deepEqual(json(keysOf([{ type: "health" }, { type: "horseUpdated", horseId: h }])), json(
    uniqueKeys([...effectKeys({ type: "health" }), ...effectKeys({ type: "horseUpdated", horseId: h })]),
  ));
  // ["horses"] already covers ["horses", "health"].
  assert.ok(!json(keysOf([{ type: "health" }, { type: "horseUpdated", horseId: h }])).includes('["horses","health"]'));
});

test("isPrefixOf follows how TanStack Query matches keys", () => {
  assert.ok(isPrefixOf(["training"], ["training", "week", "h1", "current"]));
  assert.ok(isPrefixOf(["training", "week", "h1"], ["training", "week", "h1", "2026-10-26"]));
  assert.ok(!isPrefixOf(["training", "week", "h1"], ["training", "week", "h2", "current"]));
  assert.ok(!isPrefixOf(["training", "week", "h1", "current"], ["training", "week", "h1"]));
});

test("a push that arrives while the app is open refreshes the reminder center and what its kind is about", () => {
  assert.deepEqual(json(pushKeys("health_due")), ['["reminders"]']);
  assert.deepEqual(json(pushKeys("medication")), ['["reminders"]']);
  assert.deepEqual(json(pushKeys("helper")), ['["reminders"]', '["requests"]']);
  assert.deepEqual(json(pushKeys("new_request")), ['["reminders"]', '["requests"]']);
  assert.deepEqual(json(pushKeys("urgent_observation")), ['["reminders"]', '["observations"]', '["horses","diary"]']);
  assert.deepEqual(json(pushKeys("weather_warning")), ['["reminders"]', '["blankets"]', '["weather","warnings"]']);
  assert.deepEqual(json(pushKeys(undefined)), ['["reminders"]']);
  assert.ok(reaches(pushKeys("weather_warning"), weatherKeys.warnings));
});
