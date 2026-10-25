// A one-line result that the next screen shows ("Luna wurde gelöscht."): the flow that leaves its own
// screen sets it, the screen it lands on reads it (components/flash-notice.tsx) and clears it when the
// user leaves or closes it. Pure store, no React Native imports (unit-tested in flash.test.ts).

type Listener = () => void;

let current: string | null = null;
const listeners = new Set<Listener>();

export const flash = {
  get: (): string | null => current,
  set(text: string): void {
    current = text;
    for (const l of [...listeners]) l();
  },
  clear(): void {
    if (current === null) return;
    current = null;
    for (const l of [...listeners]) l();
  },
  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => void listeners.delete(listener);
  },
};
