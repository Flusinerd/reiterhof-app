-- Blanket window (Deckenzeitraum) per horse: the stable-local time span in which the horse is
-- covered. cover_start is the evening of the blanket day, cover_end the morning after. The
-- recommendation of a horse uses the forecast over its own window. Owners and admins change it
-- in the app; the defaults are the former fixed 18:00 to 12:30.
ALTER TABLE horses
    ADD COLUMN cover_start time NOT NULL DEFAULT '18:00',
    ADD COLUMN cover_end time NOT NULL DEFAULT '12:30';
