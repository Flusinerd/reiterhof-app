-- Reha plans (JAN-66): abort criteria, end time and author of a plan, and the atomic claim of
-- the vet checkup reminders (the job inserts a reminders row per user, plan and scheduled
-- time; the unique index makes a run that happens twice send nothing twice).
ALTER TABLE reha_plans
    ADD COLUMN abort_criteria text,
    ADD COLUMN ended_at       timestamptz,
    ADD COLUMN created_by     uuid REFERENCES users (id) ON DELETE SET NULL;

CREATE UNIQUE INDEX reminders_reha_once
    ON reminders (user_id, kind, source_id, due_at)
    WHERE source_table = 'reha_plans';
