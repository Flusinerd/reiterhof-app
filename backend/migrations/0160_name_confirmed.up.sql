-- The user has chosen their display name. A magic-link sign-up only knows the email, so the
-- account starts with the part before the "@" as name and name_confirmed = false; the app then
-- asks for the name once (PATCH /api/v1/me with a name confirms it). Operator- and
-- provider-created users (admin CLI, Google with a name) have a real name, hence DEFAULT true.
-- Existing accounts whose name is still that email fallback are asked as well.
ALTER TABLE users ADD COLUMN name_confirmed boolean NOT NULL DEFAULT true;
UPDATE users SET name_confirmed = false WHERE name = split_part(email, '@', 1);
