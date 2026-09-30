-- Reminder center (M8, JAN-69, JAN-18): the evening-before training plan claims each
-- (user, evening) by inserting a reminders row (source_table = 'training_plan',
-- source_id = stable, due_at = 19:00 stable-local of the evening); the unique index makes
-- the claim atomic, so a job that runs twice (or two API instances) never notifies twice.
CREATE UNIQUE INDEX reminders_training_plan_once
    ON reminders (user_id, kind, source_id, due_at)
    WHERE source_table = 'training_plan';
