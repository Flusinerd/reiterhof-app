// Training endpoints (docs/domains/training.md) with React Query keys and hooks.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, authed, errorMessage } from "../api";
import { trainingKeys } from "../query-keys.ts";
import {
  trainingErrorText,
  type Activity,
  type ActivityMode,
  type ApiDot,
  type ApiSegment,
  type DayStatus,
  type Feel,
  type Intensity,
  type ReinSegment,
} from "../training";
import type { Feeling } from "../session-finish";
import { planApplyBody, type PlanDay, type PlanResponse, type ReproposeOptions, type TakeDayBody } from "../training-plan";
import type { DayRuleApi, QuotasApi } from "../training-profile";
import { useEffectMutation } from "./mutation";

export type { ReproposeOptions, TakeDayBody };
export { trainingKeys };

// --- types (mirror backend/internal/trainingapi) -----------------------------------------

export type TrainingHorse = {
  id: string;
  name: string;
  color_key: string;
  role: "owner" | "rider";
  profile_status: "fit" | "reha" | "pause" | "";
  has_profile: boolean;
};

export type ExerciseSummary = {
  id: string;
  title: string;
  discipline: string;
  level: string;
  goal_tags: string[];
  global: boolean;
  step_count: number;
  steps?: string[];
  next_exercise_id: string | null;
  next?: { id: string; title: string; discipline: string; level: string } | null;
};

export type Recommendation = {
  activity: Activity | "rest";
  label: string;
  minutes: number;
  intensity: Intensity;
  intensity_label: string;
  reason: string;
  note?: string;
  exercise?: ExerciseSummary | null;
};

export type TodayResponse = {
  horse_id: string;
  horse_name: string;
  date: string;
  has_profile: boolean;
  status: "fit" | "reha" | "pause";
  context: string;
  recommendations: Recommendation[];
  hidden: { activity: Activity; label: string; reason: string }[];
  week: ApiDot[];
  reha: {
    plan_id: string;
    phase: string;
    phase_index: number;
    phases: number;
    activity: string;
    /** Today's allowed minutes (ramp), 0 for a rest phase. */
    minutes: number;
    rest: boolean;
    done: boolean;
    text: string;
  } | null;
  can_log: boolean;
  can_edit: boolean;
};

export type WeekDay = {
  date: string;
  weekday: number;
  status: DayStatus;
  is_today: boolean;
  user: { id: string; name: string; color_key?: string } | null;
  is_me: boolean;
  activity: Activity | null;
  label?: string;
  minutes: number;
  note: string | null;
  rest_reason?: string;
  show: {
    name: string;
    /** Start of the show, "HH:MM". */
    time?: string;
    /** Classes (Prüfungen) the horse starts in. */
    tests: { name: string; time?: string }[];
    /** Free text of older app versions. */
    classes?: string;
    helper?: string;
  } | null;
  can_take: boolean;
  /** Focus and library exercise planned for the day (week plan). */
  focus: string | null;
  exercise: { id: string; title: string } | null;
  /** The unit the active reha plan allows that day. */
  reha: {
    plan_id: string;
    phase: string;
    activity: string;
    activity_label: string;
    rest: boolean;
    minutes: number;
    done: boolean;
  } | null;
};

export type WeekResponse = {
  horse_id: string;
  start: string;
  end: string;
  days: WeekDay[];
  segments: ApiSegment[];
  assessment: string;
  sessions: number;
  can_edit: boolean;
};

export type SessionInput = {
  activity: Activity;
  minutes: number;
  canter_share?: number;
  started_at?: string;
  gait_shares?: Record<string, number>;
  rein_changes?: ReinSegment[];
  feel?: Feel;
  /** How the session felt for the rider ("Wie war's?"), optional for older clients. */
  feeling?: Feeling;
  focus_rating?: 1 | 2 | 3;
  exercise_id?: string;
  note?: string;
  visible_to_rider?: boolean;
  distance_m?: number;
  /** Simplified GPS track of a ride (lib/tracking.ts buildApiTrack), at most 5000 points. */
  track?: { lat: number; lon: number; t: number; alt?: number; g?: string }[];
  /** Raw gait windows for model improvement (lib/gait/export.ts WindowRecord), at most 3000. */
  gait_windows?: { t: number; f: number[]; p: string; a: string; v?: number; c?: string }[];
};

export type CreatedSession = {
  session: { id: string; activity: Activity; minutes: number; load: number; intensity: Intensity; day: string };
  next_progression: { id: string; title: string; discipline: string; level: string } | null;
};

export type ProfileShow = {
  date: string;
  time?: string;
  name: string;
  tests?: { name: string; time?: string }[];
  classes?: string;
  helper?: string;
};

export type RiderRule = {
  user_id: string;
  name?: string;
  allowed_activities: Activity[];
  max_intensity: "any" | "light" | "medium" | "intense";
  may_hack_alone: boolean;
  may_ride_shows: boolean;
};

export type Rhythm = {
  sessions_min: number;
  sessions_max: number;
  rest_days_min: number;
  rest_days_max: number;
  max_minutes: number;
  rest_after_show: boolean;
  /** Owner's week structure, Monday first (JAN-93). */
  days: DayRuleApi[];
  quotas: QuotasApi;
};

export type ProfileActivity = { activity: Activity; mode: ActivityMode; note?: string };

export type TrainingProfile = {
  horse_id: string;
  horse_name: string;
  exists: boolean;
  can_edit: boolean;
  discipline: string;
  level: string;
  allowed_activities: ProfileActivity[];
  shows: ProfileShow[];
  season_end: string | null;
  rhythm: Rhythm;
  status: "fit" | "reha" | "pause";
  rider_rules: RiderRule[];
};

