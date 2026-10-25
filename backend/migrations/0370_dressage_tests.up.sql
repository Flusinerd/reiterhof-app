-- Practice tests and freestyles (docs/domains/dressage-tests.md): generated plans of a horse and the
-- requirement sets of freestyle classes that the stable's members enter themselves. The app ships no
-- required movements and no time windows of an official test (copyright of the FNverlag).

CREATE TABLE dressage_tests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id    uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id     uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    created_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    kind         text NOT NULL CHECK (kind IN ('practice', 'freestyle')),
    class        text NOT NULL,
    arena        text NOT NULL CHECK (arena IN ('20x40', '20x60')),
    title        text NOT NULL,
    focus_key    text,
    seed         bigint NOT NULL,
    -- the requirement set a freestyle was made for (a copy: later edits of the set change nothing)
    requirements jsonb,
    plan         jsonb NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX dressage_tests_horse_idx ON dressage_tests (horse_id, created_at);

-- Required movements and time window of a freestyle class as the user copies them from their own
-- Aufgabenheft: movements is [{"key", "count", "rein", "min_meters"}].
CREATE TABLE freestyle_requirements (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id   uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    class       text NOT NULL,
    label       text NOT NULL,
    movements   jsonb NOT NULL DEFAULT '[]',
    min_seconds integer CHECK (min_seconds IS NULL OR min_seconds > 0),
    max_seconds integer CHECK (max_seconds IS NULL OR max_seconds > 0),
    created_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX freestyle_requirements_stable_idx ON freestyle_requirements (stable_id, class);
