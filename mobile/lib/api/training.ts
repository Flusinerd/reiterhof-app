// Training endpoints (docs/domains/training.md, docs/domains/training-planning.md) with React Query
// keys and hooks. Everything the planning rebuild adds (focuses, units, blocks, equipment, goals)
// is optional on read, so an older backend still works.
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
import type { Feeling } from "../session-finish";
import { exerciseQueryString } from "../tracking-exercises";
import { equipmentBody, type LocationEquipment, type LocationKey, type StableEquipment } from "../training-equipment";
import type { Block, FocusEntry, FocusWarning, HorsePhase, ProfileFocus, Unit } from "../training-focus";
import type { GoalLevel, LevelProgress, TodayProgress } from "../training-goal";
import { planApplyBody, type PlanDay, type PlanResponse, type ReproposeOptions, type TakeDayBody } from "../training-plan";
import type { DayRuleApi, QuotasApi } from "../training-profile";

export type { ReproposeOptions, TakeDayBody };
export type { Block, FocusEntry, FocusWarning, HorsePhase, ProfileFocus, Unit };
export type { LevelProgress, StableEquipment, TodayProgress };

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
  /** The library (dressage, jumping, cavaletti, groundwork, lunge, hack, western); same value as `library`. */
  discipline: string;
  level: string;
  goal_tags: string[];
  global: boolean;
  step_count: number;
  steps?: string[];
  next_exercise_id: string | null;
  next?: { id: string; title: string; discipline: string; level: string } | null;
  // Metadata of the catalog (docs/domains/training-planning.md); absent on older backends.
  library?: string;
  /** The part of the unit it fits: warmup (Lösen), main (Arbeiten), cooldown (Entspannen) or any. */
  phase?: "warmup" | "main" | "cooldown" | "any";
  /** Typical duration in minutes. */
  minutes?: number | null;
  intensity?: "light" | "normal" | "demanding";
  /** `young` and `senior`. */
  suits?: string[];
  /** One sentence: what the exercise is for ("Wozu"). */
  summary?: string;
  /** Keys of the equipment it needs (facility and personal). */
  equipment?: string[];
  /** Only in the list with `?horse=`: false when the stable's equipment is not enough. */
  available?: boolean;
  // Detail only.
  requires?: { id: string; title: string }[];
  sources?: string[];
  alternatives?: { id: string; title: string; equipment?: string[] }[];
};

/** Why a unit has its focus (`why` names the cause: "Schwerpunkt Takt: zuletzt am Montag."). */
export type RecommendationFocus = { key: string; label: string; why: string };

export type Recommendation = {
  activity: Activity | "rest";
  label: string;
  minutes: number;
  intensity: Intensity;
  intensity_label: string;
  reason: string;
  note?: string;
  exercise?: ExerciseSummary | null;
  focus?: RecommendationFocus | null;
  unit?: Unit | null;
  /** The day's "Aufbau heute" override and the planner's note on the equipment. */
  equipment_ok?: boolean | null;
  equipment_note?: string | null;
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
  /** The horse's phase, the week of the return to work (0 when not returning) and the block of four weeks. */
  phase?: HorsePhase;
  restart_week?: number;
  block?: Block | null;
  /** The way to the level goal: a sentence and the lections coming next. */
  progress?: TodayProgress | null;
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
  /** Focus (free text) and library exercise planned for the day (week plan). */
  focus: string | null;
  exercise: { id: string; title: string } | null;
  /** Focus of the unit from the catalog, the unit plan and the equipment of the day; absent on older backends. */
  focus_key?: string | null;
  focus_label?: string | null;
  unit?: Unit | null;
  /** "Aufbau heute": true (possible), false (not possible), null (the location's default). */
  equipment_ok?: boolean | null;
  equipment_note?: string | null;
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
  /** The block of four weeks the week is in (weeks 1 to 3 build, week 4 is the deload week). */
  block?: Block | null;
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
  /** What was trained (catalog key); only sent when the unit had a focus (newer backends). */
  focus_key?: string;
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
  /** Focuses of the horse in priority order (at most 3) and the Monday of the first week of the block. */
  focuses?: ProfileFocus[];
  cycle_start?: string | null;
  /** The horse's phase, the scale focuses of it that are not chosen yet, hints (focus keys, `level_up`). */
  phase?: HorsePhase;
  focus_hints?: string[];
  warnings?: FocusWarning[];
  /** The way to the level goal; absent on older backends (and for a horse with no class above). */
  progress?: LevelProgress | null;
};

