-- Age confirmation and parental consent (JAN-86, Art. 8 GDPR).
-- Location, photos, push and presence sharing rest on consent, which a person under 16 cannot
-- give alone in Germany. Every account states once whether the person is 16 or older
-- (age_confirmed_at) or names a parent (parent_email) who confirms through an e-mailed link
-- (parental_consent_at). Until one of the two is set, consents cannot be granted and no stable
-- can be joined. No birth date is stored.
ALTER TABLE users
    ADD COLUMN age_confirmed_at    timestamptz,
    ADD COLUMN parent_email        text CHECK (parent_email IS NULL OR parent_email = lower(parent_email)),
    ADD COLUMN parental_consent_at timestamptz;

-- One row per mail to a parent; only a hash of the link token is stored. Using the link sets
-- used_at. Expired and used rows are pruned by the retention job.
CREATE TABLE parental_consent_tokens (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   bytea NOT NULL UNIQUE,
    parent_email text NOT NULL CHECK (parent_email = lower(parent_email)),
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX parental_consent_tokens_user_idx ON parental_consent_tokens (user_id);
