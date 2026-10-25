// Ignore a repeat of the same write (docs/design-system.md, "Feedback"). Pure, no React Native imports
// (unit-tested). The hook is in use-pending.ts.

/**
 * Runs writes per id and ignores a call for an id while its previous call still runs or succeeded less than
 * `cooldownMs` ago: the second tap of a nervous user, and the tap that was already on its way when the
 * first one finished. A failed call does not start a cooldown (the user may try again at once).
 */
export function createSubmitOnce(cooldownMs: number, now: () => number = Date.now) {
  const lastSuccess = new Map<string, number>();
  const running = new Set<string>();
  return {
    /** Resolves with the result, or `undefined` when the call was ignored. A failure is thrown. */
    async run<T>(id: string, call: () => Promise<T>): Promise<T | undefined> {
      if (running.has(id)) return undefined;
      const at = lastSuccess.get(id);
      if (at !== undefined && now() - at < cooldownMs) return undefined;
      running.add(id);
      try {
        const result = await call();
        lastSuccess.set(id, now());
        return result;
      } finally {
        running.delete(id);
      }
    },
  };
}
