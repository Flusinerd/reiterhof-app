import { useQueryClient, type QueryKey } from "@tanstack/react-query";
import { fetch as streamingFetch } from "expo/fetch";
import { useEffect, useRef } from "react";
import { AppState } from "react-native";

import { API_URL } from "./api";
import { backoffDelay } from "./backoff";
import { createSseParser } from "./sse";
import { getToken } from "./token";

/**
 * Realtime events of the stable (Server-Sent Events, `GET /api/v1/events`).
 *
 * Why `expo/fetch` instead of an EventSource library: React Native's built-in `fetch`
 * cannot stream response bodies, and the common EventSource polyfills either cannot set
 * the Authorization header or are unmaintained. `expo/fetch` ships with the Expo SDK
 * (no extra dependency, no version to pin), streams the body, supports AbortSignal and
 * lets us keep the token in the header instead of the URL. The SSE framing is parsed by
 * the small tested `createSseParser`.
 *
 * One connection is shared by all hooks (reference counted), it is closed while the app
 * is in the background and re-opened with backoff after errors. After every reconnect a
 * synthetic `resync` event is delivered because events may have been missed.
 */

export const RESYNC = "resync";

export type StableEvent = { type: string; data: unknown };
type Listener = (event: StableEvent) => void;

const listeners = new Set<Listener>();
let running = false;
let controller: AbortController | null = null;
let stopLoop: (() => void) | null = null;
let appStateSub: { remove: () => void } | null = null;

function emit(event: StableEvent) {
  for (const l of [...listeners]) {
    try {
      l(event);
    } catch {
      // a broken listener must not stop the others
    }
  }
}

async function readStream(signal: AbortSignal): Promise<void> {
  const token = getToken();
  if (!token) throw new Error("not signed in");
  const res = await streamingFetch(`${API_URL.replace(/\/+$/, "")}/api/v1/events`, {
    headers: { Authorization: `Bearer ${token}`, Accept: "text/event-stream" },
    signal,
  });
  if (!res.ok || !res.body) throw new Error(`events: status ${res.status}`);
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  const parser = createSseParser();
  let announced = false;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) return;
    if (!announced) {
      announced = true;
      connectedOnce();
    }
    for (const msg of parser.push(decoder.decode(value, { stream: true }))) {
      try {
        const parsed = JSON.parse(msg.data) as { type?: string; data?: unknown };
        emit({ type: msg.event, data: parsed.data });
      } catch {
        // malformed payload: ignore
      }
    }
  }
}

let everConnected = false;
let failures = 0;
function connectedOnce() {
  failures = 0;
  if (everConnected) emit({ type: RESYNC, data: null });
  everConnected = true;
}

let generation = 0;

async function loop(gen: number) {
  const alive = () => running && gen === generation;
  while (alive()) {
    controller = new AbortController();
    try {
      await readStream(controller.signal);
    } catch {
      // aborted or failed: retry below unless stopped
    }
    if (!alive()) return;
    const delay = backoffDelay(failures++);
    await new Promise<void>((resolve) => {
      const t = setTimeout(resolve, delay);
      stopLoop = () => {
        clearTimeout(t);
        resolve();
      };
    });
    stopLoop = null;
  }
}

function start() {
  if (running) return;
  running = true;
  failures = 0;
  void loop(++generation);
}

function stop() {
  running = false;
  controller?.abort();
  stopLoop?.();
}

function retain() {
  if (listeners.size === 0) return;
  if (AppState.currentState === "active") start();
  appStateSub ??= AppState.addEventListener("change", (state) => {
    if (state === "active" && listeners.size > 0) {
      // Back in the foreground: events were missed; the reconnect emits a resync.
      start();
    } else if (state !== "active") {
      stop();
    }
  });
}

function release() {
  if (listeners.size > 0) return;
  stop();
  appStateSub?.remove();
  appStateSub = null;
}

function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  retain();
  return () => {
    listeners.delete(listener);
    release();
  };
}

/**
 * Calls `handler` for events of the given types (and for `resync` after a reconnect).
 * The handler may change between renders without reconnecting.
 *
 * @example
 * useStableEvents(["request.changed"], () => refetch());
 */
export function useStableEvents(types: readonly string[], handler: (event: StableEvent) => void): void {
  const handlerRef = useRef(handler);
  handlerRef.current = handler;
  const key = types.join("|");
  useEffect(() => {
    const wanted = new Set(key === "" ? [] : key.split("|"));
    return subscribe((event) => {
      if (event.type === RESYNC || wanted.has(event.type)) handlerRef.current(event);
    });
  }, [key]);
}

/**
 * Invalidates TanStack Query keys when an event arrives (and after a reconnect).
 *
 * @example
 * useInvalidateOnEvents({ "presence.changed": [["presence"]] });
 */
export function useInvalidateOnEvents(map: Record<string, readonly QueryKey[]>): void {
  const queryClient = useQueryClient();
  const mapRef = useRef(map);
  mapRef.current = map;
  const types = Object.keys(map);
  useStableEvents(types, (event) => {
    const keys = event.type === RESYNC ? Object.values(mapRef.current).flat() : (mapRef.current[event.type] ?? []);
    for (const queryKey of keys) void queryClient.invalidateQueries({ queryKey });
  });
}
