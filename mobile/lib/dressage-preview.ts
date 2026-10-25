// The preview of a generated test lives between the "Neue Aufgabe" screen and the plan view
// (route `/training/tests/preview`): the plan, and the request that made it so that "Neu würfeln" and
// "Speichern" can repeat it with the seed. Memory only; a reload of the page loses it on purpose.
import type { GenerateBody } from "./api/dressage.ts";
import type { Plan } from "./dressage.ts";

export type Preview = { horseId: string; request: Omit<GenerateBody, "save">; plan: Plan };

let current: Preview | null = null;

export function setPreview(p: Preview | null): void {
  current = p;
}

export function getPreview(): Preview | null {
  return current;
}

/** A seed for "Neu würfeln" (the server accepts and returns it, so the stored test equals the preview). */
export function newSeed(): number {
  return Math.floor(Math.random() * 2_000_000_000) + 1;
}
