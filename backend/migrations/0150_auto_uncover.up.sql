-- Automatic uncovering (JAN-78).
-- On the days in auto_uncover_days the farm staff take the blankets off at the same time
-- after the horses came in from the paddock. From auto_uncover_time on (stable-local) the
-- blanket-auto-uncover job records "uncovered" for every horse that is still covered from
-- the past night. NULL auto_uncover_time = off. auto_uncover_days are ISO weekdays
-- (1 = Monday .. 7 = Sunday). blanket_states.automatic marks the rows the job wrote
-- (changed_by is NULL for them).
ALTER TABLE stables
    ADD COLUMN auto_uncover_time time,
    ADD COLUMN auto_uncover_days smallint[] NOT NULL DEFAULT '{1,2,3,4,5}'
        CHECK (auto_uncover_days <@ '{1,2,3,4,5,6,7}'::smallint[]);

ALTER TABLE blanket_states
    ADD COLUMN automatic boolean NOT NULL DEFAULT false;
