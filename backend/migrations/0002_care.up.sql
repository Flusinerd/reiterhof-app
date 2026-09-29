-- Care tables: blankets, weather, requests, health, observations, documents.

CREATE TABLE blankets (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    name       text NOT NULL,
    fill_g     integer NOT NULL DEFAULT 0 CHECK (fill_g >= 0),
    color      text,
    location   text,
    photo_path text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX blankets_horse_idx ON blankets (stable_id, horse_id);

-- Ordered rules per horse; the first matching rule (lowest position) wins.
-- NULL conditions match anything; blanket_id NULL means "no blanket".
CREATE TABLE blanket_rules (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    position   integer NOT NULL,
    temp_min   numeric,
    temp_max   numeric,
    rain       boolean,
    blanket_id uuid REFERENCES blankets (id) ON DELETE SET NULL,
    note       text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (horse_id, position)
);
CREATE INDEX blanket_rules_stable_idx ON blanket_rules (stable_id);

CREATE TABLE blanket_states (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id    uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id     uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    day          date NOT NULL,
    action       text NOT NULL CHECK (action IN ('covered', 'uncovered', 'checked')),
    covered_with uuid REFERENCES blankets (id) ON DELETE SET NULL,
    changed_at   timestamptz NOT NULL DEFAULT now(),
    changed_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX blanket_states_horse_day_idx ON blanket_states (stable_id, horse_id, day DESC, changed_at DESC);

CREATE TABLE weather_snapshots (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id        uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    fetched_at       timestamptz NOT NULL,
    valid_for        date NOT NULL,
    night_min_c      numeric,
    rain_probability integer CHECK (rain_probability BETWEEN 0 AND 100),
    rain_mm          numeric,
    wind_kmh         numeric,
    will_rain        boolean NOT NULL DEFAULT false,
    raw              jsonb,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX weather_snapshots_valid_idx ON weather_snapshots (stable_id, valid_for, fetched_at DESC);

CREATE TABLE requests (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id        uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    type             text NOT NULL CHECK (type IN ('blanket', 'show_helper', 'ride_share', 'exercise', 'feed_or_turnout', 'appointment_companion', 'other')),
    horse_id         uuid REFERENCES horses (id) ON DELETE SET NULL,
    created_by       uuid NOT NULL REFERENCES users (id),
    date             date NOT NULL,
    date_end         date,
    time_from        time,
    time_to          time,
    location         text,
    description      text,
    tasks            jsonb NOT NULL DEFAULT '[]',
    helpers_needed   integer NOT NULL DEFAULT 1 CHECK (helpers_needed > 0),
    status           text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'assigned', 'done', 'cancelled')),
    recurring_rule   text,
    remind_helper_at timestamptz,
    payload          jsonb NOT NULL DEFAULT '{}',
    created_at       timestamptz NOT NULL DEFAULT now(),
    CHECK (date_end IS NULL OR date_end >= date)
);
CREATE INDEX requests_stable_date_idx ON requests (stable_id, status, date);
CREATE INDEX requests_horse_idx ON requests (horse_id);
CREATE INDEX requests_created_by_idx ON requests (created_by);

CREATE TABLE request_assignees (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    request_id uuid NOT NULL REFERENCES requests (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    thanked    boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (request_id, user_id)
);
CREATE INDEX request_assignees_user_idx ON request_assignees (user_id);
CREATE INDEX request_assignees_stable_idx ON request_assignees (stable_id);

CREATE TABLE observations (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id   uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id    uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    reported_by uuid NOT NULL REFERENCES users (id),
    category    text,
    body_part   text,
    description text,
    media       jsonb NOT NULL DEFAULT '[]',
    urgency     text NOT NULL DEFAULT 'info' CHECK (urgency IN ('info', 'check', 'urgent')),
    status      text NOT NULL DEFAULT 'watch' CHECK (status IN ('watch', 'done')),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX observations_horse_idx ON observations (stable_id, horse_id, created_at DESC);
CREATE INDEX observations_reported_by_idx ON observations (reported_by);

CREATE TABLE health_items (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id     uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id      uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    kind          text NOT NULL CHECK (kind IN ('vaccination', 'farrier', 'deworming', 'dentist', 'physio', 'medication', 'vet')),
    label         text NOT NULL,
    due_date      date,
    interval_days integer CHECK (interval_days > 0),
    note          text,
    daily_time    time,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX health_items_horse_idx ON health_items (stable_id, horse_id);
CREATE INDEX health_items_due_idx ON health_items (stable_id, due_date);

CREATE TABLE emergency_contacts (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id   uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    label      text NOT NULL,
    name       text NOT NULL,
    phone      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX emergency_contacts_horse_idx ON emergency_contacts (stable_id, horse_id);

CREATE TABLE horse_documents (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id   uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    horse_id    uuid NOT NULL REFERENCES horses (id) ON DELETE CASCADE,
    kind        text NOT NULL,
    title       text NOT NULL,
    file_path   text NOT NULL,
    uploaded_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX horse_documents_horse_idx ON horse_documents (stable_id, horse_id);
