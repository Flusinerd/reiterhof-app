// API calls and react-query hooks for horses, the emergency card, health items and documents
// (backend: internal/horses, internal/health; docs/domains/horses.md).

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { API_URL, authed } from "../api";
import { parseErrorBody, ApiError } from "../api-core.ts";
import { getToken } from "../token";

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
  interval_days: number | null;
  note: string | null;
  /** HH:MM */
  daily_time: string | null;
  /** Negative when overdue. */
  days_until_due: number | null;
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
  interval_days: number;
  note: string;
  daily_time: string;
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

// --- transport for PUT / DELETE (the shared client offers GET, POST and PATCH) -------------

async function send<T>(method: "PUT" | "DELETE", path: string, body?: unknown): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(`${API_URL.replace(/\/+$/, "")}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(0, "network", "Network request failed");
  }
  if (res.status === 204) return undefined as T;
  const text = await res.text();
  if (!res.ok) throw parseErrorBody(res.status, text);
  return (text ? JSON.parse(text) : undefined) as T;
}

// --- endpoints -----------------------------------------------------------------------------

export const horsesApi = {
  list: () => authed.get<Horse[]>("/api/v1/horses"),
  get: (id: string) => authed.get<Horse>(`/api/v1/horses/${id}`),
  create: (input: HorseInput) => authed.post<Horse>("/api/v1/horses", input),
  update: (id: string, input: HorseInput) => authed.patch<Horse>(`/api/v1/horses/${id}`, input),
  members: () => authed.get<Member[]>("/api/v1/members"),
  setRider: (horseId: string, userId: string, rules: string[]) =>
    send<Rider>("PUT", `/api/v1/horses/${horseId}/riders/${userId}`, { rules }),
  removeRider: (horseId: string, userId: string) =>
    send<void>("DELETE", `/api/v1/horses/${horseId}/riders/${userId}`),

  emergency: (id: string) => authed.get<EmergencyCard>(`/api/v1/horses/${id}/emergency`),
  addContact: (id: string, c: Omit<Contact, "id">) =>
    authed.post<Contact>(`/api/v1/horses/${id}/emergency-contacts`, c),
  updateContact: (id: string, contactId: string, c: Partial<Omit<Contact, "id">>) =>
    authed.patch<Contact>(`/api/v1/horses/${id}/emergency-contacts/${contactId}`, c),
  removeContact: (id: string, contactId: string) =>
    send<void>("DELETE", `/api/v1/horses/${id}/emergency-contacts/${contactId}`),

  health: (id: string) => authed.get<HealthResponse>(`/api/v1/horses/${id}/health`),
  addHealthItem: (id: string, input: HealthInput) =>
    authed.post<HealthItem>(`/api/v1/horses/${id}/health-items`, input),
  updateHealthItem: (itemId: string, input: HealthInput) =>
    authed.patch<HealthItem>(`/api/v1/health-items/${itemId}`, input),
  removeHealthItem: (itemId: string) => send<void>("DELETE", `/api/v1/health-items/${itemId}`),
  markDone: (itemId: string) => authed.post<HealthItem>(`/api/v1/health-items/${itemId}/done`),

  documents: (id: string) => authed.get<HorseDocument[]>(`/api/v1/horses/${id}/documents`),
  addDocument: (id: string, input: { kind: string; title: string; file_path: string }) =>
    authed.post<HorseDocument>(`/api/v1/horses/${id}/documents`, input),
  updateDocument: (id: string, docId: string, input: { kind?: string; title?: string }) =>
    authed.patch<HorseDocument>(`/api/v1/horses/${id}/documents/${docId}`, input),
  removeDocument: (id: string, docId: string) =>
    send<void>("DELETE", `/api/v1/horses/${id}/documents/${docId}`),
};

// --- query keys and hooks ------------------------------------------------------------------

export const horseKeys = {
  all: ["horses"] as const,
  list: () => ["horses", "list"] as const,
  detail: (id: string) => ["horses", "detail", id] as const,
  emergency: (id: string) => ["horses", "emergency", id] as const,
  health: (id: string) => ["horses", "health", id] as const,
  documents: (id: string) => ["horses", "documents", id] as const,
  members: () => ["members"] as const,
};

export const useHorses = () => useQuery({ queryKey: horseKeys.list(), queryFn: horsesApi.list });
export const useHorse = (id: string) =>
  useQuery({ queryKey: horseKeys.detail(id), queryFn: () => horsesApi.get(id) });
export const useEmergency = (id: string) =>
  useQuery({ queryKey: horseKeys.emergency(id), queryFn: () => horsesApi.emergency(id) });
export const useHealth = (id: string) =>
  useQuery({ queryKey: horseKeys.health(id), queryFn: () => horsesApi.health(id) });
export const useDocuments = (id: string, enabled = true) =>
  useQuery({ queryKey: horseKeys.documents(id), queryFn: () => horsesApi.documents(id), enabled });
export const useMembers = () =>
  useQuery({ queryKey: horseKeys.members(), queryFn: horsesApi.members, staleTime: 60_000 });

/**
 * Runs a mutation and refreshes everything about horses afterwards (lists, record, card,
 * health, documents). Horse data is small; one broad invalidation keeps the screens consistent.
 */
export function useHorseMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: horseKeys.all }),
  });
}
