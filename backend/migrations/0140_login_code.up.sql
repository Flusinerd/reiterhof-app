-- Sign-in with a 6-digit code next to the magic link (JAN-76).
-- The code belongs to the same login attempt (row) as the link token. It is stored only as
-- an HMAC-SHA256 (server-side key, bound to the email); NULL means "no usable code"
-- (superseded by a newer request, locked out after too many wrong attempts, or a row from
-- before this migration). Using the link or the code sets used_at and thereby kills both.
ALTER TABLE login_tokens
    ADD COLUMN code_hash     bytea,
    ADD COLUMN code_attempts smallint NOT NULL DEFAULT 0 CHECK (code_attempts >= 0);
CREATE INDEX login_tokens_code_idx ON login_tokens (email) WHERE code_hash IS NOT NULL AND used_at IS NULL;
