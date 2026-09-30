-- Requests (M4): recurring series and duplicate-free helper reminders.

-- Occurrences of a recurring request share series_id; the first request of a series
-- (series_id = id) is the template the daily job copies from.
ALTER TABLE requests ADD COLUMN series_id uuid;
CREATE INDEX requests_series_idx ON requests (series_id);
CREATE UNIQUE INDEX requests_series_date_key ON requests (series_id, date) WHERE series_id IS NOT NULL;

-- One helper reminder per request and user; the job claims it by inserting the row.
CREATE UNIQUE INDEX reminders_request_helper_key ON reminders (source_id, user_id, kind)
    WHERE source_table = 'requests';
