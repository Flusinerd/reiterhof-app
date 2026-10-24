-- Health items: the repeat interval gets a unit, so "every 6 months" or "every year" stays on the
-- same calendar day instead of drifting as 182 or 365 days; and every item says on which days
-- before the due date and at what time its reminder goes out (before: fixed 7 and 1 days, 08:00).
ALTER TABLE health_items RENAME COLUMN interval_days TO interval_count;
ALTER TABLE health_items RENAME CONSTRAINT health_items_interval_days_check TO health_items_interval_count_check;
ALTER TABLE health_items
    ADD COLUMN interval_unit text NOT NULL DEFAULT 'day' CHECK (interval_unit IN ('day', 'week', 'month', 'year')),
    ADD COLUMN remind_days   integer[] NOT NULL DEFAULT '{7,1}'
        CHECK (cardinality(remind_days) <= 5 AND 0 <= ALL (remind_days) AND 365 >= ALL (remind_days)),
    ADD COLUMN remind_time   time NOT NULL DEFAULT '08:00';

-- Intervals were entered in days so far: whole years (365 days) become years, whole weeks become weeks.
UPDATE health_items SET interval_count = interval_count / 365, interval_unit = 'year'
WHERE interval_count IS NOT NULL AND interval_count % 365 = 0;
UPDATE health_items SET interval_count = interval_count / 7, interval_unit = 'week'
WHERE interval_count IS NOT NULL AND interval_unit = 'day' AND interval_count % 7 = 0;
