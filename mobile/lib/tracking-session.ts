import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AppState } from "react-native";

import { GaitStream, gaitShares } from "./gait";
import type { Gait, GaitWindow } from "./gait";
import { secondsToMinutes } from "./training";
import {
  activeMs,
  addCorrection,
  addFix,
  apiDistance,
  apiGaitShares,
  applyCorrections,
  buildApiTrack,
  buildApiWindows,
  changeRein,
  createState,
  currentRein,
  currentReinMinutes,
  effectiveGait,
  gaitPoints,
  isPaused,
  pause,
  pointShares,
  reinSegments,
  resume,
  stats,
  toggleStep,
  type GaitPoint,
  type Rein,
  type ReinSegment,
  type Stats,
  type TrackMode,
  type TrackState,
} from "./tracking";
import { setFixListener, startTracking, stopTracking, type StartResult } from "./tracking-location";
import {
  isAccelerometerAvailable,
  keepScreenAwake,
  releaseScreenAwake,
  requestMotionAccess,
  subscribeAccelerometer,
} from "./tracking-sensors";
import { restoreSnapshot, SNAPSHOT_VERSION, type Snapshot } from "./tracking-persist";
import { clearSnapshot, loadSnapshot, saveFinishExtras, saveSnapshot, takePendingFixes } from "./tracking-store";

const KEEP_AWAKE_TAG = "reiterhof-tracking";
const SNAPSHOT_INTERVAL_MS = 15_000;

export type TrackingOptions = {
  mode: TrackMode;
  horse: string;
  activity: string;
  exerciseId?: string;
  targetMinutes?: number;
  /** A restored session (see `useStoredSession`); it comes back paused. */
  initial?: Snapshot | null;
};

/** What the finish screen needs (router params) plus the parts that are too big for them. */
export type FinishPayload = {
  params: Record<string, string>;
};

export type TrackingSession = {
  state: TrackState;
  stats: Stats;
  paused: boolean;
  /** Current gait: the corrected label of the newest window, or null before the first window. */
  gait: Gait | null;
  points: GaitPoint[];
  windows: GaitWindow[];
  rein: Rein | null;
  reinMinutes: number;
  checked: number[];
  sensorAvailable: boolean | null;
  /** Start location updates (GPS mode) and the sensor. Resolves with the failure reason, if any. */
  begin: () => Promise<StartResult>;
  pause: () => void;
  resume: () => Promise<StartResult | null>;
  correct: (gait: Gait) => void;
  changeRein: () => void;
  toggleStep: (index: number) => void;
  /** Ends the session, hands the big parts to the store and returns the finish screen params. */
  finish: () => Promise<FinishPayload>;
  discard: () => Promise<void>;
};

/**
 * Runs one tracked session: accelerometer -> GaitStream, GPS fixes -> state, a one-second
 * tick for the clock, snapshots for crash safety, and the finish payload. All rules are in
 * lib/tracking.ts; this hook only wires the sensors to them. Untested on real devices.
 */
