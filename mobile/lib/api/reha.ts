// Reha plan endpoints (docs/domains/reha.md) with React Query keys and hooks.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { API_URL, ApiError, authed, errorMessage } from "../api";
import { parseErrorBody } from "../api-core.ts";
import type { PlanInput, PlanState } from "../reha.ts";
import { rehaErrorText } from "../reha.ts";
import { getToken } from "../token";

// --- types (mirror backend/internal/reha) --------------------------------------------------------

export type RehaPhase = {
  name: string;
  days: number;
  activity: string;
  activity_label: string;
  rest: boolean;
  min_minutes: number;
  max_minutes: number;
  conditions: string;
  start_date: string;
  end_date: string;
  status: "past" | "current" | "upcoming";
};

export type RehaPlan = {
  id: string;
  diagnosis: string;
  vet: string;
  start_date: string;
  end_date: string;
  checkup_date: string | null;
  /** Negative when overdue. */
  checkup_in_days: number | null;
  abort_criteria: string;
  observation_id: string | null;
  active: boolean;
  ended_on: string | null;
  state: PlanState;
  /** 1-based, 0 = no phase today. */
  current_phase: number;
  /** 1-based day of the plan today, 0 = not running. */
  day_index: number;
  total_days: number;
  phases: RehaPhase[];
  done_days: string[];
};

/** "Heute erlaubt". */
export type RehaToday = {
  date: string;
  plan_id: string;
  phase: string;
  phase_index: number;
  phases: number;
  day_in_phase: number;
  days_in_phase: number;
  activity: string;
  activity_label: string;
  rest: boolean;
  minutes: number;
  min_minutes: number;
  max_minutes: number;
  conditions: string;
  text: string;
  done: boolean;
  done_by?: string;
};

export type RehaView = {
  horse_id: string;
  horse_name: string;
  /** "full" for owner, admins and riders; "today" for other members (only today's rule). */
  view: "full" | "today";
  date: string;
  can_edit: boolean;
  can_mark_done: boolean;
  has_active_plan: boolean;
  plan: RehaPlan | null;
  today: RehaToday | null;
  history: RehaPlan[];
};

// --- transport for DELETE (the shared client offers GET, POST, PATCH and PUT) ------------------

async function del<T>(path: string): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(`${API_URL.replace(/\/+$/, "")}${path}`, { method: "DELETE", headers });
  } catch {
    throw new ApiError(0, "network", "Network request failed");
  }
  const text = await res.text();
  if (!res.ok) throw parseErrorBody(res.status, text);
  return (text ? JSON.parse(text) : undefined) as T;
}

// --- calls -------------------------------------------------------------------------------------

const horsePath = (horse: string) => `/api/v1/horses/${encodeURIComponent(horse)}`;
const planPath = (plan: string) => `/api/v1/reha-plans/${encodeURIComponent(plan)}`;

export const rehaApi = {
  get: (horse: string) => authed.get<RehaView>(`${horsePath(horse)}/reha`),
  /** `observation_id` links the plan to an observation (JAN-68). */
  create: (horse: string, body: PlanInput & { observation_id?: string }) =>
    authed.post<RehaView>(`${horsePath(horse)}/reha-plans`, body),
  update: (plan: string, body: Partial<PlanInput>) => authed.patch<RehaView>(planPath(plan), body),
  end: (plan: string) => authed.post<RehaView>(`${planPath(plan)}/end`),
  markDone: (plan: string, day: string) => authed.post<RehaView>(`${planPath(plan)}/days/${day}/done`),
  unmarkDone: (plan: string, day: string) => del<RehaView>(`${planPath(plan)}/days/${day}/done`),
};

export const rehaKeys = {
  all: ["reha"] as const,
  horse: (horse: string) => ["reha", horse] as const,
};

export const useReha = (horse: string | undefined) =>
  useQuery({
    queryKey: rehaKeys.horse(horse ?? ""),
    queryFn: () => rehaApi.get(horse!),
    enabled: !!horse,
  });

/** After a change of the plan: the reha view, "Was heute?", the week and the horse list are stale. */
export function useInvalidateReha() {
  const client = useQueryClient();
  return (horse: string) => {
    void client.invalidateQueries({ queryKey: rehaKeys.horse(horse) });
    void client.invalidateQueries({ queryKey: ["training"] });
    void client.invalidateQueries({ queryKey: ["requests"] });
  };
}

/** Runs a mutation that returns the new view and stores it as the horse's reha view. */
function useViewMutation<V>(horse: string, fn: (v: V) => Promise<RehaView>) {
  const client = useQueryClient();
  const invalidate = useInvalidateReha();
  return useMutation({
    mutationFn: fn,
    onSuccess: (view) => {
      client.setQueryData(rehaKeys.horse(horse), view);
      invalidate(horse);
    },
  });
}

export const useCreatePlan = (horse: string) =>
  useViewMutation(horse, (body: PlanInput & { observation_id?: string }) => rehaApi.create(horse, body));

export const useUpdatePlan = (horse: string, plan: string) =>
  useViewMutation(horse, (body: Partial<PlanInput>) => rehaApi.update(plan, body));

export const useEndPlan = (horse: string, plan: string) => useViewMutation(horse, () => rehaApi.end(plan));

export const useMarkDone = (horse: string, plan: string) =>
  useViewMutation(horse, (day: string) => rehaApi.markDone(plan, day));

export const useUnmarkDone = (horse: string, plan: string) =>
  useViewMutation(horse, (day: string) => rehaApi.unmarkDone(plan, day));

/** German error text; reha-specific codes first, then the generic ones. */
export function rehaError(err: unknown): string {
  if (err instanceof ApiError) return rehaErrorText(err.code) ?? errorMessage(err);
  return errorMessage(err);
}
