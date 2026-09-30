-- Consent for the AI week plan (JAN-89). The training week plan may ask the language model of
-- Mistral AI for a proposal; it sends the horse's training data without names, ids or free
-- text, and only when the owner of the horse has granted this consent.
ALTER TABLE consents DROP CONSTRAINT consents_kind_check;
ALTER TABLE consents ADD CONSTRAINT consents_kind_check
    CHECK (kind IN ('location_geofence', 'location_tracking', 'maps', 'presence_sharing', 'photos', 'push', 'ai_training'));
