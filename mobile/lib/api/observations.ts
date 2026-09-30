// API calls and react-query hooks for observations ("Auffälligkeiten"), see
// docs/domains/observations.md (backend: internal/observations).

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { authed } from "../api";
import { useInvalidateOnEvents } from "../realtime";
import { horseKeys, type EmergencyCard } from "./horses";

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

export const observationKeys = {
  all: ["observations"] as const,
  list: (horseId: string) => ["observations", "list", horseId] as const,
  detail: (id: string) => ["observations", "detail", id] as const,
};

/** Refetches observations when someone reports or changes one. Call once per screen. */
export function useObservationEvents() {
  useInvalidateOnEvents({ "observation.changed": [observationKeys.all] });
}

export const useObservations = (horseId: string) =>
  useQuery({ queryKey: observationKeys.list(horseId), queryFn: () => observationsApi.list(horseId) });

export const useObservation = (id: string) =>
  useQuery({ queryKey: observationKeys.detail(id), queryFn: () => observationsApi.get(id) });

/** Sends a report. An urgent one puts the emergency card into the cache so it opens instantly. */
export function useReportObservation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: observationsApi.create,
    onSuccess: (result) => {
      if (result.emergency) {
        queryClient.setQueryData(horseKeys.emergency(result.observation.horse_id), result.emergency);
      }
      return queryClient.invalidateQueries({ queryKey: observationKeys.all });
    },
  });
}

export function useSetObservationStatus() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (v: { id: string; status: "watch" | "done" }) => observationsApi.setStatus(v.id, v.status),
    onSuccess: (o) => {
      queryClient.setQueryData(observationKeys.detail(o.id), o);
      return queryClient.invalidateQueries({ queryKey: observationKeys.all });
    },
  });
}