export function useTrackingSession(options: TrackingOptions): TrackingSession {
  const { mode, horse, activity, exerciseId, targetMinutes, initial } = options;

  const [state, setStateValue] = useState<TrackState>(() =>
    initial ? restoreSnapshot(initial) : createState(mode, Date.now()),
  );
  const stateRef = useRef(state);
  const setState = useCallback((fn: (s: TrackState) => TrackState) => {
    const next = fn(stateRef.current);
    if (next !== stateRef.current) {
      stateRef.current = next;
      setStateValue(next);
    }
  }, []);

  const [now, setNow] = useState(() => Date.now());
  const [checked, setChecked] = useState<number[]>(initial?.checked ?? []);
  const checkedRef = useRef(checked);
  const [sensorAvailable, setSensorAvailable] = useState<boolean | null>(null);
  const [windowCount, setWindowCount] = useState(0);

  const archived = useRef<GaitWindow[]>(initial?.windows ?? []);
  const stream = useRef(new GaitStream());
  const sensorSub = useRef<{ remove: () => void } | null>(null);
  const begun = useRef(false);

  const allWindows = useCallback(() => [...archived.current, ...stream.current.windows], []);

  // --- accelerometer ---------------------------------------------------------------------------
  // `motion` is the pending motion permission. On web (iOS Safari) it has to be requested inside the
  // tap that starts the session, before any other await, so `begin` and `resume` pass it in.
  const startSensor = useCallback(async (motion: Promise<boolean> = requestMotionAccess()) => {
    sensorSub.current?.remove();
    sensorSub.current = null;
    try {
      const available = (await motion) && (await isAccelerometerAvailable());
      setSensorAvailable(available);
      if (!available) return;
      sensorSub.current = subscribeAccelerometer(({ x, y, z }) => {
        if (isPaused(stateRef.current)) return;
        const done = stream.current.push({ t: Date.now(), x, y, z });
        if (done.length > 0) setWindowCount(archived.current.length + stream.current.windows.length);
      }, 20); // 50 Hz
    } catch {
      setSensorAvailable(false);
    }
  }, []);

  const stopSensor = useCallback(() => {
    sensorSub.current?.remove();
    sensorSub.current = null;
  }, []);

  /** After a pause the stream restarts: its windows move to the archive. */
  const restartStream = useCallback(() => {
    archived.current = allWindows();
    stream.current = new GaitStream();
  }, [allWindows]);

  // --- GPS fixes --------------------------------------------------------------------------------
  useEffect(() => {
    if (mode !== "gps") return;
    setFixListener((fixes) => {
      for (const fix of fixes) {
        if (fix.speed != null && fix.speed >= 0) stream.current.pushGps({ t: fix.t, speedKmh: fix.speed * 3.6 });
        setState((s) => addFix(s, fix));
      }
    });
    return () => setFixListener(null);
  }, [mode, setState]);

  // --- clock, keep awake, snapshots ------------------------------------------------------------------
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    void keepScreenAwake(KEEP_AWAKE_TAG).catch(() => undefined);
    return () => {
      void releaseScreenAwake(KEEP_AWAKE_TAG).catch(() => undefined);
    };
  }, []);

  const persist = useCallback(() => {
    if (!begun.current) return;
    void saveSnapshot({
      v: SNAPSHOT_VERSION,
      horse,
      activity,
      mode,
      ...(exerciseId ? { exerciseId } : {}),
      ...(targetMinutes ? { targetMinutes } : {}),
      state: stateRef.current,
      windows: allWindows(),
      checked: checkedRef.current,
      savedAt: Date.now(),
    });
  }, [activity, allWindows, exerciseId, horse, mode, targetMinutes]);

  useEffect(() => {
    const id = setInterval(persist, SNAPSHOT_INTERVAL_MS);
    const sub = AppState.addEventListener("change", (status) => {
      if (status !== "active") persist();
    });
    return () => {
      clearInterval(id);
      sub.remove();
    };
  }, [persist]);

  // A restored session starts paused and needs the sensor only after "Fortsetzen".
  useEffect(() => {
    if (initial) begun.current = true;
    return () => {
      stopSensor();
      void stopTracking();
    };
  }, [initial, stopSensor]);

  // --- actions ------------------------------------------------------------------------------------------
  const begin = useCallback(async (): Promise<StartResult> => {
    const motion = requestMotionAccess(); // first, while the tap that started the session is still fresh
    let result: StartResult = { ok: true, background: false };
    if (mode === "gps") result = await startTracking();
    if (result.ok) {
      // The clock starts now, not when the screen opened.
      if (!initial) {
        setState(() => createState(mode, Date.now()));
        archived.current = [];
        stream.current = new GaitStream();
      }
      begun.current = true;
      await startSensor(motion);
      persist();
    }
    return result;
  }, [initial, mode, persist, setState, startSensor]);

  const doPause = useCallback(() => {
    setState((s) => pause(s, Date.now()));
    stopSensor();
    if (mode === "gps") void stopTracking();
    persist();
  }, [mode, persist, setState, stopSensor]);

  const doResume = useCallback(async (): Promise<StartResult | null> => {
    const motion = requestMotionAccess();
    let result: StartResult | null = null;
    if (mode === "gps") {
      result = await startTracking();
      if (!result.ok) return result;
    }
    restartStream();
    setState((s) => resume(s, Date.now()));
    await startSensor(motion);
    begun.current = true;
    persist();
    return result;
  }, [mode, persist, restartStream, setState, startSensor]);

  const windows = useMemo(
    () => applyCorrections(allWindows(), state.corrections),
    // windowCount changes whenever the stream finished a window
    [allWindows, state.corrections, windowCount],
  );
  const lastWindow = windows[windows.length - 1];
  const gait = lastWindow ? effectiveGait(lastWindow) : null;

  const correct = useCallback(
    (right: Gait) => {
      const list = allWindows();
      const last = list[list.length - 1];
      if (!last) return;
      setState((s) => addCorrection(s, Date.now(), last.gait, right));
      persist();
    },
    [allWindows, persist, setState],
  );

  const doChangeRein = useCallback(() => {
    setState((s) => changeRein(s, Date.now()));
    persist();
  }, [persist, setState]);

  const doToggleStep = useCallback(
    (index: number) => {
      setChecked((c) => {
        const next = toggleStep(c, index);
        checkedRef.current = next;
        return next;
      });
    },
    [],
  );

  const finish = useCallback(async (): Promise<FinishPayload> => {
    begun.current = false; // no more snapshots after this point
    const end = Date.now();
    setState((s) => pause(s, end));
    const final = pause(stateRef.current, end);
    stopSensor();
    setFixListener(null);
    await stopTracking();

    const done = applyCorrections(allWindows(), final.corrections);
    const seconds = activeMs(final, end) / 1000;
    const startedAt = new Date(final.startedAt).toISOString();
    const params: Record<string, string> = {
      horse,
      activity,
      minutes: String(secondsToMinutes(seconds)),
      started_at: startedAt,
    };
    if (exerciseId) params.exercise = exerciseId;

    const points = gaitPoints(final, done);
    let shares: Record<string, number> = {};
    if (done.length > 0) {
      const g = gaitShares(done);
      shares = apiGaitShares({
        halt: g.shares.halt.percent,
        walk: g.shares.walk.percent,
        trot: g.shares.trot.percent,
        canter: g.shares.canter.percent,
      });
    } else if (points.length > 1) {
      shares = apiGaitShares(pointShares(points));
    }
    if (Object.keys(shares).length > 0) params.gait = JSON.stringify(shares);

    if (mode === "indoor") {
      const segs: ReinSegment[] = reinSegments(final, end);
      if (segs.length > 0) params.rein = JSON.stringify(segs);
    } else {
      const distance = apiDistance(final.distanceM);
      if (distance) params.distance_m = String(distance);
    }

    const track = mode === "gps" ? buildApiTrack(final, done) : [];
    const records = done.length > 0 ? buildApiWindows(done, final.startedAt) : [];
    await saveFinishExtras({
      startedAt,
      ...(track.length > 0 ? { track } : {}),
      ...(records.length > 0 ? { gait_windows: records } : {}),
    });
    await clearSnapshot();
    return { params };
  }, [activity, allWindows, exerciseId, horse, mode, setState, stopSensor]);

  const discard = useCallback(async () => {
    begun.current = false;
    stopSensor();
    setFixListener(null);
    await stopTracking();
    await clearSnapshot();
  }, [stopSensor]);

  const currentStats = useMemo(() => stats(state, now), [state, now]);
  const points = useMemo(() => (mode === "gps" ? gaitPoints(state, windows) : []), [mode, state, windows]);

  return {
    state,
    stats: currentStats,
    paused: isPaused(state),
    gait,
    points,
    windows,
    rein: currentRein(state),
    reinMinutes: currentReinMinutes(state, now),
    checked,
    sensorAvailable,
    begin,
    pause: doPause,
    resume: doResume,
    correct,
    changeRein: doChangeRein,
    toggleStep: doToggleStep,
    finish,
    discard,
  };
}

/**
 * Loads the stored in-progress session once (null when there is none). `foldPending` also
 * takes the fixes the background task collected and adds them; only the screen that resumes
 * the session may do that, because it empties the buffer.
 */
export function useStoredSession(enabled = true, foldPending = true): { loading: boolean; snapshot: Snapshot | null } {
  const [value, setValue] = useState<{ loading: boolean; snapshot: Snapshot | null }>({ loading: enabled, snapshot: null });
  useEffect(() => {
    if (!enabled) return;
    let cancelled = false;
    void (async () => {
      const snapshot = await loadSnapshot();
      if (snapshot && foldPending) {
        const pending = await takePendingFixes();
        if (pending.length > 0) snapshot.state = restoreSnapshot(snapshot, pending);
      }
      if (!cancelled) setValue({ loading: false, snapshot });
    })();
    return () => {
      cancelled = true;
    };
  }, [enabled, foldPending]);
  return value;
}
