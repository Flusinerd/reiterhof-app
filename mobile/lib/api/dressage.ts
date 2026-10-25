// Practice tests and freestyles (docs/domains/dressage-tests.md): types of the requests, the calls and
// the React Query hooks. The backend may not have the routes yet: a 404 on the list means "not
// available" (the entry row on the training tab is hidden, see `isUnavailable`).
import { useMutation, useQuery } from "@tanstack/react-query";

import { ApiError, authed } from "../api";
import { retryServerErrors } from "../api-core.ts";
import {
  normalizeCatalog,
  normalizePlan,
  normalizeSet,
  type ArenaSize,
  type Catalog,
  type Plan,
  type RequirementMovement,
  type RequirementSet,
  type TestKind,
} from "../dressage.ts";
import { dressageKeys } from "../query-keys.ts";
import { trainingError } from "./training";
import { useEffectMutation } from "./mutation";

export { dressageKeys };

/** Body of POST /horses/{id}/dressage-tests. */
export type GenerateBody = {
  kind: TestKind;
  class: string;
  arena: ArenaSize;
  /** `false` returns the plan without storing it (preview), `true` stores it. */
  save: boolean;
  seed?: number;
  // Practice
  focus_key?: string;
  length?: "short" | "normal";
  // Freestyle: a saved set, or the editor's content.
  requirements_id?: string;
  requirements?: { movements: RequirementMovement[]; min_seconds: number; max_seconds: number };
};

/** A row of GET /horses/{id}/dressage-tests. Duration and class come from the row or from its plan. */
export type SavedTest = {
  id: string;
  kind: TestKind;
  class: string;
  arena: ArenaSize;
  title: string;
  seconds: number;
  created_at: string;
  focus_key?: string;
};

/** A stored test: the row and its plan. */
export type StoredTest = { id: string; horse_id?: string; title: string; focus_key?: string; plan: Plan };

/** Body of the requirement set routes. */
export type RequirementSetBody = { class: string; label: string; movements: RequirementMovement[]; min_seconds: number; max_seconds: number };

const base = (horse: string) => `/api/v1/horses/${encodeURIComponent(horse)}/dressage-tests`;
const one = (id: string) => `/api/v1/dressage-tests/${encodeURIComponent(id)}`;

function toSaved(raw: Record<string, unknown>): SavedTest {
  const plan = normalizePlan(raw);
  return {
    id: String(raw.id ?? ""),
    kind: (raw.kind ?? plan?.kind) === "freestyle" ? "freestyle" : "practice",
    class: String(raw.class ?? plan?.class ?? ""),
    arena: (raw.arena ?? plan?.arena) === "20x60" ? "20x60" : "20x40",
    title: String(raw.title ?? ""),
    seconds: typeof raw.seconds === "number" ? raw.seconds : (plan?.seconds ?? 0),
    created_at: String(raw.created_at ?? ""),
    focus_key: typeof raw.focus_key === "string" ? raw.focus_key : undefined,
  };
}

function toStored(raw: unknown): StoredTest {
  const root = (raw ?? {}) as Record<string, unknown>;
  const plan = normalizePlan(raw);
  if (!plan) throw new ApiError(500, "internal", "The test has no plan");
  return {
    id: String(root.id ?? ""),
    horse_id: typeof root.horse_id === "string" ? root.horse_id : undefined,
    title: String(root.title ?? ""),
    focus_key: typeof root.focus_key === "string" ? root.focus_key : undefined,
    plan,
  };
}

