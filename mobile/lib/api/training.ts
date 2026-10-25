// Training endpoints (docs/domains/training.md, docs/domains/training-planning.md) with React Query
// keys and hooks. Everything the planning rebuild adds (focuses, units, blocks, equipment, goals)
// is optional on read, so an older backend still works.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, authed, errorMessage } from "../api";
import { retryServerErrors } from "../api-core.ts";
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
import { exerciseQueryString } from "../tracking-exercises";
import { equipmentBody, type LocationEquipment, type LocationKey, type StableEquipment } from "../training-equipment";
import type { Block, FocusEntry, FocusWarning, HorsePhase, ProfileFocus, Unit } from "../training-focus";
import type { GoalLevel, LevelProgress, TodayProgress } from "../training-goal";
import { planApplyBody, type PlanDay, type PlanResponse, type ReproposeOptions, type TakeDayBody } from "../training-plan";
import type { DayRuleApi, QuotasApi } from "../training-profile";
import { useEffectMutation } from "./mutation";

export type { ReproposeOptions, TakeDayBody };
export type { Block, FocusEntry, FocusWarning, HorsePhase, ProfileFocus, Unit };
export type { LevelProgress, StableEquipment, TodayProgress };
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
  /** The level goal as stored (null: the class above the current one) and its end date; sent with or without `progress`. */
  goal_level?: GoalLevel | null;
  goal_until?: string | null;
};

/** Body of PUT /training-profile. */
export type ProfileInput = Omit<
  TrainingProfile,
  "horse_id" | "horse_name" | "exists" | "can_edit" | "phase" | "focus_hints" | "warnings" | "progress" | "goal_level" | "goal_until"
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
  /** `facility` is the key list the server sent (`keys.facility`), so that keys the app does not know are kept. */
  saveEquipment: (locations: Record<LocationKey, LocationEquipment>, facility?: readonly string[]) =>
    authed.put<unknown>("/api/v1/stables/training-equipment", equipmentBody(locations, facility)),
};

// --- hooks (the query keys live in lib/query-keys.ts) ------------------------------------------

// The optional endpoints (focuses, my focus, equipment) answer 403 or 404 on an older backend and for
// people without access: `retryServerErrors` spares them the retries that would only delay the screen.

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

/** Saves the profile (also its focuses and level goal); the answer replaces the cached profile, plans and today are stale. */
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

/** Saves the person's own focuses on the horse ("Mein Schwerpunkt"). */
export function useSetMyFocus(horse: string) {
  return useEffectMutation({
    mutationFn: (keys: string[]) => trainingApi.setMyFocus(horse, keys),
    effects: () => [{ type: "myFocus", horseId: horse }],
  });
}

/** "Kein eigener Schwerpunkt": removes the person's own focuses on the horse. */
export function useClearMyFocus(horse: string) {
  return useEffectMutation({
    mutationFn: () => trainingApi.clearMyFocus(horse),
    effects: () => [{ type: "myFocus", horseId: horse }],
  });
}

/** Saves the stable's equipment (admins); exercises (availability), plans and today of every horse are stale. */
export function useSaveTrainingEquipment() {
  return useEffectMutation({
    mutationFn: ({ locations, facility }: { locations: Record<LocationKey, LocationEquipment>; facility?: readonly string[] }) =>
      trainingApi.saveEquipment(locations, facility),
    effects: () => [{ type: "trainingEquipment" }],
  });
}

/** German error text; training-specific codes (forbidden, conflict) first, then the generic ones. */
export function trainingError(err: unknown): string {
  if (err instanceof ApiError) return trainingErrorText(err.code) ?? errorMessage(err);
  return errorMessage(err);
}
