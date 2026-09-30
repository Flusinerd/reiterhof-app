import { authed, type PresenceVisibility } from "../api";

// Mirrors docs/domains/presence.md. Times are RFC 3339 strings.

export type Visit = {
  id: string;
  arrived_at: string;
  left_at: string | null;
  source: "manual" | "geofence";
};

export type PresenceMe = {
  open_visit: Visit | null;
  last_visit: Visit | null;
  visibility: PresenceVisibility;
};

/** Someone at the stable right now. `since` is null when the person shows only the day. */
export type PresenceHere = {
  user_id: string;
  name: string;
  avatar_color: string | null;
  since: string | null;
};

/** Last visit of someone who is not here now ("zuletzt gesehen"). */
export type PresenceRecent = {
  user_id: string;
  name: string;
  avatar_color: string | null;
  /** YYYY-MM-DD in the stable's time zone. */
  last_seen_date: string;
  /** null when the person shows only the day. */
  last_seen_at: string | null;
  today: boolean;
  /** 0-23; typical arrival hour, only for people with visibility "all" and enough visits. */
  usual_arrival_hour: number | null;
};

export type PresenceOverview = {
  me: PresenceMe;
  here: PresenceHere[];
  recent: PresenceRecent[];
};

export const PRESENCE_KEY = ["presence"] as const;
export const PRESENCE_EVENT = "presence.changed";

export const presenceApi = {
  overview: () => authed.get<PresenceOverview>("/api/v1/presence"),
  /** Idempotent: an already open visit is returned. */
  checkIn: (source: "manual" | "geofence" = "manual") =>
    authed.post<{ visit: Visit | null }>("/api/v1/presence/check-in", { source }),
  /** No-op (`visit: null`) when nobody is checked in. */
  checkOut: () => authed.post<{ visit: Visit | null }>("/api/v1/presence/check-out"),
};
