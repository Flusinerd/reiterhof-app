import { API_URL, authed } from "../api";
import { toQuery, type HelpRequest, type ListParams } from "../requests";

// Calls of the requests API (docs/domains/requests.md). Everything is scoped to the
// stable of the signed-in user by the server.

export type RequestList = { requests: HelpRequest[]; open_count: number };

export type RequestInput = {
  type: string;
  horse_id?: string;
  date: string;
  date_end?: string;
  time_from?: string;
  time_to?: string;
  location?: string;
  description?: string;
  tasks?: string[];
  helpers_needed?: number;
  recurring_rule?: string;
  remind_helper_at?: string;
  payload?: Record<string, unknown>;
};

export type RequestPatch = Partial<Omit<RequestInput, "type">> & { scope?: "one" | "series" };

export type RequestOptions = {
  horses: { id: string; name: string; color_key: string | null }[];
  members: { id: string; name: string; avatar_color: string | null }[];
};

export const requestKeys = {
  all: ["requests"] as const,
  list: (params: ListParams) => ["requests", "list", params] as const,
  detail: (id: string) => ["requests", "detail", id] as const,
  calendar: ["requests", "calendar"] as const,
  options: ["requests", "options"] as const,
  notify: ["requests", "notify"] as const,
  thanks: ["requests", "thanks"] as const,
};

const base = "/api/v1/requests";

export const requestsApi = {
  list: (params: ListParams = {}) => authed.get<RequestList>(`${base}${toQuery(params)}`),
  get: (id: string) => authed.get<HelpRequest>(`${base}/${id}`),
  create: (input: RequestInput) => authed.post<HelpRequest>(base, input),
  update: (id: string, patch: RequestPatch) => authed.patch<HelpRequest>(`${base}/${id}`, patch),
  accept: (id: string) => authed.post<HelpRequest>(`${base}/${id}/accept`),
  withdraw: (id: string) => authed.post<HelpRequest>(`${base}/${id}/withdraw`),
  done: (id: string) => authed.post<HelpRequest>(`${base}/${id}/done`),
  cancel: (id: string, scope: "one" | "series" = "one") => authed.post<HelpRequest>(`${base}/${id}/cancel`, { scope }),
  thank: (id: string, userId: string) => authed.post<void>(`${base}/${id}/thanks/${userId}`),
  calendar: () => authed.get<{ requests: HelpRequest[] }>(`${base}/calendar`),
  options: () => authed.get<RequestOptions>(`${base}/options`),
  getNotify: () => authed.get<{ new_request: boolean }>(`${base}/notify-settings`),
  setNotify: (on: boolean) => authed.patch<{ new_request: boolean }>(`${base}/notify-settings`, { new_request: on }),
  myThanks: () => authed.get<{ count: number }>("/api/v1/me/thanks"),
};

/** Address of the ICS file of one request or of my calendar (needs the bearer token). */
export const icsUrl = (id?: string) => `${API_URL}${base}/${id ? `${id}.ics` : "calendar.ics"}`;
