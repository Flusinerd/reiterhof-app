// The guard behind pushOnce (lib/navigation.ts). Pure, no React Native imports (unit-tested).

/**
 * Answers `true` for a navigation to `key`, unless the same target was let through less than `windowMs` ago:
 * the second tap of a double tap, which would stack two identical screens (and "back" shows the same screen
 * again). Another target is never held back.
 */
export function createPushGuard(windowMs: number, now: () => number = Date.now): (key: string) => boolean {
  let last: { key: string; at: number } | null = null;
  return (key) => {
    const at = now();
    if (last && last.key === key && at - last.at < windowMs) return false;
    last = { key, at };
    return true;
  };
}
