-- Training tables: profiles, exercise library, sessions, week plan, rehab.

CREATE TABLE training_profiles (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id          uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id           uuid NOT NULL UNIQUE REFERENCES horses (id) ON DELETE CASCADE,
    discipline         text,
    level              text,
    allowed_activities jsonb NOT NULL DEFAULT '[]',
    shows              jsonb NOT NULL DEFAULT '[]',
    season_end         date,
    rhythm             jsonb NOT NULL DEFAULT '{}',
    rb_rules           jsonb NOT NULL DEFAULT '{}',
    status             text NOT NULL DEFAULT 'fit' CHECK (status IN ('fit', 'reha', 'pause')),
    created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX training_profiles_stable_idx ON training_profiles (stable_id);

-- stable_id NULL marks a global library row shared by all stables.
CREATE TABLE exercises (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id        uuid REFERENCES stables (id) ON DELETE CASCADE,
    discipline       text,
    level            text,
    goal_tags        text[] NOT NULL DEFAULT '{}',
    title            text NOT NULL,
    steps            jsonb NOT NULL DEFAULT '[]',
    next_exercise_id uuid REFERENCES exercises (id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX exercises_stable_idx ON exercises (stable_id, discipline, level);
CREATE INDEX exercises_goal_tags_idx ON exercises USING gin (goal_tags);

CREATE TABLE sessions (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id        uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id         uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    user_id          uuid NOT NULL REFERENCES users (id),
    activity         text NOT NULL,
    started_at       timestamptz NOT NULL,
    duration_min     integer CHECK (duration_min >= 0),
    gait_shares      jsonb NOT NULL DEFAULT '{}',
    rein_changes     jsonb NOT NULL DEFAULT '[]',
    feel             text,
    focus_rating     integer,
    note             text,
    visible_to_rider boolean NOT NULL DEFAULT true,
    load_score       numeric,
    track            jsonb,
    distance_m       integer CHECK (distance_m >= 0),
    source           text NOT NULL DEFAULT 'quick' CHECK (source IN ('quick', 'tracked')),
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX sessions_horse_started_idx ON sessions (stable_id, horse_id, started_at DESC);
CREATE INDEX sessions_user_started_idx ON sessions (user_id, started_at DESC);

CREATE TABLE week_slots (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    day        date NOT NULL,
    user_id    uuid REFERENCES users (id) ON DELETE SET NULL,
    activity   text,
    status     text NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'done', 'rest')),
    note       text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (horse_id, day)
);
CREATE INDEX week_slots_stable_day_idx ON week_slots (stable_id, day);
CREATE INDEX week_slots_user_idx ON week_slots (user_id);

CREATE TABLE reha_plans (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id      uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id       uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    diagnosis      text NOT NULL,
    vet            text,
    start_date     date NOT NULL,
    phases         jsonb NOT NULL DEFAULT '[]',
    checkup_date   date,
    active         boolean NOT NULL DEFAULT true,
    observation_id uuid REFERENCES observations (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reha_plans_horse_idx ON reha_plans (stable_id, horse_id);
-- At most one active plan per horse.
CREATE UNIQUE INDEX reha_plans_one_active_per_horse ON reha_plans (horse_id) WHERE active;

CREATE TABLE reha_days (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id    uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    reha_plan_id uuid NOT NULL REFERENCES reha_plans (id) ON DELETE CASCADE,
    day          date NOT NULL,
    done_by      uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (reha_plan_id, day)
);
CREATE INDEX reha_days_stable_idx ON reha_days (stable_id);
