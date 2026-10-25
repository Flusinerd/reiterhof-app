-- Health items: "erledigt" is idempotent per day and can be undone the same day.
--   last_done_on      stable-local day of the last "erledigt"; a second one that day changes nothing
--   last_done_at      the moment of it; the reminder rows it dismissed carry the same time, so the undo
--                     brings back exactly those
--   last_done_by      who did it
--   previous_due_date the due date before that "erledigt", restored by DELETE /health-items/{id}/done
-- A PATCH that moves the due date clears all four (an undo would restore a stale date).
ALTER TABLE health_items
    ADD COLUMN last_done_on      date,
    ADD COLUMN last_done_at      timestamptz,
    ADD COLUMN last_done_by      uuid REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN previous_due_date date;
