-- Official weather warnings of the DWD for the stable's location (Wetterwarnungen).
-- The warning job replaces the rows of a stable on every fetch: warnings that are no
-- longer active are deleted, new ones inserted, known ones (same DWD identifier) updated.
CREATE TABLE weather_warnings (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id   uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    -- CAP identifier of the DWD message; an update of a warning gets a new identifier.
    dwd_id      text NOT NULL,
    event       text NOT NULL,
    severity    text NOT NULL CHECK (severity IN ('minor', 'moderate', 'severe', 'extreme')),
    headline    text NOT NULL,
    description text NOT NULL DEFAULT '',
    instruction text NOT NULL DEFAULT '',
    onset       timestamptz NOT NULL,
    expires     timestamptz,
    fetched_at  timestamptz NOT NULL,
    -- notified: the push for this warning is done, or not needed (not relevant enough, or
    -- the same event was already pushed at this severity before the DWD updated it).
    notified    boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (stable_id, dwd_id)
);

-- The warning push claims (user, warning) once, like the other reminder jobs.
CREATE UNIQUE INDEX reminders_weather_warning_once
    ON reminders (user_id, source_id)
    WHERE source_table = 'weather_warnings';
