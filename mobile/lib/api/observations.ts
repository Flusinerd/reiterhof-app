// API calls and react-query hooks for observations ("Auffälligkeiten"), see
// docs/domains/observations.md (backend: internal/observations).

import { useQuery, useQueryClient } from "@tanstack/react-query";

import { authed } from "../api";
import { observationKeys } from "../query-keys.ts";
import { horseKeys, type EmergencyCard } from "./horses";
import { useEffectMutation } from "./mutation";

export { observationKeys };

// --- types (mirror the backend JSON) -------------------------------------------------------

export type ObservationMedia = { path: string; url: string; content_type: string };

export type Observation = {
  id: string;
  horse_id: string;
  horse_name: string;
  reporter: { id: string; name: string; color_key: string | null };
  category: string | null;
  body_part: string | null;
  description: string | null;
  media: ObservationMedia[];
  urgency: "info" | "check" | "urgent";
  status: "watch" | "done";
  created_at: string;
  /** The caller may change the status (owner, admin or reporter). */
  can_change: boolean;
  /** Reha plan created from this observation (active one first), null if none. */
  reha_plan_id: string | null;
};

/** Body of POST /observations. `media` are paths returned by the files API. */
export type ObservationInput = {
  horse_id: string;
  category: string;
  body_part?: string;
  description?: string;
  media?: string[];
  urgency?: "info" | "check" | "urgent";
};

export type ReportResult = {
  observation: Observation;
  /** Set for urgent reports: open it right away. */
  emergency: EmergencyCard | null;
};

// --- endpoints -----------------------------------------------------------------------------

export const observationsApi = {
  list: (horseId: string, status?: "watch" | "done") =>
    authed.get<Observation[]>(`/api/v1/horses/${horseId}/observations${status ? `?status=${status}` : ""}`),
  get: (id: string) => authed.get<Observation>(`/api/v1/observations/${id}`),
  create: (input: ObservationInput) => authed.post<ReportResult>("/api/v1/observations", input),
  setStatus: (id: string, status: "watch" | "done") =>
    authed.patch<Observation>(`/api/v1/observations/${id}`, { status }),
};

// --- query keys and hooks ------------------------------------------------------------------

export const useObservations = (horseId: string) =>
  useQuery({ queryKey: observationKeys.list(horseId), queryFn: () => observationsApi.list(horseId) });

export const useObservation = (id: string, enabled = true) =>
  useQuery({ queryKey: observationKeys.detail(id), queryFn: () => observationsApi.get(id), enabled: enabled && !!id });

/** Sends a report. An urgent one puts the emergency card into the cache so it opens instantly. */
export function useReportObservation() {
  const queryClient = useQueryClient();
  return useEffectMutation({
    mutationFn: observationsApi.create,
    onSuccess: (result) => {
      if (result.emergency) {
        queryClient.setQueryData(horseKeys.emergency(result.observation.horse_id), result.emergency);
      }
    },
    effects: (_vars, result) => [{ type: "observation", horseId: result.observation.horse_id }],
  });
}

export function useSetObservationStatus() {
  const queryClient = useQueryClient();
  return useEffectMutation({
    mutationFn: (v: { id: string; status: "watch" | "done" }) => observationsApi.setStatus(v.id, v.status),
    onSuccess: (o) => {
      queryClient.setQueryData(observationKeys.detail(o.id), o);
    },
    effects: (_vars, o) => [{ type: "observation", horseId: o.horse_id }],
  });
}
