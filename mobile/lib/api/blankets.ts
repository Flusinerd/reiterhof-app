// API calls and react-query hooks for blankets, rules, the plan and day states
// (backend: internal/blankets; docs/domains/blankets.md).

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { API_URL, authed } from "../api";
import { ApiError, parseErrorBody } from "../api-core.ts";
import type { Blanket, BlanketState, Plan, Rule, StateAction, Today } from "../blankets.ts";
import { getToken } from "../token";

export const BLANKET_STATE_EVENT = "blanket_state.changed";
export const BLANKET_PLAN_EVENT = "blanket_plan.changed";

/** Body of POST and PATCH of a blanket. "" clears color, location and photo. */
export type BlanketInput = Partial<{
  name: string;
  fill_g: number;
  color: string;
  location: string;
  /** Path from uploadFile(). */
  photo_path: string;
}>;

/** One rule as sent to PUT /blanket-rules; the array order is the priority. */
export type RuleInput = {
  temp_min: number | null;
  temp_max: number | null;
  rain: boolean | null;
  blanket_id: string | null;
  note: string;
};

export type StateResult = { state: BlanketState; closed_requests: string[] };

export type History = { today: string; states: BlanketState[] };

async function put<T>(path: string, body: unknown): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json", "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(`${API_URL.replace(/\/+$/, "")}${path}`, { method: "PUT", headers, body: JSON.stringify(body) });
  } catch {
    throw new ApiError(0, "network", "Network request failed");
  }
  const text = await res.text();
  if (!res.ok) throw parseErrorBody(res.status, text);
  return JSON.parse(text) as T;
}

async function del(path: string): Promise<void> {
  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(`${API_URL.replace(/\/+$/, "")}${path}`, { method: "DELETE", headers });
  } catch {
    throw new ApiError(0, "network", "Network request failed");
  }
  if (!res.ok) throw parseErrorBody(res.status, await res.text());
}

const horsePath = (horseId: string) => `/api/v1/horses/${horseId}`;

export const blanketsApi = {
  today: () => authed.get<Today>("/api/v1/blankets/today"),
  plan: (horseId: string) => authed.get<Plan>(`${horsePath(horseId)}/blanket-plan`),
  history: (horseId: string, days = 14) =>
    authed.get<History>(`${horsePath(horseId)}/blanket-states?days=${days}`),
  setState: (horseId: string, action: StateAction, coveredWith?: string) =>
    authed.post<StateResult>(`${horsePath(horseId)}/blanket-state`, {
      action,
      ...(coveredWith ? { covered_with: coveredWith } : {}),
    }),

  createBlanket: (horseId: string, input: BlanketInput) =>
    authed.post<Blanket>(`${horsePath(horseId)}/blankets`, input),
  updateBlanket: (horseId: string, id: string, input: BlanketInput) =>
    authed.patch<Blanket>(`${horsePath(horseId)}/blankets/${id}`, input),
  removeBlanket: (horseId: string, id: string) => del(`${horsePath(horseId)}/blankets/${id}`),
  saveRules: (horseId: string, rules: RuleInput[]) =>
    put<{ rules: Rule[] }>(`${horsePath(horseId)}/blanket-rules`, { rules }),
  /** Owner or admin. Start 15:00 to 23:00, end 04:00 to 15:00, both "HH:MM". */
  saveCoverWindow: (horseId: string, coverStart: string, coverEnd: string) =>
    put<{ cover_start: string; cover_end: string }>(`${horsePath(horseId)}/cover-window`, {
      cover_start: coverStart,
      cover_end: coverEnd,
    }),
};

export const blanketKeys = {
  all: ["blankets"] as const,
  today: ["blankets", "today"] as const,
  plan: (horseId: string) => ["blankets", "plan", horseId] as const,
  history: (horseId: string) => ["blankets", "history", horseId] as const,
};

export const useToday = () => useQuery({ queryKey: blanketKeys.today, queryFn: blanketsApi.today });
export const usePlan = (horseId: string) =>
  useQuery({ queryKey: blanketKeys.plan(horseId), queryFn: () => blanketsApi.plan(horseId) });
export const useHistory = (horseId: string, days = 14) =>
  useQuery({ queryKey: [...blanketKeys.history(horseId), days], queryFn: () => blanketsApi.history(horseId, days) });

/** Sets the day state of a horse and refreshes everything about blankets (and requests, which may close). */
export function useSetState() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (v: { horseId: string; action: StateAction; coveredWith?: string }) =>
      blanketsApi.setState(v.horseId, v.action, v.coveredWith),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: blanketKeys.all });
      if (result.closed_requests.length > 0) void queryClient.invalidateQueries({ queryKey: ["requests"] });
    },
  });
}

/** Runs a plan mutation (blankets, rules) and refreshes the blanket queries afterwards. */
export function usePlanMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: fn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: blanketKeys.all }),
  });
}
