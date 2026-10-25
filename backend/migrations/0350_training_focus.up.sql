-- Training planning (M9, docs/domains/training-planning.md): focuses, unit plans, stable
-- equipment and the level goal. The focus catalog is static data in Go (training.Focuses), so the
-- keys are plain text without a foreign key.

-- The owner's focuses of a horse: [{"key": "takt", "until": "2026-10-28"}], at most 3, the order is
-- the priority. cycle_start is the Monday of the first week of the four-week training blocks.
-- goal_level and goal_until: the level class the horse is trained towards (NULL = the class above
-- the level of the profile) and an optional date for it.
ALTER TABLE training_profiles
    ADD COLUMN focuses     jsonb NOT NULL DEFAULT '[]' CHECK (jsonb_typeof(focuses) = 'array'),
    ADD COLUMN cycle_start date,
    ADD COLUMN goal_level  text CHECK (goal_level IN ('intermediate', 'advanced')),
    ADD COLUMN goal_until  date;

-- The focuses a person picked for riding a horse (the owner and riders alike), at most 2.
CREATE TABLE rider_focuses (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    keys       text[] NOT NULL DEFAULT '{}' CHECK (cardinality(keys) <= 2),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (horse_id, user_id)
);
CREATE INDEX rider_focuses_stable_idx ON rider_focuses (stable_id);

-- A planned day: its focus, the unit plan {"warmup": {"exercise_id": ..., "minutes": 15}, "main": {...},
-- "cooldown": {...}} (exercise_id stays the main exercise for older clients) and whether set-up is
-- possible that day (true also in a shared location, false never, NULL = the location's default).
ALTER TABLE week_slots
    ADD COLUMN focus_key    text,
    ADD COLUMN unit_plan    jsonb,
    ADD COLUMN equipment_ok boolean;

-- What a session trained; drives the focus rotation.
ALTER TABLE sessions ADD COLUMN focus_key text;

-- Facility equipment per location: {"hall": {"equipment": ["poles"], "shared": false}, "arena": ...,
-- "lunge": ..., "jumping": ..., "groundwork": ...}. A missing location has its default (see
-- training.DefaultLocation); '{}' means the stable has not saved it yet.
ALTER TABLE stables
    ADD COLUMN training_equipment jsonb NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(training_equipment) = 'object');