/** Body of PUT /training-profile. */
export type ProfileInput = Omit<TrainingProfile, "horse_id" | "horse_name" | "exists" | "can_edit">;

// --- calls -------------------------------------------------------------------------------

const base = (horse: string) => `/api/v1/horses/${encodeURIComponent(horse)}`;

export const trainingApi = {
  horses: () => authed.get<{ horses: TrainingHorse[] }>("/api/v1/training/horses").then((r) => r.horses),
  today: (horse: string, minutes?: number) =>
    authed.get<TodayResponse>(`${base(horse)}/today${minutes ? `?minutes=${minutes}` : ""}`),
  week: (horse: string, start?: string) =>
    authed.get<WeekResponse>(`${base(horse)}/week${start ? `?start=${start}` : ""}`),
  takeDay: (horse: string, day: string, body: TakeDayBody) =>
    authed.put<WeekResponse>(`${base(horse)}/week/${day}`, body),
  /**
   * Proposal for the open days of the week (JAN-89); nothing is stored. With `repropose` only that
   * day is proposed again, the other draft days are context and `exclude` lists rejected activities.
   */
  planWeek: (horse: string, start?: string, repropose?: ReproposeOptions) => {
    const query = [start ? `start=${start}` : "", repropose ? `day=${repropose.day}` : ""].filter(Boolean).join("&");
    const body = repropose ? { draft: repropose.draft, exclude: repropose.exclude } : {};
    return authed.post<PlanResponse>(`${base(horse)}/week/plan${query ? `?${query}` : ""}`, body);
  },
  profile: (horse: string) => authed.get<TrainingProfile>(`${base(horse)}/training-profile`),
  saveProfile: (horse: string, body: ProfileInput) => authed.put<TrainingProfile>(`${base(horse)}/training-profile`, body),
  createSession: (horse: string, body: SessionInput) => authed.post<CreatedSession>(`${base(horse)}/sessions`, body),
  exercise: (id: string) => authed.get<ExerciseSummary>(`/api/v1/exercises/${encodeURIComponent(id)}`),
};

// --- hooks (the query keys live in lib/query-keys.ts) ------------------------------------------

export const useTrainingHorses = () => useQuery({ queryKey: trainingKeys.horses(), queryFn: trainingApi.horses });

export const useToday = (horse: string | undefined, minutes?: number) =>
  useQuery({
    queryKey: trainingKeys.today(horse ?? "", minutes),
    queryFn: () => trainingApi.today(horse!, minutes),
    enabled: !!horse,
  });

export const useWeek = (horse: string | undefined, start?: string) =>
  useQuery({
    queryKey: trainingKeys.week(horse ?? "", start),
    queryFn: () => trainingApi.week(horse!, start),
    enabled: !!horse,
  });

export const useProfile = (horse: string | undefined) =>
  useQuery({
    queryKey: trainingKeys.profile(horse ?? ""),
    queryFn: () => trainingApi.profile(horse!),
    enabled: !!horse,
  });

/**
 * A logged session changes the week and today of the horse, the horse list, the diary and the reminder
 * center (the day is done); the mutation ends when they are fresh, so "Ins Tagebuch" is there when the
 * screen closes.
 */
export function useCreateSession(horse: string) {
  return useEffectMutation({
    mutationFn: (body: SessionInput) => trainingApi.createSession(horse, body),
    effects: () => [{ type: "session", horseId: horse }],
  });
}

/** Claims, releases or plans one day of the week. */
export function useTakeDay(horse: string) {
  return useEffectMutation({
    mutationFn: ({ day, ...body }: { day: string } & TakeDayBody) => trainingApi.takeDay(horse, day, body),
    effects: () => [{ type: "weekSlots", horseId: horse }],
  });
}

/** Asks the server for a week plan (model and/or rules); a plain string is the start of the week. */
export function usePlanWeek(horse: string) {
  return useMutation({
    mutationFn: (v?: string | { start?: string; repropose?: ReproposeOptions }) =>
      typeof v === "object" ? trainingApi.planWeek(horse, v.start, v.repropose) : trainingApi.planWeek(horse, v),
  });
}

/** Proposes one day of the week again (own mutation, so that its pending state is separate). */
export function useReproposeDay(horse: string) {
  return useMutation({
    mutationFn: (v: { start?: string; repropose: ReproposeOptions }) => trainingApi.planWeek(horse, v.start, v.repropose),
  });
}

/**
 * Stores the plan days one after another, then refreshes the week. Days stored before a failure stay
 * stored, so the week is refreshed on failure too.
 */
export function useApplyPlan(horse: string) {
  return useEffectMutation({
    mutationFn: async (days: PlanDay[]) => {
      for (const d of days) {
        const body = planApplyBody(d);
        if (body) await trainingApi.takeDay(horse, d.date, body);
      }
    },
    effects: () => [{ type: "weekSlots", horseId: horse }],
    errorEffects: () => [{ type: "weekSlots", horseId: horse }],
  });
}

export function useSaveProfile(horse: string) {
  const client = useQueryClient();
  return useEffectMutation({
    mutationFn: (body: ProfileInput) => trainingApi.saveProfile(horse, body),
    onSuccess: (profile) => {
      client.setQueryData(trainingKeys.profile(horse), profile);
    },
    effects: () => [{ type: "trainingProfile", horseId: horse }],
  });
}

/** German error text; training-specific codes (forbidden, conflict) first, then the generic ones. */
export function trainingError(err: unknown): string {
  if (err instanceof ApiError) return trainingErrorText(err.code) ?? errorMessage(err);
  return errorMessage(err);
}