export const dressageApi = {
  catalog: () => authed.get<unknown>("/api/v1/dressage/catalog").then(normalizeCatalog),
  list: (horse: string) =>
    authed.get<unknown>(base(horse)).then((r) => {
      const rows = Array.isArray(r) ? r : ((r as { tests?: unknown })?.tests ?? []);
      return (Array.isArray(rows) ? rows : []).map((x) => toSaved(x as Record<string, unknown>));
    }),
  /** A preview (`save: false`) answers with the plan, a stored test with the row and its plan. */
  generate: async (horse: string, body: GenerateBody): Promise<{ plan: Plan; stored: StoredTest | null }> => {
    const raw = await authed.post<unknown>(base(horse), body);
    const plan = normalizePlan(raw);
    if (!plan) throw new ApiError(500, "internal", "The answer has no plan");
    const id = (raw as { id?: unknown })?.id;
    return { plan, stored: body.save && typeof id === "string" ? toStored(raw) : null };
  },
  get: (id: string) => authed.get<unknown>(one(id)).then(toStored),
  /** Stores the user's edits; the server re-checks the rules (400 with the reasons) and answers with the checked plan. */
  update: (id: string, body: { title: string; plan: Plan }) => authed.put<unknown>(one(id), body).then(toStored),
  remove: (id: string) => authed.delete<unknown>(one(id)),
  createSet: (body: RequirementSetBody) =>
    authed.post<unknown>("/api/v1/dressage/requirements", body).then((r) => normalizeSet((r ?? {}) as Record<string, unknown>)),
  updateSet: (id: string, body: RequirementSetBody) =>
    authed.put<unknown>(`/api/v1/dressage/requirements/${encodeURIComponent(id)}`, body).then((r) => normalizeSet((r ?? {}) as Record<string, unknown>)),
  removeSet: (id: string) => authed.delete<unknown>(`/api/v1/dressage/requirements/${encodeURIComponent(id)}`),
};

/** The backend answers 404 for the route: it does not have practice tests yet. */
export function isUnavailable(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404;
}

/** Error text; a rule break (400) carries the German reasons of the server. */
export function dressageError(err: unknown): string {
  if (err instanceof ApiError && err.status === 400 && err.code !== "unknown" && err.message !== err.code && !/^Request failed/.test(err.message)) {
    return err.message;
  }
  return trainingError(err);
}

// --- hooks ----------------------------------------------------------------------------------

/** The catalog is static except for the saved requirement sets: fresh on every visit of the screen. */
export const useDressageCatalog = () =>
  useQuery<Catalog, Error>({ queryKey: dressageKeys.catalog(), queryFn: dressageApi.catalog, retry: retryServerErrors });

export const useDressageTests = (horse: string | undefined) =>
  useQuery<SavedTest[], Error>({
    queryKey: dressageKeys.list(horse ?? ""),
    queryFn: () => dressageApi.list(horse!),
    enabled: !!horse,
    retry: retryServerErrors,
  });

export const useDressageTest = (id: string | undefined) =>
  useQuery<StoredTest, Error>({
    queryKey: dressageKeys.detail(id ?? ""),
    queryFn: () => dressageApi.get(id!),
    enabled: !!id,
    retry: retryServerErrors,
  });

/** Generates a preview (nothing is stored, nothing becomes stale). */
export function useGenerateTest(horse: string) {
  return useMutation({ mutationFn: (body: GenerateBody) => dressageApi.generate(horse, { ...body, save: false }) });
}

/** Stores a generated plan: the server generates it again from the seed, so the stored one equals the preview. */
export function useSaveTest(horse: string) {
  return useEffectMutation({
    mutationFn: (body: GenerateBody) => dressageApi.generate(horse, { ...body, save: true }),
    effects: () => [{ type: "dressageTests", horseId: horse }],
  });
}

/** Stores edits of a stored test (title, reordered or removed lines, or a re-rolled plan). */
export function useUpdateTest(horse: string) {
  return useEffectMutation({
    mutationFn: ({ id, ...body }: { id: string; title: string; plan: Plan }) => dressageApi.update(id, body),
    effects: () => [{ type: "dressageTests", horseId: horse }],
  });
}

export function useRemoveTest(horse: string) {
  return useEffectMutation({
    mutationFn: (id: string) => dressageApi.remove(id),
    effects: () => [{ type: "dressageTests", horseId: horse }],
  });
}

/** Saves the editor's content as a set for the stable (the catalog lists the sets, so it is refreshed). */
export function useSaveRequirementSet(horse: string) {
  return useEffectMutation({
    mutationFn: (body: RequirementSetBody & { id?: string }) => {
      const { id, ...rest } = body;
      return id ? dressageApi.updateSet(id, rest) : dressageApi.createSet(rest);
    },
    effects: () => [{ type: "dressageTests", horseId: horse }],
  });
}

export function useRemoveRequirementSet(horse: string) {
  return useEffectMutation({
    mutationFn: (id: string) => dressageApi.removeSet(id),
    effects: () => [{ type: "dressageTests", horseId: horse }],
  });
}

export type { RequirementSet };
