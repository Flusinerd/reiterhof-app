-- Privacy (JAN-19): consents and the marker for deleted accounts.
--
-- One row per user and consent kind: granting sets version and granted_at and clears
-- revoked_at, revoking sets revoked_at. A kind without a row was never granted.
-- stable_id is nullable because a person may consent before joining a stable.
CREATE TABLE consents (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    stable_id  uuid REFERENCES stables (id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('location_geofence', 'location_tracking', 'presence_sharing', 'photos', 'push')),
    version    text NOT NULL,
    granted_at timestamptz NOT NULL,
    revoked_at timestamptz,
    UNIQUE (user_id, kind)
);
CREATE INDEX consents_stable_idx ON consents (stable_id);

-- Account deletion anonymises the users row instead of removing it (other members'
-- records such as requests and training sessions reference it); see docs/domains/privacy.md.
ALTER TABLE users ADD COLUMN deleted_at timestamptz;
