// Training endpoints (docs/domains/training.md) with React Query keys and hooks.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, authed, errorMessage } from "../api";
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
  show: { name: string; classes?: string; helper?: string } | null;
  can_take: boolean;
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
  focus_rating?: 1 | 2 | 3;
  exercise_id?: string;
  note?: string;
  visible_to_rider?: boolean;
  distance_m?: number;
};

export type CreatedSession = {
  session: { id: string; activity: Activity; minutes: number; load: number; intensity: Intensity; day: string };
  next_progression: { id: string; title: string; discipline: string; level: string } | null;
};

export type ProfileShow = { date: string; name: string; classes?: string; helper?: string };

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
  takeDay: (horse: string, day: string, body: { status?: "planned" | "rest" | "open"; activity?: Activity }) =>
    authed.put<WeekResponse>(`${base(horse)}/week/${day}`, body),
  profile: (horse: string) => authed.get<TrainingProfile>(`${base(horse)}/training-profile`),
  saveProfile: (horse: string, body: ProfileInput) => authed.put<TrainingProfile>(`${base(horse)}/training-profile`, body),
  createSession: (horse: string, body: SessionInput) => authed.post<CreatedSession>(`${base(horse)}/sessions`, body),
  exercise: (id: string) => authed.get<ExerciseSummary>(`/api/v1/exercises/${encodeURIComponent(id)}`),
};

// --- query keys ----------------------------------------------------------------------------

export const trainingKeys = {
  all: ["training"] as const,
  horses: () => ["training", "horses"] as const,
  today: (horse: string, minutes?: number) => ["training", "today", horse, minutes ?? 0] as const,
  week: (horse: string, start?: string) => ["training", "week", horse, start ?? "current"] as const,
  profile: (horse: string) => ["training", "profile", horse] as const,
};

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

/** After a session or slot change: week, today and horse list are stale. */
export function useInvalidateTraining() {
  const client = useQueryClient();
  return (horse: string) => {
    void client.invalidateQueries({ queryKey: ["training", "week", horse] });
    void client.invalidateQueries({ queryKey: ["training", "today", horse] });
    void client.invalidateQueries({ queryKey: trainingKeys.horses() });
  };
}

export function useCreateSession(horse: string) {
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: (body: SessionInput) => trainingApi.createSession(horse, body),
    onSuccess: () => invalidate(horse),
  });
}

export function useTakeDay(horse: string) {
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: (v: { day: string; status?: "planned" | "rest" | "open"; activity?: Activity }) =>
      trainingApi.takeDay(horse, v.day, { status: v.status, activity: v.activity }),
    onSuccess: () => invalidate(horse),
  });
}

export function useSaveProfile(horse: string) {
  const client = useQueryClient();
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: (body: ProfileInput) => trainingApi.saveProfile(horse, body),
    onSuccess: (profile) => {
      client.setQueryData(trainingKeys.profile(horse), profile);
      invalidate(horse);
    },
  });
}

/** German error text; training-specific codes (forbidden, conflict) first, then the generic ones. */
export function trainingError(err: unknown): string {
  if (err instanceof ApiError) return trainingErrorText(err.code) ?? errorMessage(err);
  return errorMessage(err);
}
