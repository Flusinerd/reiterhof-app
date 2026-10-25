// The TanStack Query keys of the app in one place. Pure, no imports: the effect table
// (query-effects.ts) and its tests use them too. The modules that used to define a factory
// re-export it, so call sites keep their imports.
//
// Keys are hierarchical on purpose: invalidating a prefix refetches everything below it
// (["horses"] = lists, records, cards, health, diary; ["training", "week", h] = every week of h).

/** The signed-in user with stable and roles (GET /me). */
export const ME_KEY = ["me"] as const;

/** Who is at the stable (GET /presence). */
export const PRESENCE_KEY = ["presence"] as const;

/** The privacy consents of the user (GET /me/consents). */
export const CONSENTS_KEY = ["consents"] as const;

export const horseKeys = {
  all: ["horses"] as const,
  list: () => ["horses", "list"] as const,
  detail: (id: string) => ["horses", "detail", id] as const,
  emergency: (id: string) => ["horses", "emergency", id] as const,
  health: (id: string) => ["horses", "health", id] as const,
  /** The health lists of all horses: the "also due" hint of one horse depends on the dates of the others. */
  healthOfAll: () => ["horses", "health"] as const,
  documents: (id: string) => ["horses", "documents", id] as const,
  diary: (id: string) => ["horses", "diary", id] as const,
  /** The diaries of all horses. */
  diaryOfAll: () => ["horses", "diary"] as const,
  members: () => ["members"] as const,
};

export const reminderKeys = {
  all: ["reminders"] as const,
  list: (range: "today" | "week") => ["reminders", "list", range] as const,
  notifications: ["settings", "notifications"] as const,
  reminderTime: ["settings", "reminder-time"] as const,
};

export const blanketKeys = {
  all: ["blankets"] as const,
  today: ["blankets", "today"] as const,
  plan: (horseId: string) => ["blankets", "plan", horseId] as const,
  history: (horseId: string) => ["blankets", "history", horseId] as const,
};

export const trainingKeys = {
  all: ["training"] as const,
  horses: () => ["training", "horses"] as const,
  today: (horse: string, minutes?: number) => ["training", "today", horse, minutes ?? 0] as const,
  /** Every "today" query of a horse (any minutes). */
  todayOf: (horse: string) => ["training", "today", horse] as const,
  week: (horse: string, start?: string) => ["training", "week", horse, start ?? "current"] as const,
  /** Every week query of a horse (the current one and the ones by start date). */
  weekOf: (horse: string) => ["training", "week", horse] as const,
  /** The week queries of all horses. */
  weekOfAll: () => ["training", "week"] as const,
  profile: (horse: string) => ["training", "profile", horse] as const,
  sessions: (horse: string) => ["training", "sessions", horse] as const,
  focuses: () => ["training", "focuses"] as const,
  myFocus: (horse: string) => ["training", "my-focus", horse] as const,
  equipment: () => ["training", "equipment"] as const,
  /** Every exercise list (any horse, any focus); the lists themselves are keyed in lib/tracking-queries.ts. */
  exercises: () => ["training", "exercises"] as const,
  exercise: (id: string | undefined) => ["training", "exercise", id] as const,
};

export const dressageKeys = {
  all: ["dressage"] as const,
  catalog: () => ["dressage", "catalog"] as const,
  /** The saved tests of a horse. */
  list: (horse: string) => ["dressage", "tests", horse] as const,
  /** One saved test (the id alone says nothing about the horse). */
  detail: (id: string) => ["dressage", "test", id] as const,
  /** Every saved test. */
  detailOfAll: () => ["dressage", "test"] as const,
};

export const requestKeys = {
  all: ["requests"] as const,
  list: <P>(params: P) => ["requests", "list", params] as const,
  detail: (id: string) => ["requests", "detail", id] as const,
  calendar: ["requests", "calendar"] as const,
  options: ["requests", "options"] as const,
  notify: ["requests", "notify"] as const,
  thanks: ["requests", "thanks"] as const,
};

export const observationKeys = {
  all: ["observations"] as const,
  list: (horseId: string) => ["observations", "list", horseId] as const,
  detail: (id: string) => ["observations", "detail", id] as const,
};

export const rehaKeys = {
  all: ["reha"] as const,
  horse: (horse: string) => ["reha", horse] as const,
};

export const weatherKeys = {
  warnings: ["weather", "warnings"] as const,
};

/**
 * Keys of the mutations that show a pending state per item. The same key on every hook that
 * can trigger the same write lets any screen ask "is this item being written right now?"
 * (usePendingIds, lib/use-pending.ts).
 */
export const mutationKeys = {
  healthDone: ["health", "done"] as const,
  healthUndo: ["health", "undo"] as const,
  healthSave: ["health", "save"] as const,
  healthRemove: ["health", "remove"] as const,
  blanketState: ["blankets", "state"] as const,
  requestAccept: ["requests", "accept"] as const,
  requestThank: ["requests", "thank"] as const,
  reminderDismiss: ["reminders", "dismiss"] as const,
  documentRemove: ["documents", "remove"] as const,
  contactRemove: ["contacts", "remove"] as const,
};
