-- Blankets (M3): the last-person, check-out and weather-change reminders claim each
-- (user, kind, source, night) by inserting a row into reminders; the unique index makes
-- the claim atomic, so a job that runs twice (or two API instances) never notifies twice.
--   blanket_night    source_id = stable, one reminder per user and night
--   blanket_checkout source_id = stable, one check-out reminder per user and night
--   blanket_weather  source_id = horse,  one weather change per user, horse and night
-- due_at is the start of the night (12:00 stable-local of the blanket day).
CREATE UNIQUE INDEX reminders_blankets_once
    ON reminders (user_id, kind, source_table, source_id, due_at)
    WHERE source_table IN ('blanket_night', 'blanket_checkout', 'blanket_weather');