/** Body of PUT /training-profile. */
export type ProfileInput = Omit<
  TrainingProfile,
  "horse_id" | "horse_name" | "exists" | "can_edit" | "phase" | "focus_hints" | "warnings" | "progress"
> & {
  /** The level goal; null is "the class above the current one". Only sent by the goal sheet. */
  goal_level?: GoalLevel | null;
  goal_until?: string | null;
};

/** GET /horses/{id}/my-focus. `catalog_phase` is the highest catalog phase the horse's phase allows. */
export type MyFocus = { keys: string[]; catalog_phase?: number };

// --- calls -------------------------------------------------------------------------------

const base = (horse: string) => `/api/v1/horses/${encodeURIComponent(horse)}`;

export type ExerciseQuery = { horse?: string | null; focus?: string | null };

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
  exercises: (query: ExerciseQuery = {}) =>
    authed.get<{ exercises: ExerciseSummary[] }>(`/api/v1/exercises${exerciseQueryString(query)}`).then((r) => r.exercises),
  exercise: (id: string) => authed.get<ExerciseSummary>(`/api/v1/exercises/${encodeURIComponent(id)}`),
  /** The focus catalog (static data of the backend). */
  focuses: () => authed.get<{ focuses: FocusEntry[] }>("/api/v1/training/focuses").then((r) => r.focuses),
  /** The rider's own focuses on a horse (at most 2). */
  myFocus: (horse: string) => authed.get<MyFocus>(`${base(horse)}/my-focus`),
  setMyFocus: (horse: string, keys: string[]) => authed.put<unknown>(`${base(horse)}/my-focus`, { keys }),
  clearMyFocus: (horse: string) => authed.delete<unknown>(`${base(horse)}/my-focus`),
  /** What the stable has to set up for training (all members read, admins write). */
  equipment: () => authed.get<StableEquipment>("/api/v1/stables/training-equipment"),
  saveEquipment: (locations: Record<LocationKey, LocationEquipment>) =>
    authed.put<unknown>("/api/v1/stables/training-equipment", equipmentBody(locations)),
};

// --- query keys ----------------------------------------------------------------------------

export const trainingKeys = {
  all: ["training"] as const,
  horses: () => ["training", "horses"] as const,
  today: (horse: string, minutes?: number) => ["training", "today", horse, minutes ?? 0] as const,
  week: (horse: string, start?: string) => ["training", "week", horse, start ?? "current"] as const,
  profile: (horse: string) => ["training", "profile", horse] as const,
  sessions: (horse: string) => ["training", "sessions", horse] as const,
  focuses: () => ["training", "focuses"] as const,
  myFocus: (horse: string) => ["training", "my-focus", horse] as const,
  equipment: () => ["training", "equipment"] as const,
  /** Every exercise list (any horse, any focus); the lists themselves are keyed in lib/tracking-queries.ts. */
  exercises: () => ["training", "exercises"] as const,
};

/**
 * Optional endpoints (focuses, my focus, equipment) answer 403 or 404 on an older backend and for
 * people without access: no retries for those, they only delay the screen.
 */
export const retryServerErrors = (count: number, err: unknown) =>
  !(err instanceof ApiError && err.status >= 400 && err.status < 500) && count < 2;

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

