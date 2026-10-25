// API calls and react-query hooks for blankets, rules, the plan and day states
// (backend: internal/blankets; docs/domains/blankets.md).

import { useQuery } from "@tanstack/react-query";

import { API_URL, authed } from "../api";
import { ApiError, parseErrorBody } from "../api-core.ts";
import type {
  Blanket,
  BlanketState,
  Override,
  Plan,
  Recommendation,
  Rule,
  StateAction,
  Today,
} from "../blankets.ts";
import { blanketKeys, mutationKeys } from "../query-keys.ts";
import { getToken } from "../token";
import { useEffectMutation } from "./mutation";

export { blanketKeys };

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
  rain_min_mm: number | null;
  rain_max_mm: number | null;
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
  /**
   * Owner or admin: pick a blanket by hand (null = "Keine Decke") from tonight on, for `nights`
   * nights (1..30) or until reset (null). Replaces an earlier choice.
   */
  setOverride: (horseId: string, blanketId: string | null, nights: number | null, note: string) =>
    put<{ override: Override; recommendation: Recommendation; rule_recommendation: Recommendation }>(
      `${horsePath(horseId)}/blanket-override`,
      { blanket_id: blanketId, nights, note },
    ),
  /** Owner or admin: back to the rules. */
  clearOverride: (horseId: string) => del(`${horsePath(horseId)}/blanket-override`),
  /** Owner or admin. Start 15:00 to 23:00, end 04:00 to 15:00, both "HH:MM". */
  saveCoverWindow: (horseId: string, coverStart: string, coverEnd: string) =>
    put<{ cover_start: string; cover_end: string }>(`${horsePath(horseId)}/cover-window`, {
      cover_start: coverStart,
      cover_end: coverEnd,
    }),
};

export const useToday = () => useQuery({ queryKey: blanketKeys.today, queryFn: blanketsApi.today });
export const usePlan = (horseId: string) =>
  useQuery({ queryKey: blanketKeys.plan(horseId), queryFn: () => blanketsApi.plan(horseId) });
export const useHistory = (horseId: string, days = 14) =>
  useQuery({ queryKey: [...blanketKeys.history(horseId), days], queryFn: () => blanketsApi.history(horseId, days) });

/**
 * Sets the day state of a horse. Refreshes the blankets, the reminder center (the blanket check) and, when
 * blanket requests were closed, the requests, and stays pending until that is done: the buttons of a row do
 * not come back before the row shows its new state.
 */
export function useSetState() {
  return useEffectMutation({
    mutationFn: (v: { horseId: string; action: StateAction; coveredWith?: string }) =>
      blanketsApi.setState(v.horseId, v.action, v.coveredWith),
    mutationKey: mutationKeys.blanketState,
    effects: (_vars, result) => [{ type: "blanketState", closedRequests: result.closed_requests.length > 0 }],
  });
}

/** Runs a plan mutation (blankets, rules) and refreshes the blanket queries afterwards. */
export function usePlanMutation<TVars, TResult>(fn: (vars: TVars) => Promise<TResult>) {
  return useEffectMutation({ mutationFn: fn, effects: () => [{ type: "blanketPlan" }] });
}
