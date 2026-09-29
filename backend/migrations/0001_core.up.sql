-- Core tables: stables, users, horses, presence, reminders, push tokens.
-- Convention: every domain table carries stable_id (FK to stables); the API
-- enforces tenant scoping, Postgres RLS is not used.

CREATE TABLE stables (
    id                          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                        text NOT NULL,
    farm_name                   text,
    city                        text,
    lat                         double precision,
    lng                         double precision,
    timezone                    text NOT NULL DEFAULT 'Europe/Berlin',
    reminder_time               time NOT NULL DEFAULT '20:30',
    geofence_radius_m           integer NOT NULL DEFAULT 150 CHECK (geofence_radius_m > 0),
    ground_condition            text CHECK (ground_condition IN ('dry', 'wet', 'frozen', 'muddy')),
    ground_condition_updated_at timestamptz,
    created_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id           uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    name                text NOT NULL,
    email               text NOT NULL CHECK (email = lower(email)),
    phone               text,
    avatar_color        text,
    presence_visibility text NOT NULL DEFAULT 'all' CHECK (presence_visibility IN ('all', 'only_day', 'hidden')),
    is_admin            boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_key ON users (email);
CREATE INDEX users_stable_idx ON users (stable_id);

CREATE TABLE horses (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id            uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    name                 text NOT NULL,
    box                  text,
    sex                  text CHECK (sex IN ('mare', 'gelding', 'stallion')),
    birth_year           integer,
    breed                text,
    color_key            text,
    weight_kg            integer,
    owner_id             uuid REFERENCES users (id) ON DELETE SET NULL,
    emergency_note       text,
    emergency_medication text,
    permanent_medication text,
    allergies            text,
    insurance            text,
    vet_name             text,
    vet_phone            text,
    helper_note          text,
    created_at           timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX horses_stable_idx ON horses (stable_id, name);
CREATE INDEX horses_owner_idx ON horses (owner_id);

-- Riders allowed on a horse; rules is a positive list of what the rider may do.
CREATE TABLE horse_riders (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    rules      jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (horse_id, user_id)
);
CREATE INDEX horse_riders_user_idx ON horse_riders (user_id);
CREATE INDEX horse_riders_stable_idx ON horse_riders (stable_id);

CREATE TABLE presence (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    arrived_at timestamptz NOT NULL,
    left_at    timestamptz,
    source     text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'geofence')),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (left_at IS NULL OR left_at >= arrived_at)
);
CREATE INDEX presence_stable_arrived_idx ON presence (stable_id, arrived_at DESC);
CREATE INDEX presence_user_arrived_idx ON presence (user_id, arrived_at DESC);
-- At most one open visit per user.
CREATE UNIQUE INDEX presence_one_open_per_user ON presence (user_id) WHERE left_at IS NULL;

CREATE TABLE reminders (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id    uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind         text NOT NULL,
    title        text NOT NULL,
    body         text,
    due_at       timestamptz NOT NULL,
    source_table text,
    source_id    uuid,
    sent_at      timestamptz,
    dismissed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX reminders_user_due_idx ON reminders (user_id, due_at);
CREATE INDEX reminders_pending_idx ON reminders (due_at) WHERE sent_at IS NULL;
CREATE INDEX reminders_stable_idx ON reminders (stable_id);

CREATE TABLE reminder_settings (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind       text NOT NULL,
    enabled    boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, kind)
);
CREATE INDEX reminder_settings_stable_idx ON reminder_settings (stable_id);

CREATE TABLE push_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token      text NOT NULL UNIQUE,
    platform   text NOT NULL CHECK (platform IN ('ios', 'android')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX push_tokens_user_idx ON push_tokens (user_id);
CREATE INDEX push_tokens_stable_idx ON push_tokens (stable_id);
