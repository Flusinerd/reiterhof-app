// API calls and react-query hooks for horses, the emergency card, health items and documents
// (backend: internal/horses, internal/health; docs/domains/horses.md).

import { useInfiniteQuery, useQuery, type MutationKey } from "@tanstack/react-query";

import { authed } from "../api";
import { ApiError } from "../api-core.ts";
import type { Effect } from "../query-effects.ts";
import { horseKeys } from "../query-keys.ts";
import { useEffectMutation } from "./mutation";

export { horseKeys };

// --- types (mirror the backend JSON) -------------------------------------------------------

export type Person = { id: string; name: string; color_key: string | null };

export type Rider = { user_id: string; name: string; color_key: string | null; rules: string[] };

export type Horse = {
  id: string;
  name: string;
  box: string | null;
  sex: "mare" | "gelding" | "stallion" | null;
  birth_year: number | null;
  breed: string | null;
  color_key: string | null;
  weight_kg: number | null;
  helper_note: string | null;
  /** Relative API path of the photo ("/api/v1/files/..."); load it with fileSource(). null without a photo. */
  photo_url: string | null;
  owner: Person | null;
  riders: Rider[];
  is_mine: boolean;
  i_ride: boolean;
  can_manage: boolean;
  my_rules: string[];
};

/** Body of POST /horses and PATCH /horses/{id}. "" clears text fields, 0 clears numbers. */
export type HorseInput = Partial<{
  name: string;
  box: string;
  sex: string;
  birth_year: number;
  breed: string;
  color_key: string;
  weight_kg: number;
  helper_note: string;
  /** Path from uploadFile(); "" removes the photo. Owner or admin. */
  photo_path: string;
  emergency_note: string;
  emergency_medication: string;
  permanent_medication: string;
  allergies: string;
  insurance: string;
  vet_name: string;
  vet_phone: string;
  /** Admins only. */
  owner_id: string;
}>;

export type Member = { id: string; name: string; color_key: string | null; is_me: boolean };

export type Contact = { id: string; label: string; name: string; phone: string };

export type EmergencyCard = {
  horse_id: string;
  horse_name: string;
  box: string | null;
  weight_kg: number | null;
  owner: { id: string; name: string; phone: string | null } | null;
  vet_name: string | null;
  vet_phone: string | null;
  emergency_note: string | null;
  emergency_medication: string | null;
  permanent_medication: string | null;
  allergies: string | null;
  insurance: string | null;
  contacts: Contact[];
  can_manage: boolean;
};

export type HealthItem = {
  id: string;
  horse_id: string;
  kind: string;
  label: string;
  /** YYYY-MM-DD */
  due_date: string | null;
  /** Repeat interval: interval_count units (day, week, month, year); count null without one. */
  interval_count: number | null;
  interval_unit: string;
  /** The interval in days for day and week intervals, else null (older form). */
  interval_days: number | null;
  note: string | null;
  /** HH:MM */
  daily_time: string | null;
  /** Days before the due date a reminder goes out (0 = on the day), largest first; [] = none. */
  remind_days: number[];
  /** HH:MM, stable-local time of the due-date reminders. */
  remind_time: string;
  /** Negative when overdue. */
  days_until_due: number | null;
  /** The stable-local day of the last "erledigt" (YYYY-MM-DD), null without. While it is today, a second one changes nothing and it can be undone. */
  last_done_on: string | null;
  last_done_by: { id: string; name: string } | null;
  /** Other horses with the same kind of appointment due about now (JAN-54); absent if none. */
  bundle?: { kind: string; count: number; horses: { id: string; name: string }[] };
};

export type HealthResponse = {
  today: string;
  items: HealthItem[];
  summary: Record<string, HealthItem | null>;
  can_manage: boolean;
};

/** Body of POST /horses/{id}/health-items and PATCH /health-items/{id}. "" / 0 clear values. */
export type HealthInput = Partial<{
  kind: string;
  label: string;
  due_date: string;
  interval_count: number;
  interval_unit: string;
  note: string;
  daily_time: string;
  remind_days: number[];
  remind_time: string;
}>;

export type HorseDocument = {
  id: string;
  kind: string;
  title: string;
  /** Relative API path; use documentFileUrl() to get a loadable URL. */
  url: string;
  content_type: string;
  uploaded_by: Person | null;
  created_at: string;
};

/** One line of the diary (GET /horses/{id}/diary): a training session or an observation. */
export type DiaryEntry = {
  type: "session" | "observation";
  id: string;
  /** RFC 3339, UTC. */
  at: string;
  /** Calendar day of `at` in the stable's time zone, YYYY-MM-DD. */
  day: string;
  user: { id: string; name: string; color_key: string | null };
  session: {
    activity: string;
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
    urgency: "info" | "check" | "urgent";
    status: "watch" | "done";
    /** First photo (relative API path), null without photos. */
    media_url: string | null;
    media_count: number;
  } | null;
};

export type DiaryPage = { items: DiaryEntry[]; next_before: string | null };

/** A delete of something that is gone already (the first call worked, a repeat answers 404) counts as done. */
async function deleted(call: Promise<void>): Promise<void> {
  try {
    await call;
  } catch (err) {
    if (!(err instanceof ApiError && err.status === 404)) throw err;
  }
}

// --- endpoints -----------------------------------------------------------------------------

