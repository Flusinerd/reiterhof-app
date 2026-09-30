import * as FileSystem from "expo-file-system/legacy";
import { Platform } from "react-native";

import { parseSnapshot, serializeSnapshot, type Snapshot } from "./tracking-persist";
import type { ApiTrackPoint, RawFix } from "./tracking";
import type { WindowRecord } from "./gait/export";

/**
 * Local store of the tracker (JAN-63/65). Files in the app's document directory, because a
 * ride easily produces hundreds of kilobytes (the secure store only takes a few kilobytes per
 * value) and AsyncStorage is not a dependency of the project.
 *
 * - `tracking-session.json`: snapshot of the running session, written every few seconds, so
 *   an app kill or crash does not lose it. Cleared when the session is saved or discarded.
 * - `tracking-pending.json`: GPS fixes that the background task received while no screen was
 *   listening (the process was restarted by the OS).
 * - `tracking-finish.json`: track and gait windows on their way to the finish screen, which
 *   posts them with the session. Router params are too small for them.
 *
 * All calls are best effort: on web, or when the disk is not writable, they do nothing.
 */

const native = Platform.OS === "ios" || Platform.OS === "android";
const dir = native ? FileSystem.documentDirectory : null;

const SESSION_FILE = "tracking-session.json";
const PENDING_FILE = "tracking-pending.json";
const FINISH_FILE = "tracking-finish.json";

// Writes to one file must not overlap.
const queues = new Map<string, Promise<unknown>>();
function enqueue<T>(file: string, job: () => Promise<T>): Promise<T | undefined> {
  const prev = queues.get(file) ?? Promise.resolve();
  const next = prev.then(job, job).catch(() => undefined) as Promise<T | undefined>;
  queues.set(file, next);
  return next;
}

async function write(file: string, text: string): Promise<void> {
  if (!dir) return;
  await enqueue(file, () => FileSystem.writeAsStringAsync(dir + file, text));
}

async function read(file: string): Promise<string | null> {
  if (!dir) return null;
  return (
    (await enqueue(file, async () => {
      const info = await FileSystem.getInfoAsync(dir + file);
      return info.exists ? FileSystem.readAsStringAsync(dir + file) : null;
    })) ?? null
  );
}

async function remove(file: string): Promise<void> {
  if (!dir) return;
  await enqueue(file, () => FileSystem.deleteAsync(dir + file, { idempotent: true }));
}

// --- running session -----------------------------------------------------------------------------

export function saveSnapshot(snapshot: Snapshot): Promise<void> {
  return write(SESSION_FILE, serializeSnapshot(snapshot));
}

export async function loadSnapshot(): Promise<Snapshot | null> {
  return parseSnapshot(await read(SESSION_FILE));
}

export async function clearSnapshot(): Promise<void> {
  await remove(SESSION_FILE);
  await remove(PENDING_FILE);
}

// --- fixes of the background task ------------------------------------------------------------------

let pendingMemory: RawFix[] = [];

/** Called by the background task when no screen listens. */
export function appendPendingFixes(fixes: RawFix[]): Promise<void> {
  pendingMemory = [...pendingMemory, ...fixes].slice(-20_000);
  return write(PENDING_FILE, JSON.stringify(pendingMemory));
}

/** Fixes collected while the app was not listening; empties the buffer. */
export async function takePendingFixes(): Promise<RawFix[]> {
  let fixes = pendingMemory;
  if (fixes.length === 0) {
    try {
      const parsed: unknown = JSON.parse((await read(PENDING_FILE)) ?? "[]");
      fixes = Array.isArray(parsed) ? (parsed as RawFix[]) : [];
    } catch {
      fixes = [];
    }
  }
  pendingMemory = [];
  if (fixes.length > 0) await remove(PENDING_FILE);
  return fixes;
}

// --- hand-off to the finish screen --------------------------------------------------------------------

export type FinishExtras = {
  /** `started_at` of the session, the key that ties the extras to the finish params. */
  startedAt: string;
  track?: ApiTrackPoint[];
  gait_windows?: WindowRecord[];
};

let extrasMemory: FinishExtras | null = null;

export function saveFinishExtras(extras: FinishExtras): Promise<void> {
  extrasMemory = extras;
  return write(FINISH_FILE, JSON.stringify(extras));
}

/** Extras for the session that started at `startedAt`, or null (quick log, other session). */
export async function loadFinishExtras(startedAt: string | undefined): Promise<FinishExtras | null> {
  if (!startedAt) return null;
  if (extrasMemory?.startedAt === startedAt) return extrasMemory;
  try {
    const parsed = JSON.parse((await read(FINISH_FILE)) ?? "null") as FinishExtras | null;
    return parsed && parsed.startedAt === startedAt ? parsed : null;
  } catch {
    return null;
  }
}

export async function clearFinishExtras(): Promise<void> {
  extrasMemory = null;
  await remove(FINISH_FILE);
}
