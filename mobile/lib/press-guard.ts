// The re-entry guard behind Button (docs/design-system.md, "Feedback"). Pure, no React Native imports
// (unit-tested).

/** Whether a press handler is still running. One per button. */
export type PressGuard = { locked: boolean };

export function isThenable(value: unknown): value is PromiseLike<unknown> {
  return typeof value === "object" && value !== null && typeof (value as { then?: unknown }).then === "function";
}

/**
 * Runs a press handler. A handler that returns a promise keeps the guard locked until the promise
 * settles (resolved or not), and further presses are ignored meanwhile: React disables the button
 * with the next render, a fast double tap arrives before it. A handler that returns nothing is a plain
 * action and locks nothing. `hooks` tell the button when to show and hide its spinner.
 *
 * @returns whether the handler ran
 */
export function runPress(guard: PressGuard, handler: () => unknown, hooks: { start: () => void; end: () => void }): boolean {
  if (guard.locked) return false;
  const result = handler();
  if (isThenable(result)) {
    guard.locked = true;
    hooks.start();
    const end = () => {
      guard.locked = false;
      hooks.end();
    };
    // Both branches end the same way; the error itself is the caller's to show (a rejected promise
    // must not leave the button locked, nor throw again from here).
    result.then(end, end);
  }
  return true;
}
