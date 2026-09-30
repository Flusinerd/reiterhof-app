-- Helpers choose their own reminder time; the creator no longer sets one.
ALTER TABLE request_assignees ADD COLUMN remind_at timestamptz;

UPDATE request_assignees a
SET remind_at = r.remind_helper_at
FROM requests r
WHERE r.id = a.request_id AND r.remind_helper_at IS NOT NULL;

ALTER TABLE requests DROP COLUMN remind_helper_at;
