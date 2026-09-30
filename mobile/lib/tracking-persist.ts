// Pure part of the in-progress session store (JAN-63/65): the snapshot that survives an app
// kill, its (de)serialisation and the restore rule. File access is in tracking-store.ts.
// No React Native imports (unit-tested with node --test).

import type { GaitWindow } from "./gait/types.ts";
import { addFix, isPaused, pause, type RawFix, type TrackMode, type TrackState } from "./tracking.ts";

export const SNAPSHOT_VERSION = 1;

export type Snapshot = {
  v: typeof SNAPSHOT_VERSION;
  horse: string;
  activity: string;
  mode: TrackMode;
  exerciseId?: string;
  targetMinutes?: number;
  state: TrackState;
  /** Gait windows finished so far (the live GaitStream is not persisted). */
  windows: GaitWindow[];
  /** Indexes of the checked exercise steps. */
  checked: number[];
  /** Epoch ms of the last write. */
  savedAt: number;
};

export function serializeSnapshot(s: Snapshot): string {
  return JSON.stringify(s);
}

const isObj = (v: unknown): v is Record<string, unknown> => typeof v === "object" && v !== null && !Array.isArray(v);

/** Parses a stored snapshot; null when it is missing, from another version or malformed. */
export function parseSnapshot(raw: string | null | undefined): Snapshot | null {
  if (!raw) return null;
  try {
    const s: unknown = JSON.parse(raw);
    if (!isObj(s) || s.v !== SNAPSHOT_VERSION) return null;
    if (typeof s.horse !== "string" || typeof s.activity !== "string") return null;
    if (s.mode !== "gps" && s.mode !== "indoor") return null;
    if (typeof s.savedAt !== "number" || !Array.isArray(s.windows) || !Array.isArray(s.checked)) return null;
    const st = s.state;
    if (!isObj(st) || typeof st.startedAt !== "number" || st.mode !== s.mode) return null;
    if (!Array.isArray(st.intervals) || !Array.isArray(st.fixes) || !Array.isArray(st.reins) || !Array.isArray(st.corrections)) {
      return null;
    }
    if (typeof st.distanceM !== "number") return null;
    return s as unknown as Snapshot;
  } catch {
    return null;
  }
}

/**
 * State after an app kill: fixes that arrived while the app was gone (background task) are
 * added first, then the session is paused at the last sign of life, so the time the phone
 * spent dead does not count and the rider resumes on purpose.
 */
export function restoreSnapshot(snapshot: Snapshot, pending: readonly RawFix[] = []): TrackState {
  let state = snapshot.state;
  let lastAlive = snapshot.savedAt;
  if (!isPaused(state)) {
    for (const fix of pending) {
      const next = addFix(state, fix);
      if (next !== state) lastAlive = Math.max(lastAlive, fix.t);
      state = next;
    }
    state = pause(state, lastAlive);
  }
  return state;
}

/** Short German line for "there is still a session running". */
export function snapshotSummary(s: Snapshot, activityLabel: string): string {
  const minutes = Math.max(1, Math.round((s.savedAt - s.state.startedAt) / 60000));
  return `${activityLabel}, seit ${minutes} Min. unterwegs`;
}
