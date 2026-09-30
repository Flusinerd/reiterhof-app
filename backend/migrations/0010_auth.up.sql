-- Authentication: magic-link tokens, sessions, social identities, stable invites.
-- Only SHA-256 hashes of tokens are stored, never the tokens themselves.

-- A person may sign up before joining a stable. Domain handlers require a
-- stable through auth.RequireStable, so domain tables never see stable-less users.
ALTER TABLE users ALTER COLUMN stable_id DROP NOT NULL;

CREATE TABLE login_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash bytea NOT NULL UNIQUE,
    email      text NOT NULL CHECK (email = lower(email)),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX login_tokens_expires_idx ON login_tokens (expires_at);

CREATE TABLE auth_sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash   bytea NOT NULL UNIQUE,
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    user_agent   text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);
CREATE INDEX auth_sessions_user_idx ON auth_sessions (user_id);
CREATE INDEX auth_sessions_expires_idx ON auth_sessions (expires_at);

CREATE TABLE auth_identities (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   text NOT NULL CHECK (provider IN ('google', 'apple')),
    subject    text NOT NULL,
    email      text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, subject)
);
CREATE INDEX auth_identities_user_idx ON auth_identities (user_id);

CREATE TABLE stable_invites (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    stable_id  uuid NOT NULL REFERENCES stables (id) ON DELETE CASCADE,
    code       text NOT NULL UNIQUE CHECK (code = upper(code)),
    created_by uuid REFERENCES users (id) ON DELETE SET NULL,
    expires_at timestamptz,
    max_uses   integer CHECK (max_uses IS NULL OR max_uses > 0),
    uses       integer NOT NULL DEFAULT 0 CHECK (uses >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX stable_invites_stable_idx ON stable_invites (stable_id);
