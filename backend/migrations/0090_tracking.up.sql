-- M7 Tracking: raw gait windows of tracked sessions, kept for improving the gait model.
-- The simplified GPS track lives in sessions.track (jsonb, created in 0003); it is bounded by
-- the API (at most 5000 points). One row per session, at most 3000 windows (about 100 minutes
-- of riding at a 2 s hop, roughly 100 bytes each).

CREATE TABLE gait_windows (
    session_id   uuid PRIMARY KEY REFERENCES sessions (id) ON DELETE CASCADE,
    stable_id    uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    window_count integer NOT NULL CHECK (window_count BETWEEN 1 AND 3000),
    windows      jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX gait_windows_stable_idx ON gait_windows (stable_id);
