// Reconnect delay for the event stream (pure, unit-tested).

export type BackoffOptions = {
  baseMs?: number;
  maxMs?: number;
  /** Fraction of the delay that is randomised (0 = none, 0.5 = +-25 %). */
  jitter?: number;
  /** Returns a number in [0, 1); injectable for tests. */
  random?: () => number;
};

/**
 * Exponential backoff: 1 s, 2 s, 4 s ... capped at 30 s. `attempt` counts from 0
 * (first retry).
 */
export function backoffDelay(attempt: number, options: BackoffOptions = {}): number {
  const { baseMs = 1000, maxMs = 30_000, jitter = 0.5, random = Math.random } = options;
  const n = Math.max(0, Math.floor(attempt));
  const raw = Math.min(maxMs, baseMs * 2 ** Math.min(n, 30));
  const spread = raw * jitter;
  return Math.round(raw - spread / 2 + random() * spread);
}