/** The focus catalog; static data, so it is kept for an hour. Fails on an older backend (no retries). */
export const useFocuses = () =>
  useQuery({
    queryKey: trainingKeys.focuses(),
    queryFn: trainingApi.focuses,
    staleTime: 60 * 60 * 1000,
    retry: retryServerErrors,
  });

/** The signed-in person's own focuses on the horse; fails for people who may not set them. */
export const useMyFocus = (horse: string | undefined) =>
  useQuery({
    queryKey: trainingKeys.myFocus(horse ?? ""),
    queryFn: () => trainingApi.myFocus(horse!),
    enabled: !!horse,
    retry: retryServerErrors,
  });

/** The stable's training equipment; `enabled` false skips the request (only admins need it on the training tab). */
export const useTrainingEquipment = (enabled = true) =>
  useQuery({
    queryKey: trainingKeys.equipment(),
    queryFn: trainingApi.equipment,
    enabled,
    retry: retryServerErrors,
  });

/** After a session or slot change: week, today, sessions and horse list are stale; the profile when `profile` is set (its progress counts sessions). */
export function useInvalidateTraining() {
  const client = useQueryClient();
  return (horse: string, also?: { profile?: boolean }) => {
    void client.invalidateQueries({ queryKey: ["training", "week", horse] });
    void client.invalidateQueries({ queryKey: ["training", "today", horse] });
    void client.invalidateQueries({ queryKey: trainingKeys.sessions(horse) });
    void client.invalidateQueries({ queryKey: trainingKeys.horses() });
    if (also?.profile) void client.invalidateQueries({ queryKey: trainingKeys.profile(horse) });
  };
}

export function useCreateSession(horse: string) {
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: (body: SessionInput) => trainingApi.createSession(horse, body),
    onSuccess: () => invalidate(horse, { profile: true }),
  });
}

export function useTakeDay(horse: string) {
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: ({ day, ...body }: { day: string } & TakeDayBody) => trainingApi.takeDay(horse, day, body),
    onSuccess: () => invalidate(horse),
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

/** Stores the plan days one after another, then refreshes the week. */
export function useApplyPlan(horse: string) {
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: async (days: PlanDay[]) => {
      for (const d of days) {
        const body = planApplyBody(d);
        if (body) await trainingApi.takeDay(horse, d.date, body);
      }
    },
    onSettled: () => invalidate(horse),
  });
}

/** Saves the profile (also its focuses and level goal); the answer replaces the cached profile, plans and today are stale. */
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

/** Saves the person's own focuses on the horse ("Mein Schwerpunkt"). */
export function useSetMyFocus(horse: string) {
  const client = useQueryClient();
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: (keys: string[]) => trainingApi.setMyFocus(horse, keys),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: trainingKeys.myFocus(horse) });
      invalidate(horse);
    },
  });
}

/** "Kein eigener Schwerpunkt": removes the person's own focuses on the horse. */
export function useClearMyFocus(horse: string) {
  const client = useQueryClient();
  const invalidate = useInvalidateTraining();
  return useMutation({
    mutationFn: () => trainingApi.clearMyFocus(horse),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: trainingKeys.myFocus(horse) });
      invalidate(horse);
    },
  });
}

/** Saves the stable's equipment (admins); exercises (availability), plans and today of every horse are stale. */
export function useSaveTrainingEquipment() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (locations: Record<LocationKey, LocationEquipment>) => trainingApi.saveEquipment(locations),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: trainingKeys.equipment() });
      void client.invalidateQueries({ queryKey: trainingKeys.exercises() });
      void client.invalidateQueries({ queryKey: ["training", "week"] });
      void client.invalidateQueries({ queryKey: ["training", "today"] });
    },
  });
}

/** German error text; training-specific codes (forbidden, conflict) first, then the generic ones. */
export function trainingError(err: unknown): string {
  if (err instanceof ApiError) return trainingErrorText(err.code) ?? errorMessage(err);
  return errorMessage(err);
}
