-- Health reminders (JAN-12): the reminder job claims each (user, kind, health item,
-- scheduled time) by inserting a row into reminders; the unique index makes the claim
-- atomic, so a job that runs twice (or two API instances) never notifies twice.
CREATE UNIQUE INDEX reminders_health_once
    ON reminders (user_id, kind, source_id, due_at)
    WHERE source_table = 'health_items';
