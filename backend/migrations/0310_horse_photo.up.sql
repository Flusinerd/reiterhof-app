-- Optional photo of a horse (Abendlicht horse page). photo_path is the path of a file of the
-- same stable from POST /api/v1/files ("<stable_id>/<random>.<ext>"); NULL means no photo, the
-- app then shows the horse's color with its initial. Every member of the stable may read the
-- file (internal/files follows horses.photo_path like blankets.photo_path).
ALTER TABLE horses ADD COLUMN photo_path text;
