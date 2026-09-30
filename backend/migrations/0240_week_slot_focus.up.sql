-- Focus and library exercise of a planned day (JAN-92). The week plan proposes both; the owner
-- stores them with the day. Deleting an exercise keeps the day.
ALTER TABLE week_slots
    ADD COLUMN focus       text,
    ADD COLUMN exercise_id uuid REFERENCES exercises (id) ON DELETE SET NULL;
