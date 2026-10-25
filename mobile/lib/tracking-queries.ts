// React Query hooks of the exercise library (JAN-58) and the exercise checklist (JAN-65).
// Endpoints: docs/domains/training.md and docs/domains/training-planning.md ("GET /exercises",
// "GET /exercises/{id}").
import { useQuery } from "@tanstack/react-query";

import { trainingApi, trainingKeys, type ExerciseQuery } from "./api/training";

/**
 * The exercises of the library (global and own stable), easiest first; filtering is local except for
 * the focus, which the server matches against the focus's tags. With a `horse` every item says
 * whether the stable's equipment is enough for it (`available`). Every combination is cached on its own.
 */
export const useExerciseList = (query: ExerciseQuery = {}, enabled = true) =>
  useQuery({
    queryKey: [...trainingKeys.exercises(), query.horse ?? "", query.focus ?? ""],
    queryFn: () => trainingApi.exercises(query),
    enabled,
  });

/** One exercise with steps and follow-up; shares its cache key with the finish screen. */
export const useExercise = (id: string | undefined) =>
  useQuery({
    queryKey: ["training", "exercise", id],
    queryFn: () => trainingApi.exercise(id!),
    enabled: !!id,
  });
