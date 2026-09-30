// React Query hooks of the exercise library (JAN-58) and the exercise checklist (JAN-65).
// Endpoints: docs/domains/training.md ("GET /exercises", "GET /exercises/{id}").
import { useQuery } from "@tanstack/react-query";

import { authed } from "./api";
import { trainingApi, type ExerciseSummary } from "./api/training";

/** All exercises of the library (global and own stable), easiest first; filtering is local. */
export const useExerciseList = () =>
  useQuery({
    queryKey: ["training", "exercises"],
    queryFn: () => authed.get<{ exercises: ExerciseSummary[] }>("/api/v1/exercises").then((r) => r.exercises),
  });

/** One exercise with steps and follow-up; shares its cache key with the finish screen. */
export const useExercise = (id: string | undefined) =>
  useQuery({
    queryKey: ["training", "exercise", id],
    queryFn: () => trainingApi.exercise(id!),
    enabled: !!id,
  });