export const horsesApi = {
  list: () => authed.get<Horse[]>("/api/v1/horses"),
  get: (id: string) => authed.get<Horse>(`/api/v1/horses/${id}`),
  create: (input: HorseInput) => authed.post<Horse>("/api/v1/horses", input),
  update: (id: string, input: HorseInput) => authed.patch<Horse>(`/api/v1/horses/${id}`, input),
  members: () => authed.get<Member[]>("/api/v1/members"),
  setRider: (horseId: string, userId: string, rules: string[]) =>
    authed.put<Rider>(`/api/v1/horses/${horseId}/riders/${userId}`, { rules }),
  removeRider: (horseId: string, userId: string) =>
    deleted(authed.delete<void>(`/api/v1/horses/${horseId}/riders/${userId}`)),

  emergency: (id: string) => authed.get<EmergencyCard>(`/api/v1/horses/${id}/emergency`),
  addContact: (id: string, c: Omit<Contact, "id">) =>
    authed.post<Contact>(`/api/v1/horses/${id}/emergency-contacts`, c),
  updateContact: (id: string, contactId: string, c: Partial<Omit<Contact, "id">>) =>
    authed.patch<Contact>(`/api/v1/horses/${id}/emergency-contacts/${contactId}`, c),
  removeContact: (id: string, contactId: string) =>
    deleted(authed.delete<void>(`/api/v1/horses/${id}/emergency-contacts/${contactId}`)),

  health: (id: string) => authed.get<HealthResponse>(`/api/v1/horses/${id}/health`),
  addHealthItem: (id: string, input: HealthInput) =>
    authed.post<HealthItem>(`/api/v1/horses/${id}/health-items`, input),
  updateHealthItem: (itemId: string, input: HealthInput) =>
    authed.patch<HealthItem>(`/api/v1/health-items/${itemId}`, input),
  removeHealthItem: (itemId: string) => deleted(authed.delete<void>(`/api/v1/health-items/${itemId}`)),
  /**
   * "Erledigt". Once per day on the server: a repeat answers with the unchanged item. `seenDueDate` is the date the
   * app showed ("" = none); 409 `due_date_changed` when somebody else moved it meanwhile. For a medication it ends the course.
   */
  markDone: (itemId: string, seenDueDate: string | null) =>
    authed.post<HealthItem>(`/api/v1/health-items/${itemId}/done`, { due_date: seenDueDate ?? "" }),
  /** Takes today's "erledigt" back (the date and the reminders return); 409 `undo_expired` for an earlier day. */
  undoDone: (itemId: string) => authed.delete<HealthItem>(`/api/v1/health-items/${itemId}/done`),

  diary: (id: string, before?: string, limit = 20) =>
    authed.get<DiaryPage>(
      `/api/v1/horses/${id}/diary?limit=${limit}${before ? `&before=${encodeURIComponent(before)}` : ""}`,
    ),

  documents: (id: string) => authed.get<HorseDocument[]>(`/api/v1/horses/${id}/documents`),
  addDocument: (id: string, input: { kind: string; title: string; file_path: string }) =>
    authed.post<HorseDocument>(`/api/v1/horses/${id}/documents`, input),
  updateDocument: (id: string, docId: string, input: { kind?: string; title?: string }) =>
    authed.patch<HorseDocument>(`/api/v1/horses/${id}/documents/${docId}`, input),
  removeDocument: (id: string, docId: string) =>
    deleted(authed.delete<void>(`/api/v1/horses/${id}/documents/${docId}`)),
};

// --- query keys and hooks ------------------------------------------------------------------

export const useHorses = () => useQuery({ queryKey: horseKeys.list(), queryFn: horsesApi.list });
export const useHorse = (id: string) =>
  useQuery({ queryKey: horseKeys.detail(id), queryFn: () => horsesApi.get(id) });
export const useEmergency = (id: string) =>
  useQuery({ queryKey: horseKeys.emergency(id), queryFn: () => horsesApi.emergency(id) });
export const useHealth = (id: string) =>
  useQuery({ queryKey: horseKeys.health(id), queryFn: () => horsesApi.health(id) });
export const useDocuments = (id: string, enabled = true) =>
  useQuery({ queryKey: horseKeys.documents(id), queryFn: () => horsesApi.documents(id), enabled });
/** The diary, newest first, a page at a time (`fetchNextPage` loads older entries). */
export const useDiary = (id: string, enabled = true) =>
  useInfiniteQuery({
    queryKey: horseKeys.diary(id),
    queryFn: ({ pageParam }) => horsesApi.diary(id, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next_before ?? undefined,
    enabled,
  });
export const useMembers = () =>
  useQuery({ queryKey: horseKeys.members(), queryFn: horsesApi.members, staleTime: 60_000 });

/**
 * Runs a write about a horse and refreshes what it changes (lib/query-effects.ts) before the mutation
 * ends. `effects` says what the call changed, e.g. `() => [{ type: "health" }]`; `mutationKey` lets other
 * components ask whether this write is running (usePendingIds).
 */
export function useHorseMutation<TVars, TResult>(
  fn: (vars: TVars) => Promise<TResult>,
  effects: (vars: TVars, result: TResult) => Effect[],
  options: {
    mutationKey?: MutationKey;
    /** What a failed call may still have changed (a 409 because somebody else changed the item first). */
    errorEffects?: (vars: TVars) => Effect[];
    /** Runs when the call succeeded, before the refresh: write the answer somewhere the screen shows at once. */
    onSuccess?: (result: TResult, vars: TVars) => void;
  } = {},
) {
  return useEffectMutation({
    mutationFn: fn,
    mutationKey: options.mutationKey,
    effects,
    errorEffects: options.errorEffects,
    onSuccess: options.onSuccess,
  });
}
